package db

import (
	"context"
	"testing"
)

func TestConnect_BadURLReturnsError(t *testing.T) {
	pool, err := Connect(context.Background(), "not a valid url")
	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("expected error for invalid URL, got nil")
	}
	if pool != nil {
		t.Errorf("pool should be nil on error, got %v", pool)
	}
}
