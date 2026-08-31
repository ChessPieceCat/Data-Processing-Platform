package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Storage struct {
	Client *s3.Client
	Bucket string
}

func NewS3Storage(
	client *s3.Client,
	bucket string,
) *S3Storage {
	return &S3Storage{
		Client: client,
		Bucket: bucket,
	}
}

func (s *S3Storage) Put(
	ctx context.Context,
	key string,
	r io.Reader,
) error {
	_, err := s.Client.PutObject(
		ctx,
		&s3.PutObjectInput{
			Bucket: aws.String(s.Bucket),
			Key:    aws.String(key),
			Body:   r,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"failed to upload object %q: %w",
			key,
			err,
		)
	}

	return nil
}

func (s *S3Storage) Get(
	ctx context.Context,
	key string,
) (io.ReadCloser, error) {
	result, err := s.Client.GetObject(
		ctx,
		&s3.GetObjectInput{
			Bucket: aws.String(s.Bucket),
			Key:    aws.String(key),
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to download object %q: %w",
			key,
			err,
		)
	}

	return result.Body, nil
}

func (s *S3Storage) Delete(
	ctx context.Context,
	key string,
) error {
	_, err := s.Client.DeleteObject(
		ctx,
		&s3.DeleteObjectInput{
			Bucket: aws.String(s.Bucket),
			Key:    aws.String(key),
		},
	)
	if err != nil {
		return fmt.Errorf(
			"failed to delete object %q: %w",
			key,
			err,
		)
	}

	return nil
}

func (s *S3Storage) Exists(
	ctx context.Context,
	key string,
) (bool, error) {
	_, err := s.Client.HeadObject(
		ctx,
		&s3.HeadObjectInput{
			Bucket: aws.String(s.Bucket),
			Key:    aws.String(key),
		},
	)
	if err == nil {
		return true, nil
	}

	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return false, nil
	}

	return false, fmt.Errorf(
		"failed to check object %q: %w",
		key,
		err,
	)
}

func (s *S3Storage) DeletePrefix(
	ctx context.Context,
	prefix string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var continuationToken *string

	for {
		result, err := s.Client.ListObjectsV2(
			ctx,
			&s3.ListObjectsV2Input{
				Bucket:            aws.String(s.Bucket),
				Prefix:            aws.String(prefix),
				ContinuationToken: continuationToken,
			},
		)
		if err != nil {
			return fmt.Errorf(
				"failed to list objects under prefix %q: %w",
				prefix,
				err,
			)
		}

		if len(result.Contents) > 0 {
			objects := make(
				[]s3types.ObjectIdentifier,
				0,
				len(result.Contents),
			)

			for _, object := range result.Contents {
				if object.Key != nil {
					objects = append(
						objects,
						s3types.ObjectIdentifier{
							Key: object.Key,
						},
					)
				}
			}

			if len(objects) > 0 {
				if _, err := s.Client.DeleteObjects(
					ctx,
					&s3.DeleteObjectsInput{
						Bucket: aws.String(s.Bucket),
						Delete: &s3types.Delete{
							Objects: objects,
						},
					},
				); err != nil {
					return fmt.Errorf(
						"failed to delete objects under prefix %q: %w",
						prefix,
						err,
					)
				}
			}
		}

		if result.IsTruncated == nil || !*result.IsTruncated {
			break
		}

		continuationToken = result.NextContinuationToken
	}

	return nil
}
