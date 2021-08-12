package gatewayconfigurer

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
)

type GoogleBucketReader struct {
	BucketName       string
	BucketObjectName string
}

func (g GoogleBucketReader) ReadBucketObject(ctx context.Context) (io.Reader, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("instantiating storage client: %w", err)
	}

	reader, err := client.Bucket(g.BucketName).Object(g.BucketObjectName).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating google bucket reader: %w", err)
	}

	return reader, nil
}
