package instagrammeta

import (
	"encoding/json"
	"errors"
	"time"
)

const (
	MaxWebhookBodyBytes = 256 << 10
	MaxSupportedEvents  = 100
)

var (
	ErrInvalidPayload       = errors.New("invalid Instagram webhook payload")
	ErrPayloadTooLarge      = errors.New("Instagram webhook payload too large")
	ErrTooManyMessageEvents = errors.New("too many Instagram webhook message events")
)

// WorkItem is the entire output of webhook payload reduction. In particular,
// it has no field capable of carrying raw JSON, message text, a plaintext
// challenge, or unrelated upstream data.
type WorkItem struct {
	MessageIDDigest   KeyedDigest `json:"-"`
	SenderIGSID       string      `json:"-"`
	OfficialAccountID string      `json:"-"`
	ChallengeDigest   KeyedDigest `json:"-"`
	EventAt           time.Time   `json:"-"`
}

func (WorkItem) String() string {
	return "instagram webhook work item [REDACTED]"
}

func (WorkItem) GoString() string {
	return "instagram webhook work item [REDACTED]"
}

type PayloadReducer struct {
	officialAccountID string
	digests           *DigestCodec
}

func (*PayloadReducer) String() string {
	return "instagram payload reducer [REDACTED]"
}

func (*PayloadReducer) GoString() string {
	return "instagram payload reducer [REDACTED]"
}

func NewPayloadReducer(officialAccountID string, digests *DigestCodec) (*PayloadReducer, error) {
	if !validProviderID(officialAccountID) {
		return nil, errors.New("official Instagram account ID is required")
	}
	if digests == nil {
		return nil, errors.New("Instagram webhook digest codec is required")
	}
	return &PayloadReducer{officialAccountID: officialAccountID, digests: digests}, nil
}

// Reduce accepts only incoming text messages addressed to the configured
// official account. Unsupported events are ignored; supported events are
// reduced to keyed digests before this method returns.
func (r *PayloadReducer) Reduce(body []byte) ([]WorkItem, error) {
	if len(body) == 0 {
		return nil, ErrInvalidPayload
	}
	if len(body) > MaxWebhookBodyBytes {
		return nil, ErrPayloadTooLarge
	}

	var payload webhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, ErrInvalidPayload
	}
	if payload.Object == "" {
		return nil, ErrInvalidPayload
	}
	if payload.Object != "instagram" {
		return nil, nil
	}

	items := make([]WorkItem, 0)
	seenMessageIDs := make(map[KeyedDigest]struct{})
	for _, entry := range payload.Entry {
		if entry.ID != r.officialAccountID {
			continue
		}
		for _, event := range entry.Messaging {
			if !r.isSupported(event) {
				continue
			}
			challengeDigest, err := r.digests.Challenge(*event.Message.Text)
			if err != nil {
				continue
			}
			messageDigest, err := r.digests.MessageID(event.Message.MID)
			if err != nil {
				continue
			}
			if _, duplicate := seenMessageIDs[messageDigest]; duplicate {
				continue
			}
			if len(items) == MaxSupportedEvents {
				return nil, ErrTooManyMessageEvents
			}
			seenMessageIDs[messageDigest] = struct{}{}
			items = append(items, WorkItem{
				MessageIDDigest:   messageDigest,
				SenderIGSID:       event.Sender.ID,
				OfficialAccountID: r.officialAccountID,
				ChallengeDigest:   challengeDigest,
				EventAt:           time.UnixMilli(event.Timestamp).UTC(),
			})
		}
	}
	return items, nil
}

func (r *PayloadReducer) isSupported(event webhookMessageEvent) bool {
	return validProviderID(event.Sender.ID) &&
		event.Sender.ID != r.officialAccountID &&
		event.Recipient.ID == r.officialAccountID &&
		event.Timestamp > 0 &&
		event.Message != nil &&
		validMessageID(event.Message.MID) &&
		event.Message.Text != nil &&
		*event.Message.Text != "" &&
		!event.Message.IsEcho &&
		!event.Message.IsDeleted
}

type webhookPayload struct {
	Object string         `json:"object"`
	Entry  []webhookEntry `json:"entry"`
}

type webhookEntry struct {
	ID        string                `json:"id"`
	Messaging []webhookMessageEvent `json:"messaging"`
}

type webhookMessageEvent struct {
	Sender    webhookParty    `json:"sender"`
	Recipient webhookParty    `json:"recipient"`
	Timestamp int64           `json:"timestamp"`
	Message   *webhookMessage `json:"message"`
}

type webhookParty struct {
	ID string `json:"id"`
}

type webhookMessage struct {
	MID       string  `json:"mid"`
	Text      *string `json:"text"`
	IsEcho    bool    `json:"is_echo"`
	IsDeleted bool    `json:"is_deleted"`
}
