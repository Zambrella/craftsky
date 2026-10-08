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
