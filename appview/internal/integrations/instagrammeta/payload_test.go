package instagrammeta

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPayloadReducerReducesOfficialIncomingTextToMinimalDigests(t *testing.T) {
	t.Parallel()

	const (
		officialAccount = "synthetic-official-account"
		senderIGSID     = "synthetic-sender-igsid"
		messageID       = "synthetic-private-message-id"
		challenge       = "CSKY-2345-6789-ABCD-E"
		bodyCanary      = "synthetic-unrelated-body-canary"
	)
	codec, err := NewDigestCodec(bytes.Repeat([]byte{0x42}, 32), func(input string) (string, error) {
		return strings.ToUpper(strings.TrimSpace(input)), nil
	})
	if err != nil {
		t.Fatalf("NewDigestCodec: %v", err)
	}
	reducer, err := NewPayloadReducer(officialAccount, codec)
	if err != nil {
		t.Fatalf("NewPayloadReducer: %v", err)
	}

	body := []byte(`{
  "object":"instagram",
  "synthetic_unknown":"` + bodyCanary + `",
  "entry":[{
    "id":"` + officialAccount + `",
    "time":1721386800,
    "messaging":[{
      "sender":{"id":"` + senderIGSID + `"},
      "recipient":{"id":"` + officialAccount + `"},
      "timestamp":1721386800123,
      "message":{"mid":"` + messageID + `","text":"  csky-2345-6789-abcd-e\n"},
      "synthetic_unknown":"ignored"
    }]
  }]
}`)

	items, err := reducer.Reduce(body)
	if err != nil {
		t.Fatalf("Reduce: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	item := items[0]
	if item.SenderIGSID != senderIGSID || item.OfficialAccountID != officialAccount {
		t.Fatalf("work identity fields = %q/%q", item.SenderIGSID, item.OfficialAccountID)
	}
	if want := time.UnixMilli(1721386800123).UTC(); !item.EventAt.Equal(want) {
		t.Fatalf("EventAt = %s, want %s", item.EventAt, want)
	}
	wantMessageDigest, err := codec.MessageID(messageID)
	if err != nil {
		t.Fatalf("MessageID: %v", err)
	}
	wantChallengeDigest, err := codec.Challenge(challenge)
	if err != nil {
		t.Fatalf("Challenge: %v", err)
	}
	if !item.MessageIDDigest.Equal(wantMessageDigest) {
		t.Fatal("message ID digest mismatch")
	}
	if !item.ChallengeDigest.Equal(wantChallengeDigest) {
		t.Fatal("challenge digest mismatch")
	}

	diagnostic := fmt.Sprintf("item=%v detailed=%+v go=%#v items=%#v", item, item, item, items)
	for _, private := range []string{messageID, challenge, strings.ToLower(challenge), bodyCanary, senderIGSID, officialAccount} {
		if strings.Contains(diagnostic, private) {
			t.Fatalf("diagnostic leaked %q: %s", private, diagnostic)
		}
	}
	encoded, err := json.Marshal(items)
	if err != nil {
		t.Fatalf("Marshal work items: %v", err)
	}
	for _, private := range []string{messageID, challenge, senderIGSID, officialAccount} {
		if bytes.Contains(encoded, []byte(private)) {
			t.Fatalf("JSON serialization leaked %q: %s", private, encoded)
		}
	}
}

func TestPayloadReducerDeduplicatesMessageIDsWithinOneDelivery(t *testing.T) {
	t.Parallel()

	codec, err := NewDigestCodec(bytes.Repeat([]byte{0x51}, 32), func(input string) (string, error) {
		return strings.ToUpper(strings.TrimSpace(input)), nil
	})
	if err != nil {
		t.Fatalf("NewDigestCodec: %v", err)
	}
	reducer, err := NewPayloadReducer("official", codec)
	if err != nil {
		t.Fatalf("NewPayloadReducer: %v", err)
	}
	body := []byte(`{
  "object":"instagram",
  "entry":[{"id":"official","messaging":[
    {"sender":{"id":"sender"},"recipient":{"id":"official"},"timestamp":1721386800123,"message":{"mid":"duplicate-mid","text":"CSKY-2345-6789-ABCD-E"}},
    {"sender":{"id":"sender"},"recipient":{"id":"official"},"timestamp":1721386800456,"message":{"mid":"duplicate-mid","text":"CSKY-2345-6789-ABCD-E"}}
  ]}]
}`)

	items, err := reducer.Reduce(body)
	if err != nil {
		t.Fatalf("Reduce: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want one unique message", len(items))
	}
}

func TestPayloadReducerIgnoresUnsupportedEvents(t *testing.T) {
	t.Parallel()

	validMessage := map[string]any{
		"sender":    map[string]any{"id": "sender"},
		"recipient": map[string]any{"id": "official"},
		"timestamp": int64(1721386800123),
		"message":   map[string]any{"mid": "message", "text": "CSKY-2345-6789-ABCD-E"},
	}
	for name, mutate := range map[string]func(map[string]any){
		"self sender":       func(event map[string]any) { event["sender"] = map[string]any{"id": "official"} },
		"wrong recipient":   func(event map[string]any) { event["recipient"] = map[string]any{"id": "other"} },
		"missing timestamp": func(event map[string]any) { delete(event, "timestamp") },
		"echo": func(event map[string]any) {
			event["message"] = map[string]any{"mid": "message", "text": "CSKY-2345-6789-ABCD-E", "is_echo": true}
		},
		"deleted": func(event map[string]any) {
			event["message"] = map[string]any{"mid": "message", "text": "CSKY-2345-6789-ABCD-E", "is_deleted": true}
		},
		"non-text": func(event map[string]any) {
			event["message"] = map[string]any{"mid": "message", "attachments": []any{map[string]any{"type": "image"}}}
		},
		"missing message ID": func(event map[string]any) {
			event["message"] = map[string]any{"text": "CSKY-2345-6789-ABCD-E"}
		},
		"surrounding prose": func(event map[string]any) {
			event["message"] = map[string]any{"mid": "message", "text": "send CSKY-2345-6789-ABCD-E"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			event := cloneMap(validMessage)
			mutate(event)
			reducer := newStrictTestReducer(t, "official")
			body, err := json.Marshal(map[string]any{
				"object": "instagram",
				"entry": []any{map[string]any{
					"id":        "official",
					"messaging": []any{event},
				}},
			})
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			items, err := reducer.Reduce(body)
			if err != nil {
				t.Fatalf("Reduce: %v", err)
			}
			if len(items) != 0 {
				t.Fatalf("len(items) = %d, want 0", len(items))
			}
		})
	}

	for _, body := range [][]byte{
		[]byte(`{"object":"page","entry":[]}`),
		[]byte(`{"object":"instagram","entry":[{"id":"wrong","messaging":[]}]}`),
	} {
		items, err := newStrictTestReducer(t, "official").Reduce(body)
		if err != nil {
			t.Fatalf("Reduce unsupported body: %v", err)
		}
		if len(items) != 0 {
			t.Fatalf("len(items) = %d, want 0", len(items))
		}
	}
}

func TestPayloadReducerEnforcesBodyAndUniqueEventLimitsWithoutPartialOutput(t *testing.T) {
	t.Parallel()

	reducer := newStrictTestReducer(t, "official")
	atLimit := supportedPayload(t, MaxSupportedEvents)
	items, err := reducer.Reduce(atLimit)
	if err != nil {
		t.Fatalf("Reduce(%d events): %v", MaxSupportedEvents, err)
	}
	if len(items) != MaxSupportedEvents {
		t.Fatalf("len(items) = %d, want %d", len(items), MaxSupportedEvents)
	}

	overLimit := supportedPayload(t, MaxSupportedEvents+1)
	items, err = reducer.Reduce(overLimit)
	if !errors.Is(err, ErrTooManyMessageEvents) {
		t.Fatalf("Reduce(%d events) error = %v, want %v", MaxSupportedEvents+1, err, ErrTooManyMessageEvents)
	}
	if items != nil {
		t.Fatalf("Reduce over event limit returned %d partial items", len(items))
	}

	base := []byte(`{"object":"instagram","entry":[]}`)
	exactBody := append(append([]byte(nil), base...), bytes.Repeat([]byte{' '}, MaxWebhookBodyBytes-len(base))...)
	if _, err := reducer.Reduce(exactBody); err != nil {
		t.Fatalf("Reduce(exact body limit): %v", err)
	}
	if items, err := reducer.Reduce(append(exactBody, ' ')); !errors.Is(err, ErrPayloadTooLarge) || items != nil {
		t.Fatalf("Reduce(over body limit) = (%v, %v), want (nil, %v)", items, err, ErrPayloadTooLarge)
	}
	if items, err := reducer.Reduce([]byte(`{"object":`)); !errors.Is(err, ErrInvalidPayload) || items != nil {
		t.Fatalf("Reduce(malformed) = (%v, %v), want (nil, %v)", items, err, ErrInvalidPayload)
	}
	for _, body := range [][]byte{[]byte(`null`), []byte(`{}`)} {
		if items, err := reducer.Reduce(body); !errors.Is(err, ErrInvalidPayload) || items != nil {
			t.Fatalf("Reduce(missing object) = (%v, %v), want (nil, %v)", items, err, ErrInvalidPayload)
		}
	}
}

func newStrictTestReducer(t *testing.T, officialAccount string) *PayloadReducer {
	t.Helper()
	codec, err := NewDigestCodec(bytes.Repeat([]byte{0x61}, 32), func(input string) (string, error) {
		input = strings.ToUpper(strings.TrimSpace(input))
		if input != "CSKY-2345-6789-ABCD-E" {
			return "", errors.New("invalid challenge")
		}
		return input, nil
	})
	if err != nil {
		t.Fatalf("NewDigestCodec: %v", err)
	}
	reducer, err := NewPayloadReducer(officialAccount, codec)
	if err != nil {
		t.Fatalf("NewPayloadReducer: %v", err)
	}
	return reducer
}

func supportedPayload(t *testing.T, count int) []byte {
	t.Helper()
	messages := make([]any, 0, count)
	for i := range count {
		messages = append(messages, map[string]any{
			"sender":    map[string]any{"id": fmt.Sprintf("sender-%03d", i)},
			"recipient": map[string]any{"id": "official"},
			"timestamp": int64(1721386800000 + i),
			"message": map[string]any{
				"mid":  fmt.Sprintf("message-%03d", i),
				"text": "CSKY-2345-6789-ABCD-E",
			},
		})
	}
	body, err := json.Marshal(map[string]any{
		"object": "instagram",
		"entry": []any{map[string]any{
			"id":        "official",
			"messaging": messages,
		}},
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return body
}

func cloneMap(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		if nested, ok := value.(map[string]any); ok {
			output[key] = cloneMap(nested)
			continue
		}
		output[key] = value
	}
	return output
}
