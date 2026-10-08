package sourcevalidation

import (
	"encoding/json"
	"time"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/rivo/uniseg"

	"social.craftsky/appview/internal/languages"
	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	lexiconschema "social.craftsky/appview/internal/lexicon/schema"
	"social.craftsky/appview/internal/tap"
)

type Status string

const (
	Pending Status = "pending"
	Valid   Status = "valid"
	Invalid Status = "invalid"
)

type Result struct {
	StructuralStatus Status
	SemanticStatus   Status
	Reason           string
	Cause            error
}

type recordKeyPolicy uint8

const (
	recordKeySelf recordKeyPolicy = iota + 1
	recordKeyTID
)

var recordKeyPolicies = map[syntax.NSID]recordKeyPolicy{
	"social.craftsky.actor.profile":    recordKeySelf,
	"social.craftsky.feed.post":        recordKeyTID,
	"social.craftsky.feed.like":        recordKeyTID,
	"social.craftsky.feed.repost":      recordKeyTID,
	"social.craftsky.business.profile": recordKeySelf,
	"social.craftsky.business.event":   recordKeyTID,
	"app.bsky.actor.profile":           recordKeySelf,
	"app.bsky.graph.follow":            recordKeyTID,
	"app.bsky.graph.block":             recordKeyTID,
}

func Validate(event tap.Event) Result {
	keyPolicy, ok := recordKeyPolicies[event.Collection]
	if !ok {
		return invalid("unsupported_collection")
	}
	if !validRecordKey(keyPolicy, event.Rkey) {
		return invalid("invalid_record_key")
	}
	switch event.Action {
	case "delete":
		return valid()
	case "create", "update":
	default:
		return invalid("unsupported_action")
	}
	if len(event.Record) == 0 || !json.Valid(event.Record) {
		return invalid("malformed_record")
	}
	if !matchesExplicitType(event.Record, event.Collection) {
		return invalid("invalid_lexicon")
	}
	return validateRecordBody(event)
}

func validRecordKey(policy recordKeyPolicy, rkey syntax.RecordKey) bool {
	switch policy {
	case recordKeySelf:
		return rkey == "self"
	case recordKeyTID:
		_, err := syntax.ParseTID(string(rkey))
		return err == nil
	default:
		return false
	}
}

