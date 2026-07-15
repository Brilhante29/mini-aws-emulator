package cloud

import "context"

type Message struct {
	Body          string
	ReceiptHandle string
}

type ObjectStorage interface {
	CreateBucket(context.Context, string) error
	PutObject(context.Context, string, string, []byte) error
	GetObject(context.Context, string, string) ([]byte, error)
	ListObjects(context.Context, string) ([]string, error)
	DeleteObject(context.Context, string, string) error
	DeleteBucket(context.Context, string) error
}

type Queue interface {
	CreateQueue(context.Context, string) (string, error)
	SendMessage(context.Context, string, string) (string, error)
	ReceiveMessage(context.Context, string) (Message, error)
	DeleteMessage(context.Context, string, string) error
	ListQueues(context.Context, string) ([]string, error)
	DeleteQueue(context.Context, string) error
}

type KeyValueStore interface {
	CreateTable(context.Context, string) error
	PutItem(context.Context, string, string, string) error
	GetItem(context.Context, string, string) (string, bool, error)
	DeleteItem(context.Context, string, string) error
	DeleteTable(context.Context, string) error
}

type Ports struct {
	Objects ObjectStorage
	Queue   Queue
	Values  KeyValueStore
}
