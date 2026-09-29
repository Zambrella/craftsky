package main

import (
	"bytes"
	"context"
	"testing"
)

func TestRunPrintsVersionBeforeConfiguration(t *testing.T) {
	for _, args := range [][]string{
		{"appview", "version"},
		{"appview", "--version"},
	} {
		var output bytes.Buffer
		if err := run(context.Background(), args, &output); err != nil {
			t.Fatalf("run(%v): %v", args, err)
		}
		if got := output.String(); got != "dev\n" {
			t.Fatalf("run(%v) output = %q, want %q", args, got, "dev\\n")
		}
	}
}