func validateRecordBody(event tap.Event) Result {
	switch event.Collection {
	case "social.craftsky.actor.profile":
		if err := lexiconschema.ValidateCraftskyRecord(event.Record, event.Collection.String()); err != nil {
			result := invalid("invalid_lexicon")
			result.Cause = err
			return result
		}
		var record craftskylex.ActorProfile
		if err := json.Unmarshal(event.Record, &record); err != nil {
			return invalid("malformed_record")
		}
	case "social.craftsky.business.profile":
		if err := lexiconschema.ValidateBusinessRecord(event.Record, event.Collection.String()); err != nil {
			result := invalid("invalid_lexicon")
			result.Cause = err
			return result
		}
		var record craftskylex.BusinessProfile
		if err := json.Unmarshal(event.Record, &record); err != nil {
			return invalid("malformed_record")
		}
	case "social.craftsky.business.event":
		if err := lexiconschema.ValidateBusinessRecord(event.Record, event.Collection.String()); err != nil {
			result := invalid("invalid_lexicon")
			result.Cause = err
			return result
		}
		var record craftskylex.BusinessEvent
		if err := json.Unmarshal(event.Record, &record); err != nil {
			return invalid("malformed_record")
		}
		if !validTimestamp(record.StartsAt) || !validTimestamp(record.EndsAt) || !validTimestamp(record.CreatedAt) {
			return semanticInvalid("invalid_timestamp")
		}
	case "social.craftsky.feed.post":
		for _, field := range []string{"text", "sponsored", "createdAt"} {
			if !hasRequiredFields(event.Record, field) {
				result := invalid("invalid_lexicon")
				result.Cause = &lexiconschema.ValidationError{MissingField: field}
				return result
			}
		}
		var record craftskylex.FeedPost
		if err := json.Unmarshal(event.Record, &record); err != nil {
			return invalid("malformed_record")
		}
		if len(record.Text) > 20000 || uniseg.GraphemeClusterCount(record.Text) > 2000 {
			return invalid("invalid_lexicon")
		}
		if len(record.Images) > 4 || len(record.Langs) > 3 {
			return invalid("invalid_lexicon")
		}
		if err := languages.ValidatePostTags(record.Langs); err != nil {
			return semanticInvalid("invalid_language")
		}
		if !validTimestamp(record.CreatedAt) {
			return semanticInvalid("invalid_timestamp")
		}
	case "social.craftsky.feed.like":
		var record craftskylex.FeedLike
		if err := json.Unmarshal(event.Record, &record); err != nil {
			return invalid("malformed_record")
		}
		if result := validateInteraction(record.Subject, record.CreatedAt); result.SemanticStatus != Valid {
			return result
		}
	case "social.craftsky.feed.repost":
		var record craftskylex.FeedRepost
		if err := json.Unmarshal(event.Record, &record); err != nil {
			return invalid("malformed_record")
		}
		if result := validateInteraction(record.Subject, record.CreatedAt); result.SemanticStatus != Valid {
			return result
		}
	case "app.bsky.actor.profile":
		var record bsky.ActorProfile
		if err := json.Unmarshal(event.Record, &record); err != nil {
			return invalid("malformed_record")
		}
		if exceedsStringLimit(record.DisplayName, 640, 64) ||
			exceedsStringLimit(record.Description, 2560, 256) ||
			exceedsStringLimit(record.Pronouns, 200, 20) {
			return invalid("invalid_lexicon")
		}
		if record.CreatedAt != nil && !validTimestamp(*record.CreatedAt) {
			return semanticInvalid("invalid_timestamp")
		}
	case "app.bsky.graph.follow":
		var record bsky.GraphFollow
		if err := json.Unmarshal(event.Record, &record); err != nil {
			return invalid("malformed_record")
		}
		if _, err := syntax.ParseDID(record.Subject); err != nil {
			return semanticInvalid("invalid_subject")
		}
		if !validTimestamp(record.CreatedAt) {
			return semanticInvalid("invalid_timestamp")
		}
	case "app.bsky.graph.block":
		var record bsky.GraphBlock
		if err := json.Unmarshal(event.Record, &record); err != nil {
			return invalid("malformed_record")
		}
		if _, err := syntax.ParseDID(record.Subject); err != nil {
			return semanticInvalid("invalid_subject")
		}
		if !validTimestamp(record.CreatedAt) {
			return semanticInvalid("invalid_timestamp")
		}
	default:
		return invalid("unsupported_collection")
	}
	return valid()
}

func validateInteraction(subject *comatproto.RepoStrongRef, createdAt string) Result {
	if subject == nil || subject.Uri == "" || subject.Cid == "" {
		return semanticInvalid("missing_subject")
	}
	if _, err := syntax.ParseATURI(subject.Uri); err != nil {
		return semanticInvalid("invalid_subject")
	}
	if !validTimestamp(createdAt) {
		return semanticInvalid("invalid_timestamp")
	}
	return valid()
}

func hasRequiredFields(raw json.RawMessage, fields ...string) bool {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return false
	}
	for _, field := range fields {
		if _, ok := object[field]; !ok {
			return false
		}
	}
	return true
}

func matchesExplicitType(raw json.RawMessage, collection syntax.NSID) bool {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return false
	}
	rawType, ok := object["$type"]
	if !ok {
		return true
	}
	var recordType string
	return json.Unmarshal(rawType, &recordType) == nil && recordType == collection.String()
}

func validTimestamp(value string) bool {
	_, err := time.Parse(time.RFC3339Nano, value)
	return err == nil
}

func exceedsStringLimit(value *string, maxBytes, maxGraphemes int) bool {
	return value != nil && (len(*value) > maxBytes || uniseg.GraphemeClusterCount(*value) > maxGraphemes)
}

func valid() Result {
	return Result{StructuralStatus: Valid, SemanticStatus: Valid}
}

func invalid(reason string) Result {
	return Result{
		StructuralStatus: Invalid,
		SemanticStatus:   Invalid,
		Reason:           reason,
	}
}

func semanticInvalid(reason string) Result {
	return Result{
		StructuralStatus: Valid,
		SemanticStatus:   Invalid,
		Reason:           reason,
	}
}
