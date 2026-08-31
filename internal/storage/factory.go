package storage

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	BackendLocal = "local"
	BackendS3    = "s3"

	DefaultLocalStorageDirectory = "uploads"
)

func NewFromEnvironment(ctx context.Context) (Storage, error) {
	backend := os.Getenv("STORAGE_BACKEND")

	if backend == "" {
		backend = BackendLocal
	}

	switch backend {
	case BackendLocal:
		return NewLocalStorage(
			DefaultLocalStorageDirectory,
		), nil

	case BackendS3:
		bucket := os.Getenv("S3_BUCKET")
		if bucket == "" {
			return nil, fmt.Errorf(
				"S3_BUCKET must be set when STORAGE_BACKEND=s3",
			)
		}

		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to load AWS configuration: %w",
				err,
			)
		}

		client := s3.NewFromConfig(cfg)

		return NewS3Storage(
			client,
			bucket,
		), nil

	default:
		return nil, fmt.Errorf(
			"unsupported storage backend %q",
			backend,
		)
	}
}
