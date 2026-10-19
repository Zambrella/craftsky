package api

import (
	"context"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/scheduledposts"
)

func TestScheduledPostCraftsPassThroughAndValidation(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name, crafts string
		valid        bool
	}{
		{"three", `[{"craftType":"sewing"},{"craftType":"embroidery"},{"craftType":"future"}]`, true},
		{"four", `[{"craftType":"a"},{"craftType":"b"},{"craftType":"c"},{"craftType":"d"}]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request, err := decodeScheduledPostCreate(strings.NewReader(`{"operationId":"00000000-0000-4000-8000-000000000621","scheduledAt":"2026-10-19T12:05:00Z","payload":{"kind":"standard","text":"hello","sponsored":false,"crafts":` + tc.crafts + `}}`))
			if err != nil {
				t.Fatal(err)
			}
			_, _, encoded, err := validateScheduledPostRequest(now, request, DefaultMediaLimits())
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
			// The worker must apply the same bounds again before the PDS write.
			err = ValidateScheduledPublication(context.Background(), scheduledPolicyFake{}, "did:plc:owner", request.Payload.canonical(), DefaultMediaLimits())
			if (err == nil) != tc.valid {
				t.Fatalf("publication valid=%v, error=%v", tc.valid, err)
			}
			if !tc.valid {
				return
			}
			payload, err := scheduledposts.DecodePayload(encoded)
			if err != nil {
				t.Fatal(err)
			}
			if len(payload.Crafts) != 3 || payload.Crafts[2].CraftType != "future" {
				t.Fatalf("lost saved crafts: %+v", payload.Crafts)
			}
		})
	}
}
