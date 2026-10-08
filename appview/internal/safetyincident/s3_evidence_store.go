package safetyincident

import (
	"bytes"
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

type S3EvidenceStoreConfig struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	Environment     string
}

type S3EvidenceStore struct {
	client *s3.Client
	bucket string
}

func NewS3EvidenceStore(ctx context.Context, settings S3EvidenceStoreConfig) (*S3EvidenceStore, error) {
	endpoint, err := url.Parse(settings.Endpoint)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		return nil, errors.New("restricted evidence store endpoint is invalid")
	}
	environment := strings.ToLower(strings.TrimSpace(settings.Environment))
	if endpoint.Scheme != "https" && environment != "dev" && environment != "test" {
		return nil, errors.New("restricted evidence store requires HTTPS outside dev and test")
	}
	if settings.Region == "" || settings.Bucket == "" || strings.Contains(settings.Bucket, "/") ||
		settings.AccessKeyID == "" || settings.SecretAccessKey == "" {
		return nil, errors.New("restricted evidence store configuration is incomplete")
	}
	awsConfig, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(settings.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			settings.AccessKeyID, settings.SecretAccessKey, "",
		)),
	)
	if err != nil {
		return nil, ErrEvidenceStoreBoundary
	}
	client := s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint.String())
		options.UsePathStyle = true
	})
	return &S3EvidenceStore{client: client, bucket: settings.Bucket}, nil
}

func (store *S3EvidenceStore) Check(ctx context.Context) error {
	if store == nil || store.client == nil || store.bucket == "" {
		return ErrEvidenceStoreBoundary
	}
	if _, err := store.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(store.bucket)}); err != nil {
		return ErrEvidenceStoreBoundary
	}
	return nil
}

func (store *S3EvidenceStore) Put(ctx context.Context, object RestrictedObject) (ObjectRef, error) {
	if store == nil || store.client == nil || !validEvidenceObjectKey(object.Key) || len(object.Bytes) == 0 {
		return ObjectRef{}, ErrInvalidEvidence
	}
	if _, err := store.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(store.bucket), Key: aws.String(object.Key), Body: bytes.NewReader(object.Bytes),
		ContentLength: aws.Int64(int64(len(object.Bytes))), ContentType: aws.String("application/octet-stream"),
	}); err != nil {
		return ObjectRef{}, ErrEvidenceStoreBoundary
	}
	return ObjectRef{Key: object.Key}, nil
}

func (store *S3EvidenceStore) Delete(ctx context.Context, ref ObjectRef) error {
	if store == nil || store.client == nil || !validEvidenceObjectKey(ref.Key) {
		return ErrInvalidEvidence
	}
	if _, err := store.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(store.bucket), Key: aws.String(ref.Key),
	}); err != nil {
		return ErrEvidenceStoreBoundary
	}
	return nil
}

func (store *S3EvidenceStore) open(ctx context.Context, ref ObjectRef) (io.ReadCloser, error) {
	if store == nil || store.client == nil || !validEvidenceObjectKey(ref.Key) {
		return nil, ErrInvalidEvidence
	}
	result, err := store.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(store.bucket), Key: aws.String(ref.Key),
	})
	if err != nil {
		return nil, ErrEvidenceStoreBoundary
	}
	return result.Body, nil
}

func validEvidenceObjectKey(key string) bool {
	parts := strings.Split(key, "/")
	if len(parts) != 2 {
		return false
	}
	for _, part := range parts {
		parsed, err := uuid.Parse(part)
		if err != nil || parsed == uuid.Nil || parsed.String() != part {
			return false
		}
	}
	return true
}

var _ EvidenceStore = (*S3EvidenceStore)(nil)
var _ restrictedObjectReader = (*S3EvidenceStore)(nil)
