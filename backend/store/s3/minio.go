package s3

import (
	"context"
	"fmt"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/chaitin/panda-wiki/config"
)

type MinioClient struct {
	*minio.Client
	config *config.Config
}

func NewMinioClient(config *config.Config) (*MinioClient, error) {
	s3cfg := config.S3
	endpoint := strings.TrimPrefix(strings.TrimPrefix(s3cfg.Endpoint, "https://"), "http://")

	opts := &minio.Options{
		Creds:        credentials.NewStaticV4(s3cfg.AccessKey, s3cfg.SecretKey, ""),
		Secure:       s3cfg.UseSecure(),
		BucketLookup: minio.BucketLookupDNS,
	}
	if s3cfg.Region != "" {
		opts.Region = s3cfg.Region
	}

	minioClient, err := minio.New(endpoint, opts)
	if err != nil {
		return nil, err
	}

	bucket := s3cfg.BucketName()
	exists, err := minioClient.BucketExists(context.Background(), bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket %q: %w", bucket, err)
	}
	if !exists {
		return nil, fmt.Errorf("OSS bucket %q not found: create it in cloud console and set public read if needed", bucket)
	}

	return &MinioClient{Client: minioClient, config: config}, nil
}

func (c *MinioClient) Bucket() string {
	return c.config.S3.BucketName()
}

func (c *MinioClient) PublicObjectURL(objectKey string) string {
	return c.config.S3.PublicObjectURL(objectKey)
}
