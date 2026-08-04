package audit

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Archiver stores immutable audit blobs. Implementations must not expose Delete.
type Archiver interface {
	Put(ctx context.Context, key string, body []byte) (uri string, err error)
}

type FilesystemArchiver struct {
	root string
}

func NewFilesystemArchiver(root string) *FilesystemArchiver {
	return &FilesystemArchiver{root: root}
}

func (a *FilesystemArchiver) Put(_ context.Context, key string, body []byte) (string, error) {
	path := filepath.Join(a.root, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, body, 0o444); err != nil {
		return "", err
	}
	return "file://" + filepath.ToSlash(path), nil
}

type S3Archiver struct {
	client *s3.Client
	bucket string
}

func NewS3Archiver(endpoint, region, bucket, accessKey, secretKey string) (*S3Archiver, error) {
	if bucket == "" {
		return nil, fmt.Errorf("audit_s3_bucket_required")
	}
	if region == "" {
		region = "auto"
	}
	cfg := aws.Config{
		Region: region,
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}
	})
	return &S3Archiver{client: client, bucket: bucket}, nil
}

func (a *S3Archiver) Put(ctx context.Context, key string, body []byte) (string, error) {
	_, err := a.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(a.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("s3://%s/%s", a.bucket, key), nil
}

// NewArchiverFromEnv picks S3 when AUDIT_S3_BUCKET is set, otherwise filesystem.
func NewArchiverFromEnv(bucket, endpoint, region, accessKey, secretKey, dir string) (Archiver, error) {
	if bucket == "" {
		if dir == "" {
			dir = "./var/audit"
		}
		return NewFilesystemArchiver(dir), nil
	}
	return NewS3Archiver(endpoint, region, bucket, accessKey, secretKey)
}
