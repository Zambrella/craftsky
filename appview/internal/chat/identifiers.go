package chat

import "errors"

var ErrInvalidIdentifier = errors.New("invalid messaging identifier")

type ConversationID string
type MessageID string

func ParseConversationID(value string) (ConversationID, error) {
	if !validOpaqueID(value) {
		return "", ErrInvalidIdentifier
	}
	return ConversationID(value), nil
}
func ParseMessageID(value string) (MessageID, error) {
	if !validOpaqueID(value) {
		return "", ErrInvalidIdentifier
	}
	return MessageID(value), nil
}
func (ConversationID) String() string                  { return "ConversationID([REDACTED])" }
func (ConversationID) GoString() string                { return "ConversationID([REDACTED])" }
func (MessageID) String() string                       { return "MessageID([REDACTED])" }
func (MessageID) GoString() string                     { return "MessageID([REDACTED])" }
func (id ConversationID) MarshalText() ([]byte, error) { return []byte(id), nil }
func (id MessageID) MarshalText() ([]byte, error)      { return []byte(id), nil }
func (id *ConversationID) UnmarshalText(raw []byte) error {
	value, err := ParseConversationID(string(raw))
	if err == nil {
		*id = value
	}
	return err
}
func (id *MessageID) UnmarshalText(raw []byte) error {
	value, err := ParseMessageID(string(raw))
	if err == nil {
		*id = value
	}
	return err
}
