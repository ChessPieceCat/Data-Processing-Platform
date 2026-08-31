package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	BaseDir string
}

func NewLocalStorage(baseDir string) *LocalStorage {
	return &LocalStorage{
		BaseDir: baseDir,
	}
}

func (s *LocalStorage) path(key string) string {
	return filepath.Join(
		s.BaseDir,
		filepath.FromSlash(key),
	)
}

func (s *LocalStorage) Put(
	ctx context.Context,
	key string,
	r io.Reader,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	path := s.path(key)

	if err := os.MkdirAll(
		filepath.Dir(path),
		0755,
	); err != nil {
		return fmt.Errorf(
			"failed to create storage directory: %w",
			err,
		)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf(
			"failed to create storage object: %w",
			err,
		)
	}

	defer file.Close()

	if _, err := io.Copy(file, r); err != nil {
		return fmt.Errorf(
			"failed to write storage object: %w",
			err,
		)
	}

	return nil
}

func (s *LocalStorage) Get(
	ctx context.Context,
	key string,
) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	file, err := os.Open(
		s.path(key),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to open storage object: %w",
			err,
		)
	}

	return file, nil
}

func (s *LocalStorage) Delete(
	ctx context.Context,
	key string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := os.Remove(
		s.path(key),
	); err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf(
			"failed to delete storage object: %w",
			err,
		)
	}

	return nil
}

func (s *LocalStorage) Exists(
	ctx context.Context,
	key string,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	_, err := os.Stat(
		s.path(key),
	)

	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, fmt.Errorf(
		"failed to check storage object: %w",
		err,
	)
}

func (s *LocalStorage) DeletePrefix(
	ctx context.Context,
	prefix string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := os.RemoveAll(
		s.path(prefix),
	); err != nil {
		return fmt.Errorf(
			"failed to delete storage prefix %q: %w",
			prefix,
			err,
		)
	}

	return nil
}
