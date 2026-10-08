package safetyincident

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestRedactForSinkUsesOneSafeAllowlist(t *testing.T) {
	canaries := []string{
		"CANARY-CONTENT", "CANARY-HASH", "CANARY-EVIDENCE", "CANARY-CONTACT",
		"CANARY-CREDENTIAL", "CANARY-JUDGMENT", "CANARY-PROVIDER",
		"CANARY-OBJECT", "CANARY-AUTHORITY",
	}
	record := RestrictedRecord{
		Reference: "safe-reference", Kind: "incident", State: "untriaged", Priority: 1, Alert: true,
		Sensitive: RestrictedFields{
			Content: canaries[0], Hash: canaries[1], Evidence: canaries[2], Contact: canaries[3],
			Credential: canaries[4], InternalJudgment: canaries[5], ProviderPayload: canaries[6],
			EvidenceObjectKey: canaries[7], AuthorityReference: canaries[8],
		},
	}

	for _, sink := range []Sink{SinkLog, SinkMetric, SinkError, SinkPush, SinkOwnerAPI} {
		t.Run(string(sink), func(t *testing.T) {
			projection, ok := RedactForSink(record, sink)
			if !ok {
				t.Fatal("approved sink rejected")
			}
			encoded, err := json.Marshal(projection)
			if err != nil {
				t.Fatal(err)
			}
			output := string(encoded)
			for _, canary := range canaries {
				if strings.Contains(output, canary) {
					t.Fatalf("%s leaked to %s: %s", canary, sink, output)
				}
			}
			if output != `{"reference":"safe-reference","kind":"incident","state":"untriaged","priority":1,"alert":true}` {
				t.Fatalf("unexpected allowlist projection: %s", output)
			}
		})
	}

	if _, ok := RedactForSink(record, Sink("unknown")); ok {
		t.Fatal("unknown sink must fail closed")
	}

	directJSON, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	slog.New(slog.NewJSONHandler(&logs, nil)).Info("safety", "record", record, "restricted", record.Sensitive)
	for _, output := range []string{string(directJSON), logs.String(), record.Sensitive.String()} {
		for _, canary := range canaries {
			if strings.Contains(output, canary) {
				t.Fatalf("default serialization leaked %s: %s", canary, output)
			}
		}
	}
}
