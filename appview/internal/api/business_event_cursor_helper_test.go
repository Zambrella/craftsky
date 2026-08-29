package api_test

import (
	"bytes"
	"testing"

	"social.craftsky/appview/internal/api"
)

func testEventCursorCodec(t *testing.T) *api.EventCursorCodec {
	t.Helper()
	codec, err := api.NewEventCursorCodec(bytes.Repeat([]byte{0x52}, 32))
	if err != nil {
		t.Fatalf("new event cursor codec: %v", err)
	}
	return codec
}
