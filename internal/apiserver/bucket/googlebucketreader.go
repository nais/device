package bucket

import (
	"context"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
)

type Object interface {
	LastUpdated() time.Time
	Reader() io.Reader
	Close() error
}

type Client interface {
	Open(ctx context.Context) (Object, error)
}

type googleBucketReader struct {
	bucketName string
	objectName string
}

type googleObject struct {
	attrs  *storage.ObjectAttrs
	reader *storage.Reader
}

func NewClient(bucketName, objectName string) Client {
	return &googleBucketReader{
		bucketName: bucketName,
		objectName: objectName,
	}
}

func (g *googleBucketReader) Open(ctx context.Context) (Object, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("instantiating storage client: %w", err)
	}

	object := client.Bucket(g.bucketName).Object(g.objectName)

	attr, err := object.Attrs(ctx)
	if err != nil {
		return nil, err
	}

	reader, err := object.NewReader(ctx)
	if err != nil {
		return nil, err
	}

	return &googleObject{
		attrs:  attr,
		reader: reader,
	}, nil
}

func (o *googleObject) LastUpdated() time.Time {
	return o.attrs.Updated
}

func (o *googleObject) Reader() io.Reader {
	return o.reader
}

func (o *googleObject) Close() error {
	return o.reader.Close()
}
