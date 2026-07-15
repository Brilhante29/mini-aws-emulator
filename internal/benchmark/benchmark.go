package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/Brilhante29/mini-aws-emulator/internal/cloud"
)

type Result struct {
	Durations []time.Duration
	Failed    int
	Elapsed   time.Duration
}

func Run(ctx context.Context, ports cloud.Ports, prefix string, iterations int) (Result, error) {
	bucket := prefix + "-s3"
	queueName := prefix + "-sqs"
	table := prefix + "-ddb"

	if err := ports.Objects.CreateBucket(ctx, bucket); err != nil {
		return Result{}, fmt.Errorf("benchmark setup S3: %w", err)
	}
	defer ports.Objects.DeleteBucket(context.WithoutCancel(ctx), bucket)

	queueURL, err := ports.Queue.CreateQueue(ctx, queueName)
	if err != nil {
		return Result{}, fmt.Errorf("benchmark setup SQS: %w", err)
	}
	defer ports.Queue.DeleteQueue(context.WithoutCancel(ctx), queueURL)

	if err := ports.Values.CreateTable(ctx, table); err != nil {
		return Result{}, fmt.Errorf("benchmark setup DynamoDB: %w", err)
	}
	defer ports.Values.DeleteTable(context.WithoutCancel(ctx), table)

	result := Result{Durations: make([]time.Duration, 0, iterations*9)}
	started := time.Now()
	for iteration := 0; iteration < iterations; iteration++ {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		key := fmt.Sprintf("object-%03d", iteration)
		id := fmt.Sprintf("item-%03d", iteration)
		payload := []byte(fmt.Sprintf("payload-%03d", iteration))
		messageBody := fmt.Sprintf("message-%03d", iteration)

		record(&result, func() error { return ports.Objects.PutObject(ctx, bucket, key, payload) })
		record(&result, func() error {
			body, getErr := ports.Objects.GetObject(ctx, bucket, key)
			if getErr != nil {
				return getErr
			}
			if !bytes.Equal(body, payload) {
				return fmt.Errorf("S3 body mismatch for %s", key)
			}
			return nil
		})
		record(&result, func() error { return ports.Objects.DeleteObject(ctx, bucket, key) })

		record(&result, func() error { return ports.Values.PutItem(ctx, table, id, string(payload)) })
		record(&result, func() error {
			value, found, getErr := ports.Values.GetItem(ctx, table, id)
			if getErr != nil {
				return getErr
			}
			if !found || value != string(payload) {
				return fmt.Errorf("DynamoDB value mismatch for %s", id)
			}
			return nil
		})
		record(&result, func() error { return ports.Values.DeleteItem(ctx, table, id) })

		record(&result, func() error {
			_, sendErr := ports.Queue.SendMessage(ctx, queueURL, messageBody)
			return sendErr
		})
		var receiptHandle string
		record(&result, func() error {
			message, receiveErr := ports.Queue.ReceiveMessage(ctx, queueURL)
			if receiveErr != nil {
				return receiveErr
			}
			if message.Body != messageBody {
				return fmt.Errorf("SQS body mismatch for iteration %d", iteration)
			}
			receiptHandle = message.ReceiptHandle
			return nil
		})
		record(&result, func() error { return ports.Queue.DeleteMessage(ctx, queueURL, receiptHandle) })
	}
	result.Elapsed = time.Since(started)
	return result, nil
}

func record(result *Result, operation func() error) {
	started := time.Now()
	err := operation()
	result.Durations = append(result.Durations, time.Since(started))
	if err != nil {
		result.Failed++
	}
}
