package app

import (
	"strings"
	"testing"
	"time"
)

func TestPushConfigDefaultsDisabledAndValidatesEveryEnabledEnvironment(t *testing.T) {
	base := "DATABASE_URL=postgres://example\nALLOWED_ORIGINS=https://craftsky.social\nTAP_WS_URL=ws://tap\n"
	cfg, err := LoadConfig(EnvProd, testConfigFile(t, base))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PushEnabled || cfg.PushBatchSize != 100 || cfg.PushPollInterval != time.Second || cfg.PushLeaseDuration != time.Minute || cfg.PushSendTimeout != 10*time.Second {
		t.Fatalf("defaults=%+v", cfg)
	}
	_, err = LoadConfig(EnvProd, testConfigFile(t, base+"PUSH_ENABLED=true\n"))
	if err == nil || !strings.Contains(err.Error(), "FIREBASE_PROJECT_ID") {
		t.Fatalf("err=%v", err)
	}
	cfg, err = LoadConfig(EnvProd, testConfigFile(t, base+"PUSH_ENABLED=true\nFIREBASE_PROJECT_ID=craftsky-test\n"))
	if err != nil || !cfg.PushEnabled {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}

	devBase := "DATABASE_URL=postgres://example\nALLOWED_ORIGINS=*\nTAP_WS_URL=ws://tap\nCRAFTSKY_DEV_DID=did:plc:test\n"
	_, err = LoadConfig(EnvDev, testConfigFile(t, devBase+"PUSH_ENABLED=true\n"))
	if err == nil || !strings.Contains(err.Error(), "FIREBASE_PROJECT_ID") {
		t.Fatalf("dev err=%v", err)
	}
}
