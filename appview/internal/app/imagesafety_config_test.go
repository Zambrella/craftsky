package app

import (
	"strings"
	"testing"

	"social.craftsky/appview/internal/imagesafety"
)

func TestImageSafetyConfigurationFailsClosedByEnvironment(t *testing.T) {
	const devBase = "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n"
	const prodBase = "DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=https://craftsky.social\nTAP_WS_URL=ws://tap\n"

	dev, err := LoadConfig(EnvDev, testConfigFile(t, devBase))
	if err != nil {
		t.Fatal(err)
	}
	if !dev.ImageSafety.Ready() || !dev.ImageSafety.CanMarkClear() {
		t.Fatal("development fixture scanner should be ready")
	}

	prod, err := LoadConfig(EnvProd, testConfigFile(t, withProductionOAuth(prodBase)))
	if err != nil {
		t.Fatal(err)
	}
	if prod.ImageSafety.Ready() || prod.ImageSafety.CanMarkClear() {
		t.Fatal("production must remain unready without a validated adapter")
	}

	stub, err := LoadConfig(EnvProd, testConfigFile(t, withProductionOAuth(prodBase)+
		"IMAGE_SAFETY_SCANNER_MODE=stub\n"+
		"IMAGE_SAFETY_SCANNER_ID=fixture\n"+
		"IMAGE_SAFETY_POLICY_VERSION=policy-v1\n"+
		"IMAGE_SAFETY_CORPUS_VERSION=corpus-v1\n"))
	if err != nil {
		t.Fatal(err)
	}
	if stub.ImageSafety.Ready() || stub.ImageSafety.CanMarkClear() {
		t.Fatal("production fixture scanner must not become ready")
	}
}

func TestImageSafetyWorkerGeometryIsValidated(t *testing.T) {
	const base = "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n"
	_, err := LoadConfig(EnvDev, testConfigFile(t, base+
		"IMAGE_SAFETY_POLL_INTERVAL=1m\nIMAGE_SAFETY_LEASE_DURATION=1m\n"))
	if err == nil || !strings.Contains(err.Error(), "IMAGE_SAFETY_LEASE_DURATION") {
		t.Fatalf("unsafe lease geometry error=%v", err)
	}

	approved := imagesafety.Config{
		Environment: imagesafety.EnvironmentProduction,
		Mode:        imagesafety.ScannerModeApproved,
		ScannerID:   "approved-scanner", PolicyVersion: "policy-v1", CorpusVersion: "corpus-v1",
		AdapterReady: true,
	}
	if !approved.Ready() || !approved.CanMarkClear() {
		t.Fatal("validated approved adapter test double should be ready")
	}
}

func TestImageSafetyProjectionKeyFailsClosedWhenAdapterIsUnconfigured(t *testing.T) {
	key := imageSafetyProjectionKey(imagesafety.Config{})
	if key.ScannerID == "" || key.PolicyVersion == "" || key.CorpusVersion == "" {
		t.Fatalf("unconfigured projection key must remain gateable: %+v", key)
	}

	configured := imagesafety.Config{
		ScannerID: "scanner", PolicyVersion: "policy", CorpusVersion: "corpus",
	}
	key = imageSafetyProjectionKey(configured)
	if key.ScannerID != configured.ScannerID || key.PolicyVersion != configured.PolicyVersion || key.CorpusVersion != configured.CorpusVersion {
		t.Fatalf("configured projection key = %+v", key)
	}
}
