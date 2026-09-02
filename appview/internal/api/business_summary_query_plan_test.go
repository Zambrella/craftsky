package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
)

func TestBusinessSummaryQueryPlanUsesOneBoundedBatchForOneAndFiftyRows(t *testing.T) {
	for _, rows := range []int{1, 50} {
		t.Run(fmt.Sprintf("rows_%d", rows), func(t *testing.T) {
			reader := &summaryAccountTypeReader{}
			items := make([]map[string]string, rows)
			for i := range items {
				items[i] = map[string]string{
					"did":    fmt.Sprintf("did:plc:summary-%02d", i),
					"handle": fmt.Sprintf("summary-%02d.test", i),
				}
			}
			raw, err := json.Marshal(map[string]any{"items": items})
			if err != nil {
				t.Fatalf("marshal fixture: %v", err)
			}

			hydrated, err := api.NewIdentityAccountTypeHydrator(reader).HydrateJSON(context.Background(), raw)
			if err != nil {
				t.Fatalf("hydrate %d summaries: %v", rows, err)
			}
			if len(reader.calls) != 1 {
				t.Fatalf("ReadAccountTypes calls = %d, want 1", len(reader.calls))
			}
			if got := len(reader.calls[0]); got != rows || got > 50 {
				t.Fatalf("batch size = %d, want %d and at most 50", got, rows)
			}
			var response struct {
				Items []struct {
					DID         syntax.DID `json:"did"`
					AccountType string     `json:"accountType"`
				} `json:"items"`
			}
			if err := json.Unmarshal(hydrated, &response); err != nil {
				t.Fatalf("decode hydrated response: %v", err)
			}
			for _, item := range response.Items {
				if item.AccountType != "regular" {
					t.Errorf("%s accountType = %q, want regular", item.DID, item.AccountType)
				}
			}
		})
	}
}
