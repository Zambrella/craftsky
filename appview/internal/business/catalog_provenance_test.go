package business

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestBusinessCatalogProvenance(t *testing.T) {
	assertCatalogProvenance(t,
		"catalogdata/iso-3166-1-obp-2026-08-28.html",
		"catalogdata/iso-3166-1-obp-2026-08-28.metadata.json",
		"catalogdata/iso-3166-1-obp-2026-08-28.sha256",
		"https://www.iso.org/obp/ui/#search",
		"f33ff970874a35c9b0d8a535f00d1ad130e4ef9ecc6d0f7052f4d3bf694984c1",
	)
	assertCatalogProvenance(t,
		"catalogdata/iso-4217-list-one-2026-08-28.xml",
		"catalogdata/iso-4217-list-one-2026-08-28.metadata.json",
		"catalogdata/iso-4217-list-one-2026-08-28.sha256",
		"https://www.six-group.com/dam/download/financial-information/data-center/iso-currrency/lists/list-one.xml",
		"838dfb991648cf36df939edd5fe3811737962b75a32252847d239cedd1e291c9",
	)

	if len(assignedCountryCodes) != 249 {
		t.Fatalf("assigned country count = %d, want 249", len(assignedCountryCodes))
	}
	for _, code := range []string{"AF", "AX", "GB", "US", "ZW"} {
		if _, ok := assignedCountryCodes[code]; !ok {
			t.Errorf("assigned country %s missing", code)
		}
	}
	for _, code := range []string{"XK", "ZZ"} {
		if _, ok := assignedCountryCodes[code]; ok {
			t.Errorf("unassigned country %s present", code)
		}
	}
	if len(currencyMinorUnits) != 165 {
		t.Fatalf("currency count = %d, want 165", len(currencyMinorUnits))
	}
}

func assertCatalogProvenance(t *testing.T, sourcePath, metadataPath, sidecarPath, wantSource, wantDigest string) {
	t.Helper()
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source %s: %v", sourcePath, err)
	}
	digest := sha256.Sum256(source)
	if got := hex.EncodeToString(digest[:]); got != wantDigest {
		t.Fatalf("source digest %s = %s, want %s", sourcePath, got, wantDigest)
	}
	var metadata struct {
		Source string `json:"source"`
		SHA256 string `json:"sha256"`
	}
	encodedMetadata, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("read metadata %s: %v", metadataPath, err)
	}
	if err := json.Unmarshal(encodedMetadata, &metadata); err != nil {
		t.Fatalf("decode metadata %s: %v", metadataPath, err)
	}
	if metadata.Source != wantSource || metadata.SHA256 != wantDigest {
		t.Fatalf("metadata %s = %+v", metadataPath, metadata)
	}
	sidecar, err := os.ReadFile(sidecarPath)
	if err != nil {
		t.Fatalf("read sidecar %s: %v", sidecarPath, err)
	}
	if !strings.HasPrefix(string(sidecar), wantDigest+"  ") {
		t.Fatalf("sidecar %s does not contain expected digest", sidecarPath)
	}
}
