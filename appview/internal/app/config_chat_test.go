package app

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

// IT-006 / FR-006, NFR-002: low-entropy private text is never hashed with
// an absent, weak, shared or accidentally printable operation key.
func TestChatOperationKeyRejectsMalformedConfiguration(t *testing.T) {
	const base = "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n"
	_, err := LoadConfig(EnvDev, testConfigFile(t, base+"CHAT_OPERATION_HMAC_KEY=private-short-key-canary\n"))
	if err == nil || !strings.Contains(err.Error(), "CHAT_OPERATION_HMAC_KEY") || strings.Contains(err.Error(), "private-short-key-canary") {
		t.Fatalf("malformed key error=%v", err)
	}
}

func TestChatOperationKeyIsDistinctAndRedacted(t *testing.T) {
	const base = "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n"
	key := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x63}, 32))
	cfg, err := LoadConfig(EnvDev, testConfigFile(t, base+"CHAT_OPERATION_HMAC_KEY="+key+"\nCHAT_OPERATION_HMAC_KEY_VERSION=7\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ChatOperationHMACKey.Reveal() != key || cfg.ChatOperationHMACKeyVersion != 7 {
		t.Fatal("operation key/version not retained")
	}
	if strings.Contains(fmt.Sprintf("%+v %#v", cfg, cfg), key) {
		t.Fatal("operation secret exposed by formatting")
	}
}
