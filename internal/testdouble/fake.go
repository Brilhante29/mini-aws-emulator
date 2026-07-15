package testdouble

import (
	"context"
	"fmt"
	"strings"

	"github.com/Brilhante29/mini-aws-emulator/internal/cloud"
)

type Fake struct {
	FailOperation string
	objects       map[string][]byte
	items         map[string]string
	messages      []string
}

func New() *Fake {
	return &Fake{objects: map[string][]byte{}, items: map[string]string{}}
}

func (f *Fake) Ports() cloud.Ports { return cloud.Ports{Objects: f, Queue: f, Values: f} }

func (f *Fake) fail(operation string) error {
	if f.FailOperation == operation {
		return fmt.Errorf("forced %s failure", operation)
	}
	return nil
}

func (f *Fake) CreateBucket(context.Context, string) error { return f.fail("create_bucket") }
func (f *Fake) PutObject(_ context.Context, _, key string, body []byte) error {
	if err := f.fail("put_object"); err != nil {
		return err
	}
	f.objects[key] = append([]byte(nil), body...)
	return nil
}
func (f *Fake) GetObject(_ context.Context, _, key string) ([]byte, error) {
	if err := f.fail("get_object"); err != nil {
		return nil, err
	}
	return append([]byte(nil), f.objects[key]...), nil
}
func (f *Fake) ListObjects(context.Context, string) ([]string, error) {
	if err := f.fail("list_objects"); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(f.objects))
	for key := range f.objects {
		keys = append(keys, key)
	}
	return keys, nil
}
func (f *Fake) DeleteObject(_ context.Context, _, key string) error {
	if err := f.fail("delete_object"); err != nil {
		return err
	}
	delete(f.objects, key)
	return nil
}
func (f *Fake) DeleteBucket(context.Context, string) error { return f.fail("delete_bucket") }

func (f *Fake) CreateQueue(_ context.Context, name string) (string, error) {
	if err := f.fail("create_queue"); err != nil {
		return "", err
	}
	return "http://queue/" + name, nil
}
func (f *Fake) SendMessage(_ context.Context, _ string, body string) (string, error) {
	if err := f.fail("send_message"); err != nil {
		return "", err
	}
	f.messages = append(f.messages, body)
	return fmt.Sprintf("message-%d", len(f.messages)), nil
}
func (f *Fake) ReceiveMessage(context.Context, string) (cloud.Message, error) {
	if err := f.fail("receive_message"); err != nil {
		return cloud.Message{}, err
	}
	if len(f.messages) == 0 {
		return cloud.Message{}, fmt.Errorf("no messages")
	}
	body := f.messages[0]
	f.messages = f.messages[1:]
	return cloud.Message{Body: body, ReceiptHandle: "receipt"}, nil
}
func (f *Fake) DeleteMessage(context.Context, string, string) error { return f.fail("delete_message") }
func (f *Fake) ListQueues(_ context.Context, prefix string) ([]string, error) {
	if err := f.fail("list_queues"); err != nil {
		return nil, err
	}
	return []string{"http://queue/" + strings.TrimSuffix(prefix, "/")}, nil
}
func (f *Fake) DeleteQueue(context.Context, string) error { return f.fail("delete_queue") }

func (f *Fake) CreateTable(context.Context, string) error { return f.fail("create_table") }
func (f *Fake) PutItem(_ context.Context, _, id, value string) error {
	if err := f.fail("put_item"); err != nil {
		return err
	}
	f.items[id] = value
	return nil
}
func (f *Fake) GetItem(_ context.Context, _, id string) (string, bool, error) {
	if err := f.fail("get_item"); err != nil {
		return "", false, err
	}
	value, found := f.items[id]
	return value, found, nil
}
func (f *Fake) DeleteItem(_ context.Context, _, id string) error {
	if err := f.fail("delete_item"); err != nil {
		return err
	}
	delete(f.items, id)
	return nil
}
func (f *Fake) DeleteTable(context.Context, string) error { return f.fail("delete_table") }
