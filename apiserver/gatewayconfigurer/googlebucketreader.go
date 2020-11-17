package gatewayconfigurer

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"io"
)

type GoogleBucketReader struct {
	BucketName       string
	BucketObjectName string
}

func (g GoogleBucketReader) ReadBucketObject() (io.Reader, error) {
	client := storage.Client{}
	reader, err := client.Bucket(g.BucketName).Object(g.BucketObjectName).NewReader(context.Background())
	if err != nil {
		return nil, fmt.Errorf("creating google bucket reader: %w", err)
	}

	return reader, nil
}
