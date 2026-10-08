package schema_test

import (
	"encoding/json"
	"errors"
	"social.craftsky/appview/internal/lexicon/schema"
	"social.craftsky/appview/internal/observability"
	"strings"
	"testing"
)

func TestOversizedPublishedBlobRetainsSafeCause(t *testing.T) {
	raw := json.RawMessage(`{"products":[{"title":"private-canary","uri":"https://shop.example/yarn","image":{"image":{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"image/jpeg","size":11189477}}}]}`)
	err := schema.ValidateBusinessRecord(raw, "social.craftsky.business.profile")
	var failure *schema.ValidationError
	if !errors.As(err, &failure) || failure.BlobBytes != 11189477 || errors.Unwrap(failure) == nil {
		t.Fatalf("missing typed original validator cause: %+v", err)
	}
	encoded, _ := json.Marshal(observability.DescribeError(err, nil))
	if !strings.Contains(string(encoded), "11189477") || !strings.Contains(string(encoded), "lexicon size limit") || strings.Contains(string(encoded), "private-canary") {
		t.Fatalf("unsafe/unhelpful cause: %s", encoded)
	}
}
