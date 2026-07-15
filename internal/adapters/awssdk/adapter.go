package awssdk

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/Brilhante29/mini-aws-emulator/internal/cloud"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

var ErrNoMessage = errors.New("queue returned no message")

type Options struct {
	Region           string
	Endpoint         string
	UseLocalEndpoint bool
	WaitForResources bool
	diagnostics      *Diagnostics
}

type Adapter struct {
	objects          *s3.Client
	queue            *sqs.Client
	values           *dynamodb.Client
	waitForResources bool
	diagnostics      *Diagnostics
}

func New(ctx context.Context, options Options) (*Adapter, error) {
	loadOptions := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(options.Region)}
	if options.UseLocalEndpoint {
		loadOptions = append(loadOptions, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("load AWS SDK config: %w", err)
	}

	diagnostics := &Diagnostics{}
	cfg.Logger = newDiagnosticLogger(diagnostics)

	adapter := &Adapter{waitForResources: options.WaitForResources, diagnostics: diagnostics}
	adapter.objects = s3.NewFromConfig(cfg, func(clientOptions *s3.Options) {
		if options.UseLocalEndpoint {
			clientOptions.BaseEndpoint = aws.String(options.Endpoint)
			clientOptions.UsePathStyle = true
			clientOptions.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
			clientOptions.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
		}
	})
	adapter.queue = sqs.NewFromConfig(cfg, func(clientOptions *sqs.Options) {
		if options.UseLocalEndpoint {
			clientOptions.BaseEndpoint = aws.String(options.Endpoint)
		}
	})
	adapter.values = dynamodb.NewFromConfig(cfg, func(clientOptions *dynamodb.Options) {
		if options.UseLocalEndpoint {
			clientOptions.BaseEndpoint = aws.String(options.Endpoint)
		}
	})
	return adapter, nil
}

func (a *Adapter) Diagnostics() *Diagnostics {
	return a.diagnostics
}

func (a *Adapter) Ports() cloud.Ports {
	return cloud.Ports{Objects: a, Queue: a, Values: a}
}

func (a *Adapter) CreateBucket(ctx context.Context, bucket string) error {
	_, err := a.objects.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	return err
}

func (a *Adapter) PutObject(ctx context.Context, bucket, key string, body []byte) error {
	_, err := a.objects.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	})
	return err
}

func (a *Adapter) GetObject(ctx context.Context, bucket, key string) ([]byte, error) {
	output, err := a.objects.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		return nil, err
	}
	defer output.Body.Close()
	body, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, fmt.Errorf("read S3 object body: %w", err)
	}
	return body, nil
}

func (a *Adapter) ListObjects(ctx context.Context, bucket string) ([]string, error) {
	output, err := a.objects.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String(bucket)})
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(output.Contents))
	for _, object := range output.Contents {
		if object.Key != nil {
			keys = append(keys, *object.Key)
		}
	}
	return keys, nil
}

func (a *Adapter) DeleteObject(ctx context.Context, bucket, key string) error {
	_, err := a.objects.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	return err
}

func (a *Adapter) DeleteBucket(ctx context.Context, bucket string) error {
	_, err := a.objects.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	return err
}

func (a *Adapter) CreateQueue(ctx context.Context, name string) (string, error) {
	output, err := a.queue.CreateQueue(ctx, &sqs.CreateQueueInput{QueueName: aws.String(name)})
	if err != nil {
		return "", err
	}
	if output.QueueUrl == nil {
		return "", fmt.Errorf("SQS CreateQueue returned no queue URL")
	}
	return *output.QueueUrl, nil
}

func (a *Adapter) SendMessage(ctx context.Context, queueURL, body string) (string, error) {
	output, err := a.queue.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(body),
	})
	if err != nil {
		return "", err
	}
	if output.MessageId == nil {
		return "", fmt.Errorf("SQS SendMessage returned no message ID")
	}
	return *output.MessageId, nil
}

func (a *Adapter) ReceiveMessage(ctx context.Context, queueURL string) (cloud.Message, error) {
	for attempt := 0; attempt < 3; attempt++ {
		output, err := a.queue.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: 1,
			WaitTimeSeconds:     1,
			VisibilityTimeout:   10,
		})
		if err != nil {
			return cloud.Message{}, err
		}
		if len(output.Messages) > 0 {
			message := output.Messages[0]
			if message.Body == nil || message.ReceiptHandle == nil {
				return cloud.Message{}, fmt.Errorf("SQS ReceiveMessage returned incomplete message")
			}
			return cloud.Message{Body: *message.Body, ReceiptHandle: *message.ReceiptHandle}, nil
		}
		select {
		case <-ctx.Done():
			return cloud.Message{}, ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}
	}
	return cloud.Message{}, ErrNoMessage
}

func (a *Adapter) DeleteMessage(ctx context.Context, queueURL, receiptHandle string) error {
	_, err := a.queue.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	return err
}

func (a *Adapter) ListQueues(ctx context.Context, prefix string) ([]string, error) {
	output, err := a.queue.ListQueues(ctx, &sqs.ListQueuesInput{QueueNamePrefix: aws.String(prefix)})
	if err != nil {
		return nil, err
	}
	return output.QueueUrls, nil
}

func (a *Adapter) DeleteQueue(ctx context.Context, queueURL string) error {
	_, err := a.queue.DeleteQueue(ctx, &sqs.DeleteQueueInput{QueueUrl: aws.String(queueURL)})
	return err
}

func (a *Adapter) CreateTable(ctx context.Context, table string) error {
	_, err := a.values.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(table),
		AttributeDefinitions: []ddbtypes.AttributeDefinition{{
			AttributeName: aws.String("id"),
			AttributeType: ddbtypes.ScalarAttributeTypeS,
		}},
		KeySchema: []ddbtypes.KeySchemaElement{{
			AttributeName: aws.String("id"),
			KeyType:       ddbtypes.KeyTypeHash,
		}},
		BillingMode: ddbtypes.BillingModePayPerRequest,
	})
	if err != nil {
		return err
	}
	if a.waitForResources {
		waiter := dynamodb.NewTableExistsWaiter(a.values)
		if err := waiter.Wait(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(table)}, 2*time.Minute); err != nil {
			return fmt.Errorf("wait for DynamoDB table: %w", err)
		}
	}
	return nil
}

func (a *Adapter) PutItem(ctx context.Context, table, id, value string) error {
	_, err := a.values.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item: map[string]ddbtypes.AttributeValue{
			"id":    &ddbtypes.AttributeValueMemberS{Value: id},
			"value": &ddbtypes.AttributeValueMemberS{Value: value},
		},
	})
	return err
}

func (a *Adapter) GetItem(ctx context.Context, table, id string) (string, bool, error) {
	output, err := a.values.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      aws.String(table),
		Key:            map[string]ddbtypes.AttributeValue{"id": &ddbtypes.AttributeValueMemberS{Value: id}},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return "", false, err
	}
	if len(output.Item) == 0 {
		return "", false, nil
	}
	value, ok := output.Item["value"].(*ddbtypes.AttributeValueMemberS)
	if !ok {
		return "", false, fmt.Errorf("DynamoDB item has no string value attribute")
	}
	return value.Value, true, nil
}

func (a *Adapter) DeleteItem(ctx context.Context, table, id string) error {
	_, err := a.values.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(table),
		Key:       map[string]ddbtypes.AttributeValue{"id": &ddbtypes.AttributeValueMemberS{Value: id}},
	})
	return err
}

func (a *Adapter) DeleteTable(ctx context.Context, table string) error {
	_, err := a.values.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String(table)})
	return err
}
