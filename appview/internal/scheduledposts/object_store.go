package scheduledposts

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

var ErrPrivateObjectStoreUnavailable = errors.New("private object store unavailable")

type PrivateObjectStore interface {
	Put(context.Context, string, io.Reader, int64, string) error
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}

type S3ObjectStoreConfig struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	Environment     string
}

type S3ObjectStore struct {
	client *s3.Client
	bucket string
}

func NewS3ObjectStore(ctx context.Context, settings S3ObjectStoreConfig) (*S3ObjectStore, error) {
	endpoint, err := url.Parse(settings.Endpoint)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		return nil, errors.New("private object store endpoint is invalid")
	}
	environment := strings.ToLower(strings.TrimSpace(settings.Environment))
	if endpoint.Scheme != "https" && environment != "dev" && environment != "test" {
		return nil, errors.New("private object store requires HTTPS outside dev and test")
	}
	if settings.Region == "" || settings.Bucket == "" || strings.Contains(settings.Bucket, "/") ||
		settings.AccessKeyID == "" || settings.SecretAccessKey == "" {
		return nil, errors.New("private object store configuration is incomplete")
	}
	awsConfig, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(settings.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			settings.AccessKeyID,
			settings.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, ErrPrivateObjectStoreUnavailable
	}
	client := s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint.String())
		options.UsePathStyle = true
	})
	return &S3ObjectStore{client: client, bucket: settings.Bucket}, nil
}

func (s *S3ObjectStore) Check(ctx context.Context) error {
	if s == nil || s.client == nil || s.bucket == "" {
		return ErrPrivateObjectStoreUnavailable
	}
	if _, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	}); err != nil {
		return ErrPrivateObjectStoreUnavailable
	}
	return nil
}

func (s *S3ObjectStore) Put(
	ctx context.Context,
	objectKey string,
	body io.Reader,
	size int64,
	contentType string,
) error {
	if s == nil || s.client == nil || !validPrivateObjectKey(objectKey) ||
		body == nil || size < 1 || contentType == "" {
		return errors.New("invalid private object write")
	}
	if _, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(objectKey),
		Body:          body,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	}); err != nil {
		return ErrPrivateObjectStoreUnavailable
	}
	return nil
}

func (s *S3ObjectStore) Open(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	if s == nil || s.client == nil || !validPrivateObjectKey(objectKey) {
		return nil, errors.New("invalid private object read")
	}
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, ErrPrivateObjectStoreUnavailable
	}
	return result.Body, nil
}

func (s *S3ObjectStore) Delete(ctx context.Context, objectKey string) error {
	if s == nil || s.client == nil || !validPrivateObjectKey(objectKey) {
		return errors.New("invalid private object delete")
	}
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectKey),
	}); err != nil {
		return ErrPrivateObjectStoreUnavailable
	}
	return nil
}

func validPrivateObjectKey(objectKey string) bool {
	const prefix = "scheduled-media/"
	if !strings.HasPrefix(objectKey, prefix) {
		return false
	}
	_, err := uuid.Parse(strings.TrimPrefix(objectKey, prefix))
	return err == nil
}
