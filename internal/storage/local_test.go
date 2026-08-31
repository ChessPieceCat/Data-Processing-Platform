package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalStoragePutGetExistsDelete(t *testing.T) {
	baseDir := t.TempDir()

	store := NewLocalStorage(baseDir)
	ctx := context.Background()

	key := "jobs/123/input.txt"
	content := "hello storage"

	err := store.Put(
		ctx,
		key,
		strings.NewReader(content),
	)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	exists, err := store.Exists(
		ctx,
		key,
	)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	if !exists {
		t.Fatal("expected object to exist after Put")
	}

	reader, err := store.Get(
		ctx,
		key,
	)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read object: %v", err)
	}

	if string(data) != content {
		t.Fatalf(
			"expected %q, got %q",
			content,
			string(data),
		)
	}

	if err := store.Delete(
		ctx,
		key,
	); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	exists, err = store.Exists(
		ctx,
		key,
	)
	if err != nil {
		t.Fatalf("Exists after Delete failed: %v", err)
	}

	if exists {
		t.Fatal("expected object to not exist after Delete")
	}
}

func TestLocalStorageGetMissingObject(t *testing.T) {
	store := NewLocalStorage(
		t.TempDir(),
	)

	_, err := store.Get(
		context.Background(),
		"jobs/123/missing.txt",
	)
	if err == nil {
		t.Fatal("expected Get to fail for missing object")
	}
}

func TestLocalStorageDeleteMissingObjectIsNoOp(t *testing.T) {
	store := NewLocalStorage(
		t.TempDir(),
	)

	err := store.Delete(
		context.Background(),
		"jobs/123/missing.txt",
	)
	if err != nil {
		t.Fatalf(
			"expected Delete of missing object to succeed, got %v",
			err,
		)
	}
}

func TestLocalStorageExistsMissingObject(t *testing.T) {
	store := NewLocalStorage(
		t.TempDir(),
	)

	exists, err := store.Exists(
		context.Background(),
		"jobs/123/missing.txt",
	)
	if err != nil {
		t.Fatalf(
			"Exists failed: %v",
			err,
		)
	}

	if exists {
		t.Fatal("expected missing object to report false")
	}
}

func TestLocalStoragePutCreatesParentDirectories(t *testing.T) {
	baseDir := t.TempDir()

	store := NewLocalStorage(baseDir)

	err := store.Put(
		context.Background(),
		"jobs/123/nested/input.txt",
		&stringReader{value: "nested"},
	)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	path := filepath.Join(
		baseDir,
		"jobs",
		"123",
		"nested",
		"input.txt",
	)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf(
			"failed to read stored file: %v",
			err,
		)
	}

	if string(data) != "nested" {
		t.Fatalf(
			"expected %q, got %q",
			"nested",
			string(data),
		)
	}
}

func TestLocalStorageRespectsCancelledContext(t *testing.T) {
	store := NewLocalStorage(
		t.TempDir(),
	)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel()

	_, err := store.Get(
		ctx,
		"jobs/123/input.txt",
	)
	if err == nil {
		t.Fatal(
			"expected Get to fail with cancelled context",
		)
	}
}

type stringReader struct {
	value string
	read  bool
}

func (r *stringReader) Read(
	p []byte,
) (int, error) {
	if r.read {
		return 0, io.EOF
	}

	n := copy(
		p,
		r.value,
	)

	r.read = true

	return n, nil
}
