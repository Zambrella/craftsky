package safetyincident

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestS3EvidenceStoreContract(t *testing.T) {
	const (
		bucket = "restricted-safety-evidence"
		key    = "10000000-0000-4000-8000-000000000001/20000000-0000-4000-8000-000000000002"
	)
	want := []byte("restricted-evidence")
	var stored []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" || r.Header.Get("X-Amz-Acl") != "" {
			http.Error(w, "invalid request", http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/"+bucket && r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path != "/"+bucket+"/"+key {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodPut:
			stored, _ = io.ReadAll(r.Body)
			w.Header().Set("ETag", `"restricted"`)
		case http.MethodGet:
			_, _ = w.Write(stored)
		case http.MethodDelete:
			stored = nil
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()
	store, err := NewS3EvidenceStore(context.Background(), S3EvidenceStoreConfig{
		Endpoint: server.URL, Region: "test", Bucket: bucket,
		AccessKeyID: "key", SecretAccessKey: "secret", Environment: "test",
	})
	if err != nil || store.Check(context.Background()) != nil {
		t.Fatalf("construct/check store: %v", err)
	}
	ref, err := store.Put(context.Background(), RestrictedObject{Key: key, Bytes: want})
	if err != nil {
		t.Fatal(err)
	}
	reader, err := store.open(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(reader)
	_ = reader.Close()
	if !bytes.Equal(got, want) {
		t.Fatalf("object=%q, want %q", got, want)
	}
	if err := store.Delete(context.Background(), ref); err != nil {
		t.Fatal(err)
	}
	_, err = NewS3EvidenceStore(context.Background(), S3EvidenceStoreConfig{
		Endpoint: server.URL, Region: "test", Bucket: bucket,
		AccessKeyID: "key", SecretAccessKey: "secret", Environment: "prod",
	})
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("production HTTP endpoint error=%v", err)
	}
}
