package app

import "testing"

func TestProductionVideoUsesConfiguredFlag(t *testing.T) {
	const base = "DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=https://craftsky.social\nTAP_WS_URL=ws://tap\n"
	for _, enabled := range []string{"true", "false"} {
		t.Run(enabled, func(t *testing.T) {
			cfg, err := LoadConfig(EnvProd, testConfigFile(t, withProductionOAuth(base)+"VIDEO_ENABLED="+enabled+"\n"))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.VideoEnabled != (enabled == "true") {
				t.Fatalf("VideoEnabled=%v, configured=%s", cfg.VideoEnabled, enabled)
			}
		})
	}
}

func TestVideoDefaultsAndOverrides(t *testing.T) {
	const base = "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nTAP_WS_URL=ws://tap\nCRAFTSKY_DEV_DID=did:plc:dev\nVIDEO_PROVIDER=\nBUNNY_STREAM_LIBRARY_ID=123\nBUNNY_STREAM_API_KEY=private-bunny-key\nBUNNY_STREAM_CDN_HOSTNAME=test.b-cdn.net\n"
	for _, tc := range []struct {
		name, override string
		enabled        bool
		provider       string
	}{
		{"defaults", "", true, "bunny"},
		{"disabled", "VIDEO_ENABLED=false\n", false, "bunny"},
		{"bluesky", "VIDEO_PROVIDER=bluesky\n", true, "bluesky"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := LoadConfig(EnvDev, testConfigFile(t, base+tc.override))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.VideoEnabled != tc.enabled || cfg.VideoProvider != tc.provider {
				t.Fatalf("enabled=%v provider=%s", cfg.VideoEnabled, cfg.VideoProvider)
			}
		})
	}
}
