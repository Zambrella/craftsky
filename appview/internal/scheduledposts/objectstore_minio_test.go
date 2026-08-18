package scheduledposts

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

func TestS3ObjectStoreAgainstMinIO(t *testing.T) {
	endpoint := os.Getenv("TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("TEST_S3_ENDPOINT is not configured")
	}

	settings := S3ObjectStoreConfig{
		Endpoint:        endpoint,
		Region:          requiredTestEnv(t, "TEST_S3_REGION"),
		Bucket:          requiredTestEnv(t, "TEST_S3_BUCKET"),
		AccessKeyID:     requiredTestEnv(t, "TEST_S3_ACCESS_KEY_ID"),
		SecretAccessKey: requiredTestEnv(t, "TEST_S3_SECRET_ACCESS_KEY"),
		Environment:     "test",
	}
	store, err := NewS3ObjectStore(context.Background(), settings)
	if err != nil {
		t.Fatalf("construct MinIO object store: %v", err)
	}
	if err := store.Check(context.Background()); err != nil {
		t.Fatalf("check private MinIO bucket: %v", err)
	}

	objectKey, _, err := NewGenerationObjectKey(
		syntax.DID("did:plc:minio-fixture"),
		1,
		uuid.New(),
	)
	if err != nil {
		t.Fatalf("derive MinIO object key: %v", err)
	}
	payload := []byte("minio-private-object-checksum-fixture")
	t.Cleanup(func() {
		_ = store.Delete(context.Background(), objectKey)
	})

	if err := store.Put(
		context.Background(),
		objectKey,
		bytes.NewReader(payload),
		int64(len(payload)),
		"image/jpeg",
	); err != nil {
		t.Fatalf("put private MinIO object: %v", err)
	}
	if exists, err := store.Exists(context.Background(), objectKey); err != nil || !exists {
		t.Fatalf("private MinIO object exists=%v error=%v after Put", exists, err)
	}
	opened, err := store.Open(context.Background(), objectKey)
	if err != nil {
		t.Fatalf("open private MinIO object: %v", err)
	}
	got, readErr := io.ReadAll(opened)
	closeErr := opened.Close()
	if readErr != nil || closeErr != nil {
		t.Fatalf("read private MinIO object: read=%v close=%v", readErr, closeErr)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("MinIO object=%q, want %q", got, payload)
	}

	anonymousResponse, err := http.Get(endpoint + "/" + settings.Bucket + "/" + objectKey)
	if err != nil {
		t.Fatalf("anonymous MinIO request: %v", err)
	}
	_ = anonymousResponse.Body.Close()
	if anonymousResponse.StatusCode != http.StatusForbidden {
		t.Fatalf("anonymous status=%d, want %d", anonymousResponse.StatusCode, http.StatusForbidden)
	}

	if err := store.Delete(context.Background(), objectKey); err != nil {
		t.Fatalf("delete private MinIO object: %v", err)
	}
	if exists, err := store.Exists(context.Background(), objectKey); err != nil || exists {
		t.Fatalf("private MinIO object exists=%v error=%v after Delete", exists, err)
	}
	if err := store.Delete(context.Background(), objectKey); err != nil {
		t.Fatalf("repeat private MinIO delete: %v", err)
	}
	if _, err := store.Open(context.Background(), objectKey); err == nil {
		t.Fatal("deleted private MinIO object remained readable")
	}
	if err := store.Put(
		context.Background(),
		"foreign-prefix/"+uuid.NewString(),
		bytes.NewReader(payload),
		int64(len(payload)),
		"image/jpeg",
	); err == nil {
		t.Fatal("cross-prefix object write succeeded")
	}
}

func requiredTestEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("%s is not configured", name)
	}
	return value
}
