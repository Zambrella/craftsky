package index

import (
	"context"
	"strings"
	"testing"
)

func TestNotImplemented_BackfillErrors(t *testing.T) {
	var idx Indexer = NotImplemented{}
	err := idx.Backfill(context.Background(), "did:plc:abc")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "indexer") || !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("err = %q, want containing 'indexer' and 'not yet implemented'", err.Error())
	}
}
