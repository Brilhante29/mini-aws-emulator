package conformance

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Brilhante29/mini-aws-emulator/internal/cloud"
	"github.com/Brilhante29/mini-aws-emulator/internal/report"
)

type Suite struct {
	ports  cloud.Ports
	prefix string
}

func New(ports cloud.Ports, prefix string) Suite {
	return Suite{ports: ports, prefix: prefix}
}

func (s Suite) Run(ctx context.Context) []report.Check {
	checks := make([]report.Check, 0, 18)
	checks = append(checks, s.runS3(ctx)...)
	checks = append(checks, s.runSQS(ctx)...)
	checks = append(checks, s.runDynamoDB(ctx)...)
	return checks
}

func (s Suite) runS3(ctx context.Context) []report.Check {
	bucket := s.prefix + "-s3"
	key := "proof.txt"
	payload := []byte("portable cloud conformance")
	checks := make([]report.Check, 0, 6)
	checks = append(checks, measure("s3", "create_bucket", func() error {
		return s.ports.Objects.CreateBucket(ctx, bucket)
	}))
	checks = append(checks, measure("s3", "put_object", func() error {
		return s.ports.Objects.PutObject(ctx, bucket, key, payload)
	}))
	checks = append(checks, measure("s3", "get_object_body", func() error {
		body, err := s.ports.Objects.GetObject(ctx, bucket, key)
		if err != nil {
			return err
		}
		if !bytes.Equal(body, payload) {
			return fmt.Errorf("object body mismatch")
		}
		return nil
	}))
	checks = append(checks, measure("s3", "list_objects", func() error {
		keys, err := s.ports.Objects.ListObjects(ctx, bucket)
		if err != nil {
			return err
		}
		if !contains(keys, key) {
			return fmt.Errorf("listed objects do not contain %q", key)
		}
		return nil
	}))
	checks = append(checks, measure("s3", "delete_object", func() error {
		return s.ports.Objects.DeleteObject(ctx, bucket, key)
	}))
	checks = append(checks, measure("s3", "delete_bucket", func() error {
		return s.ports.Objects.DeleteBucket(ctx, bucket)
	}))
	return checks
}

func (s Suite) runSQS(ctx context.Context) []report.Check {
	name := s.prefix + "-sqs"
	body := "portable queue conformance"
	var queueURL string
	var receiptHandle string
	checks := make([]report.Check, 0, 6)
	checks = append(checks, measure("sqs", "create_queue", func() error {
		var err error
		queueURL, err = s.ports.Queue.CreateQueue(ctx, name)
		return err
	}))
	checks = append(checks, measure("sqs", "send_message", func() error {
		messageID, err := s.ports.Queue.SendMessage(ctx, queueURL, body)
		if err != nil {
			return err
		}
		if messageID == "" {
			return fmt.Errorf("empty message ID")
		}
		return nil
	}))
	checks = append(checks, measure("sqs", "receive_message_body", func() error {
		message, err := s.ports.Queue.ReceiveMessage(ctx, queueURL)
		if err != nil {
			return err
		}
		if message.Body != body {
			return fmt.Errorf("message body mismatch")
		}
		receiptHandle = message.ReceiptHandle
		return nil
	}))
	checks = append(checks, measure("sqs", "delete_message", func() error {
		return s.ports.Queue.DeleteMessage(ctx, queueURL, receiptHandle)
	}))
	checks = append(checks, measure("sqs", "list_queues", func() error {
		queues, err := s.ports.Queue.ListQueues(ctx, name)
		if err != nil {
			return err
		}
		for _, queue := range queues {
			if strings.HasSuffix(queue, "/"+name) || queue == queueURL {
				return nil
			}
		}
		return fmt.Errorf("queue list does not contain %q", name)
	}))
	checks = append(checks, measure("sqs", "delete_queue", func() error {
		return s.ports.Queue.DeleteQueue(ctx, queueURL)
	}))
	return checks
}

func (s Suite) runDynamoDB(ctx context.Context) []report.Check {
	table := s.prefix + "-ddb"
	id := "proof"
	value := "portable key-value conformance"
	checks := make([]report.Check, 0, 6)
	checks = append(checks, measure("dynamodb", "create_table", func() error {
		return s.ports.Values.CreateTable(ctx, table)
	}))
	checks = append(checks, measure("dynamodb", "put_item", func() error {
		return s.ports.Values.PutItem(ctx, table, id, value)
	}))
	checks = append(checks, measure("dynamodb", "get_item_value", func() error {
		actual, found, err := s.ports.Values.GetItem(ctx, table, id)
		if err != nil {
			return err
		}
		if !found || actual != value {
			return fmt.Errorf("item value mismatch")
		}
		return nil
	}))
	checks = append(checks, measure("dynamodb", "delete_item", func() error {
		return s.ports.Values.DeleteItem(ctx, table, id)
	}))
	checks = append(checks, measure("dynamodb", "confirm_item_missing", func() error {
		_, found, err := s.ports.Values.GetItem(ctx, table, id)
		if err != nil {
			return err
		}
		if found {
			return fmt.Errorf("deleted item is still visible")
		}
		return nil
	}))
	checks = append(checks, measure("dynamodb", "delete_table", func() error {
		return s.ports.Values.DeleteTable(ctx, table)
	}))
	return checks
}

func measure(service, name string, operation func() error) report.Check {
	started := time.Now()
	err := operation()
	check := report.Check{
		Service:    service,
		Name:       name,
		Passed:     err == nil,
		DurationMS: report.Round3(float64(time.Since(started).Microseconds()) / 1000),
	}
	if err != nil {
		check.Error = err.Error()
	}
	return check
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
