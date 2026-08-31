package storage

import (
	"context"
	"testing"
)

func TestNewFromEnvironmentDefaultsToLocal(t *testing.T) {
	t.Setenv("STORAGE_BACKEND", "")
	t.Setenv("S3_BUCKET", "")

	store, err := NewFromEnvironment(context.Background())
	if err != nil {
		t.Fatalf("NewFromEnvironment failed: %v", err)
	}

	localStore, ok := store.(*LocalStorage)
	if !ok {
		t.Fatalf("expected *LocalStorage, got %T", store)
	}

	if localStore.BaseDir != DefaultLocalStorageDirectory {
		t.Fatalf(
			"expected base directory %q, got %q",
			DefaultLocalStorageDirectory,
			localStore.BaseDir,
		)
	}
}

func TestNewFromEnvironmentLocal(t *testing.T) {
	t.Setenv("STORAGE_BACKEND", BackendLocal)
	t.Setenv("S3_BUCKET", "")

	store, err := NewFromEnvironment(context.Background())
	if err != nil {
		t.Fatalf("NewFromEnvironment failed: %v", err)
	}

	if _, ok := store.(*LocalStorage); !ok {
		t.Fatalf("expected *LocalStorage, got %T", store)
	}
}

func TestNewFromEnvironmentUnsupportedBackend(t *testing.T) {
	t.Setenv("STORAGE_BACKEND", "unsupported")

	_, err := NewFromEnvironment(context.Background())
	if err == nil {
		t.Fatal("expected error for unsupported storage backend")
	}
}

func TestNewFromEnvironmentS3RequiresBucket(t *testing.T) {
	t.Setenv("STORAGE_BACKEND", BackendS3)
	t.Setenv("S3_BUCKET", "")

	_, err := NewFromEnvironment(context.Background())
	if err == nil {
		t.Fatal("expected error when S3_BUCKET is not set")
	}
}

func TestNewFromEnvironmentS3(t *testing.T) {
	t.Setenv("STORAGE_BACKEND", BackendS3)
	t.Setenv("S3_BUCKET", "test-bucket")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	store, err := NewFromEnvironment(context.Background())
	if err != nil {
		t.Fatalf("NewFromEnvironment failed: %v", err)
	}

	s3Store, ok := store.(*S3Storage)
	if !ok {
		t.Fatalf("expected *S3Storage, got %T", store)
	}

	if s3Store.Bucket != "test-bucket" {
		t.Fatalf(
			"expected bucket %q, got %q",
			"test-bucket",
			s3Store.Bucket,
		)
	}
}
