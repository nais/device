package gatewayconfigurer

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
)

type Bucket interface {
	Object(ctx context.Context) (Object, error)
}

type Object interface {
	Attrs(ctx context.Context) (attrs *storage.ObjectAttrs, err error)
	NewReader(ctx context.Context) (*storage.Reader, error)
}

type GoogleBucketReader struct {
	BucketName       string
	BucketObjectName string
}

func (g GoogleBucketReader) Object(ctx context.Context) (Object, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("instantiating storage client: %w", err)
	}

	return client.Bucket(g.BucketName).Object(g.BucketObjectName), nil
}
