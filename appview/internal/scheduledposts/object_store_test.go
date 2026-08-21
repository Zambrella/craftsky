package scheduledposts

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestS3ObjectStoreContract(t *testing.T) {
	const (
		bucket    = "private-scheduled-media"
		objectKey = "scheduled-media/v2/7/00000000-0000-5000-8000-000000000501"
	)
	wantBody := []byte("private-image-bytes")
	var mu sync.Mutex
	stored := map[string][]byte{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") == "" {
			http.Error(writer, "missing signature", http.StatusUnauthorized)
			return
		}
		if request.Header.Get("X-Amz-Acl") != "" {
			http.Error(writer, "public ACL forbidden", http.StatusBadRequest)
			return
		}
		if request.URL.Path == "/"+bucket && request.Method == http.MethodHead {
			writer.WriteHeader(http.StatusOK)
			return
		}
		if request.URL.Path != "/"+bucket+"/"+objectKey {
			http.NotFound(writer, request)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		switch request.Method {
		case http.MethodPut:
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatalf("read object PUT: %v", err)
			}
			stored[objectKey] = body
			writer.Header().Set("ETag", `"private-etag"`)
			writer.WriteHeader(http.StatusOK)
		case http.MethodGet:
			body, ok := stored[objectKey]
			if !ok {
				http.NotFound(writer, request)
				return
			}
			writer.Header().Set("Content-Type", "image/jpeg")
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write(body)
		case http.MethodDelete:
			delete(stored, objectKey)
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.Error(writer, "unsupported", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	store, err := NewS3ObjectStore(context.Background(), S3ObjectStoreConfig{
		Endpoint: server.URL, Region: "test-region", Bucket: bucket,
		AccessKeyID: "test-access", SecretAccessKey: "test-secret",
		Environment: "test",
	})
	if err != nil {
		t.Fatalf("construct object store: %v", err)
	}
	if err := store.Check(context.Background()); err != nil {
		t.Fatalf("check private bucket: %v", err)
	}
	if err := store.Put(
		context.Background(),
		objectKey,
		bytes.NewReader(wantBody),
		int64(len(wantBody)),
		"image/jpeg",
	); err != nil {
		t.Fatalf("put private object: %v", err)
	}
	body, err := store.Open(context.Background(), objectKey)
	if err != nil {
		t.Fatalf("open private object: %v", err)
	}
	gotBody, err := io.ReadAll(body)
	closeErr := body.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("read private object: read=%v close=%v", err, closeErr)
	}
	if !bytes.Equal(gotBody, wantBody) {
		t.Fatalf("private object=%q, want %q", gotBody, wantBody)
	}
	if err := store.Delete(context.Background(), objectKey); err != nil {
		t.Fatalf("delete private object: %v", err)
	}

	_, err = NewS3ObjectStore(context.Background(), S3ObjectStoreConfig{
		Endpoint: server.URL, Region: "test-region", Bucket: bucket,
		AccessKeyID: "test-access", SecretAccessKey: "test-secret",
		Environment: "production",
	})
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("production HTTP endpoint error=%v", err)
	}
}

func TestPrivateObjectKeyValidationRequiresCanonicalGenerationIdentity(t *testing.T) {
	t.Parallel()

	for _, objectKey := range []string{
		"scheduled-media/v2/1/00000000-0000-5000-8000-000000000501",
		"scheduled-media/v2/9223372036854775807/00000000-0000-5000-8000-000000000501",
	} {
		if !validPrivateObjectKey(objectKey) {
			t.Errorf("validPrivateObjectKey(%q) = false, want true", objectKey)
		}
	}

	for _, objectKey := range []string{
		"scheduled-media/00000000-0000-4000-8000-000000000501",
		"scheduled-media/v2/0/00000000-0000-5000-8000-000000000501",
		"scheduled-media/v2/01/00000000-0000-5000-8000-000000000501",
		"scheduled-media/v2/1/00000000-0000-0000-0000-000000000000",
		"scheduled-media/v2/1/00000000-0000-4000-8000-000000000501",
		"scheduled-media/v2/1/00000000-0000-5000-8000-00000000050A",
		"scheduled-media/v2/1/not-a-uuid",
	} {
		if validPrivateObjectKey(objectKey) {
			t.Errorf("validPrivateObjectKey(%q) = true, want false", objectKey)
		}
	}
}
