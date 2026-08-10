package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/Brilhante29/mini-aws-emulator/internal/cloud"
)

type Result struct {
	Durations        []time.Duration
	Failed           int
	WarmupOperations int
	Elapsed          time.Duration
}

func Run(ctx context.Context, ports cloud.Ports, prefix string, warmupIterations, measuredIterations int) (Result, error) {
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

	result := Result{
		Durations:        make([]time.Duration, 0, measuredIterations*9),
		WarmupOperations: warmupIterations * 9,
	}
	for iteration := 0; iteration < warmupIterations; iteration++ {
		if err := runIteration(ctx, ports, bucket, queueURL, table, "warmup", iteration, nil); err != nil {
			return result, fmt.Errorf("benchmark warmup iteration %d: %w", iteration, err)
		}
	}

	started := time.Now()
	for iteration := 0; iteration < measuredIterations; iteration++ {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		_ = runIteration(ctx, ports, bucket, queueURL, table, "measured", iteration, &result)
	}
	result.Elapsed = time.Since(started)
	return result, nil
}

func runIteration(ctx context.Context, ports cloud.Ports, bucket, queueURL, table, phase string, iteration int, result *Result) error {
	key := fmt.Sprintf("%s-object-%03d", phase, iteration)
	id := fmt.Sprintf("%s-item-%03d", phase, iteration)
	payload := []byte(fmt.Sprintf("%s-payload-%03d", phase, iteration))
	messageBody := fmt.Sprintf("%s-message-%03d", phase, iteration)

	operations := []struct {
		run func() error
	}{
		{func() error { return ports.Objects.PutObject(ctx, bucket, key, payload) }},
		{func() error {
			body, err := ports.Objects.GetObject(ctx, bucket, key)
			if err != nil {
				return err
			}
			if !bytes.Equal(body, payload) {
				return fmt.Errorf("S3 body mismatch for %s", key)
			}
			return nil
		}},
		{func() error { return ports.Objects.DeleteObject(ctx, bucket, key) }},
		{func() error { return ports.Values.PutItem(ctx, table, id, string(payload)) }},
		{func() error {
			value, found, err := ports.Values.GetItem(ctx, table, id)
			if err != nil {
				return err
			}
			if !found || value != string(payload) {
				return fmt.Errorf("DynamoDB value mismatch for %s", id)
			}
			return nil
		}},
		{func() error { return ports.Values.DeleteItem(ctx, table, id) }},
	}
	for _, operation := range operations {
		if err := execute(result, operation.run); err != nil && result == nil {
			return err
		}
	}

	if err := execute(result, func() error {
		_, sendErr := ports.Queue.SendMessage(ctx, queueURL, messageBody)
		return sendErr
	}); err != nil && result == nil {
		return err
	}
	var receiptHandle string
	if err := execute(result, func() error {
		message, receiveErr := ports.Queue.ReceiveMessage(ctx, queueURL)
		if receiveErr != nil {
			return receiveErr
		}
		if message.Body != messageBody {
			return fmt.Errorf("SQS body mismatch for iteration %d", iteration)
		}
		receiptHandle = message.ReceiptHandle
		return nil
	}); err != nil && result == nil {
		return err
	}
	if err := execute(result, func() error {
		return ports.Queue.DeleteMessage(ctx, queueURL, receiptHandle)
	}); err != nil && result == nil {
		return err
	}
	return nil
}

func execute(result *Result, operation func() error) error {
	started := time.Now()
	err := operation()
	if result != nil {
		duration := time.Since(started)
		result.Durations = append(result.Durations, duration)
		if err != nil {
			result.Failed++
		}
	}
	return err
}
