package app

import (
	"strings"
	"testing"
	"time"
)

func TestOwnerEffectAndScheduledMediaConfigDefaults(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n")
	cfg, err := LoadConfig(EnvDev, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.OwnerFenceAcquireTimeout != 5*time.Second ||
		cfg.PDSEffectTimeout != 10*time.Second ||
		cfg.ScheduledMediaPutTimeout != 30*time.Second {
		t.Fatalf("lifecycle effect defaults = fence %s pds %s object %s",
			cfg.OwnerFenceAcquireTimeout, cfg.PDSEffectTimeout, cfg.ScheduledMediaPutTimeout)
	}
}

func TestTerminalTapAckBudgetDefaultsAndExactGeometry(t *testing.T) {
	base := "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n"
	cfg, err := LoadConfig(EnvDev, testConfigFile(t, base))
	if err != nil {
		t.Fatalf("LoadConfig defaults: %v", err)
	}
	if cfg.TapTerminalTransactionBudget != time.Second || cfg.TapAckSafetyMargin != 500*time.Millisecond {
		t.Fatalf("terminal Tap defaults = transaction %s margin %s",
			cfg.TapTerminalTransactionBudget, cfg.TapAckSafetyMargin)
	}
	if got := cfg.tapIngestionTimeout(); got != 9500*time.Millisecond {
		t.Fatalf("Tap ingestion timeout=%s, want 9.5s with ACK margin reserved", got)
	}
	if got := cfg.tapTerminalCommitTimeout(); got != 6*time.Second {
		t.Fatalf("terminal Tap commit timeout=%s, want 6s fence plus fixed transaction budget", got)
	}

	if _, err := LoadConfig(EnvDev, testConfigFile(t, base+
		"OWNER_FENCE_ACQUIRE_TIMEOUT=5s\n"+
		"TAP_TERMINAL_TRANSACTION_BUDGET=1s\n"+
		"TAP_ACK_SAFETY_MARGIN=500ms\n"+
		"TAP_ACK_TIMEOUT=6.500000001s\n")); err != nil {
		t.Fatalf("one nanosecond of validated ACK headroom should pass: %v", err)
	}
	_, err = LoadConfig(EnvDev, testConfigFile(t, base+
		"OWNER_FENCE_ACQUIRE_TIMEOUT=5s\n"+
		"TAP_TERMINAL_TRANSACTION_BUDGET=1s\n"+
		"TAP_ACK_SAFETY_MARGIN=500ms\n"+
		"TAP_ACK_TIMEOUT=6.5s\n"))
	if err == nil || !strings.Contains(err.Error(), "TAP_TERMINAL_TRANSACTION_BUDGET") ||
		!strings.Contains(err.Error(), "TAP_ACK_SAFETY_MARGIN") || !strings.Contains(err.Error(), "TAP_ACK_TIMEOUT") {
		t.Fatalf("exact exhausted ACK budget error = %v", err)
	}
}

func TestOwnerEffectAndScheduledMediaDurationsFailClosed(t *testing.T) {
	base := "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n"
	for _, test := range []struct {
		name     string
		override string
		want     string
	}{
		{name: "zero owner fence", override: "OWNER_FENCE_ACQUIRE_TIMEOUT=0s\n", want: "OWNER_FENCE_ACQUIRE_TIMEOUT"},
		{name: "zero terminal Tap transaction budget", override: "TAP_TERMINAL_TRANSACTION_BUDGET=0s\n", want: "TAP_TERMINAL_TRANSACTION_BUDGET"},
		{name: "zero Tap ACK safety margin", override: "TAP_ACK_SAFETY_MARGIN=0s\n", want: "TAP_ACK_SAFETY_MARGIN"},
		{name: "zero PDS effect", override: "PDS_EFFECT_TIMEOUT=0s\n", want: "PDS_EFFECT_TIMEOUT"},
		{name: "zero media put", override: "SCHEDULED_MEDIA_PUT_TIMEOUT=0s\n", want: "SCHEDULED_MEDIA_PUT_TIMEOUT"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadConfig(EnvDev, testConfigFile(t, base+test.override))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %s", err, test.want)
			}
		})
	}
}

func TestAccountDeletionIntentExpiryConfigDefaults(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n")
	cfg, err := LoadConfig(EnvDev, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.AccountDeletionIntentTTL != 10*time.Minute ||
		cfg.AccountDeletionIntentSweepInterval != time.Minute ||
		cfg.AccountDeletionIntentSweepBatch != 100 {
		t.Fatalf(
			"account deletion intent defaults = ttl %s interval %s batch %d",
			cfg.AccountDeletionIntentTTL,
			cfg.AccountDeletionIntentSweepInterval,
			cfg.AccountDeletionIntentSweepBatch,
		)
	}
}

func TestAccountDeletionIntentExpiryConfigFailsClosed(t *testing.T) {
	base := "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n"
	for _, test := range []struct {
		name     string
		override string
		want     string
	}{
		{name: "zero ttl", override: "ACCOUNT_DELETION_INTENT_TTL=0s\n", want: "ACCOUNT_DELETION_INTENT_TTL"},
		{name: "zero sweep interval", override: "ACCOUNT_DELETION_INTENT_SWEEP_INTERVAL=0s\n", want: "ACCOUNT_DELETION_INTENT_SWEEP_INTERVAL"},
		{name: "zero batch", override: "ACCOUNT_DELETION_INTENT_SWEEP_BATCH=0\n", want: "ACCOUNT_DELETION_INTENT_SWEEP_BATCH"},
		{
			name:     "sweep cannot lag beyond intent lifetime",
			override: "ACCOUNT_DELETION_INTENT_TTL=10m\nACCOUNT_DELETION_INTENT_SWEEP_INTERVAL=11m\n",
			want:     "ACCOUNT_DELETION_INTENT_SWEEP_INTERVAL must not exceed ACCOUNT_DELETION_INTENT_TTL",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadConfig(EnvDev, testConfigFile(t, base+test.override))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %s", err, test.want)
			}
		})
	}
}

func TestAuthLifecycleWorkerConfigDefaults(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n")
	cfg, err := LoadConfig(EnvDev, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.OAuthRevocationPollInterval != 5*time.Second ||
		cfg.OAuthRevocationBatchSize != 20 ||
		cfg.OAuthRevocationLeaseDuration != time.Minute ||
		cfg.OAuthRevocationOperationTimeout != 20*time.Second ||
		cfg.OAuthRevocationMaxAttempts != 5 ||
		cfg.OAuthRevocationBackoffMin != time.Minute ||
		cfg.OAuthRevocationBackoffMax != time.Hour ||
		cfg.OAuthRevocationMaxCredentialRetention != 24*time.Hour {
		t.Fatalf("unexpected OAuth revocation defaults: %+v", cfg)
	}
	if cfg.AuthAuxiliaryCleanupPollInterval != 5*time.Second ||
		cfg.AuthAuxiliaryCleanupBatchSize != 20 ||
		cfg.AuthAuxiliaryCleanupLeaseDuration != time.Minute ||
		cfg.AuthAuxiliaryCleanupOperationTimeout != 20*time.Second ||
		cfg.AuthAuxiliaryCleanupMaxAttempts != 5 ||
		cfg.AuthAuxiliaryCleanupBackoffMin != time.Minute ||
		cfg.AuthAuxiliaryCleanupBackoffMax != time.Hour {
		t.Fatalf("unexpected auxiliary cleanup defaults: %+v", cfg)
	}
	if cfg.SessionExpirySweepInterval != time.Minute || cfg.SessionExpirySweepBatch != 100 {
		t.Fatalf("unexpected session expiry defaults: interval %s batch %d", cfg.SessionExpirySweepInterval, cfg.SessionExpirySweepBatch)
	}
}

func TestAuthLifecycleWorkerConfigFailsClosed(t *testing.T) {
	base := "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n"
	for _, test := range []struct {
		name     string
		override string
		want     string
	}{
		{
			name:     "revocation operation must fit lease",
			override: "OAUTH_REVOCATION_LEASE_DURATION=20s\nOAUTH_REVOCATION_OPERATION_TIMEOUT=20s\n",
			want:     "OAUTH_REVOCATION_OPERATION_TIMEOUT must be less than OAUTH_REVOCATION_LEASE_DURATION",
		},
		{
			name:     "auxiliary operation must fit lease",
			override: "AUTH_AUXILIARY_CLEANUP_LEASE_DURATION=20s\nAUTH_AUXILIARY_CLEANUP_OPERATION_TIMEOUT=20s\n",
			want:     "AUTH_AUXILIARY_CLEANUP_OPERATION_TIMEOUT must be less than AUTH_AUXILIARY_CLEANUP_LEASE_DURATION",
		},
		{
			name:     "revocation backoff is ordered",
			override: "OAUTH_REVOCATION_BACKOFF_MIN=2h\nOAUTH_REVOCATION_BACKOFF_MAX=1h\n",
			want:     "OAUTH_REVOCATION_BACKOFF_MAX must not be less than OAUTH_REVOCATION_BACKOFF_MIN",
		},
		{
			name:     "retention cannot outlive absolute session lifetime",
			override: "OAUTH_SESSION_ABSOLUTE_LIFETIME=1h\nCRAFTSKY_SESSION_INACTIVITY=30m\nOAUTH_REVOCATION_MAX_CREDENTIAL_RETENTION=2h\n",
			want:     "OAUTH_REVOCATION_MAX_CREDENTIAL_RETENTION must not exceed OAUTH_SESSION_ABSOLUTE_LIFETIME",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadConfig(EnvDev, testConfigFile(t, base+test.override))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %s", err, test.want)
			}
		})
	}
}

func TestTerminalPurgeWorkerConfigDefaults(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n")
	cfg, err := LoadConfig(EnvDev, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.TerminalPurgePollInterval != time.Second ||
		cfg.TerminalPurgeComponentLimit != 16 ||
		cfg.TerminalPurgeRowBatchSize != 100 ||
		cfg.TerminalPurgeLeaseDuration != time.Minute ||
		cfg.TerminalPurgeRetryDelay != time.Second {
		t.Fatalf("unexpected terminal purge defaults: %+v", cfg)
	}
}

func TestIdentityCacheRefreshWorkerConfigDefaults(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n")
	cfg, err := LoadConfig(EnvDev, path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IdentityCacheRefreshPollInterval != 5*time.Minute ||
		cfg.IdentityCacheRefreshBatchSize != 100 ||
		cfg.IdentityCacheRefreshOperationTimeout != 10*time.Second ||
		cfg.IdentityCacheRefreshRetryDelay != 15*time.Minute {
		t.Fatalf("unexpected identity refresh defaults: interval=%s batch=%d timeout=%s retry=%s",
			cfg.IdentityCacheRefreshPollInterval, cfg.IdentityCacheRefreshBatchSize,
			cfg.IdentityCacheRefreshOperationTimeout, cfg.IdentityCacheRefreshRetryDelay)
	}
}

func TestIdentityCacheRefreshWorkerConfigFailsClosed(t *testing.T) {
	base := "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n"
	for _, test := range []struct {
		name     string
		override string
		want     string
	}{
		{name: "zero poll", override: "IDENTITY_CACHE_REFRESH_POLL_INTERVAL=0s\n", want: "IDENTITY_CACHE_REFRESH_POLL_INTERVAL"},
		{name: "zero batch", override: "IDENTITY_CACHE_REFRESH_BATCH_SIZE=0\n", want: "IDENTITY_CACHE_REFRESH_BATCH_SIZE"},
		{name: "zero operation timeout", override: "IDENTITY_CACHE_REFRESH_OPERATION_TIMEOUT=0s\n", want: "IDENTITY_CACHE_REFRESH_OPERATION_TIMEOUT"},
		{name: "zero retry delay", override: "IDENTITY_CACHE_REFRESH_RETRY_DELAY=0s\n", want: "IDENTITY_CACHE_REFRESH_RETRY_DELAY"},
		{name: "oversized poll", override: "IDENTITY_CACHE_REFRESH_POLL_INTERVAL=1h1ns\n", want: "IDENTITY_CACHE_REFRESH_POLL_INTERVAL"},
		{name: "oversized batch", override: "IDENTITY_CACHE_REFRESH_BATCH_SIZE=1001\n", want: "IDENTITY_CACHE_REFRESH_BATCH_SIZE"},
		{name: "oversized operation timeout", override: "IDENTITY_CACHE_REFRESH_OPERATION_TIMEOUT=1m1ns\n", want: "IDENTITY_CACHE_REFRESH_OPERATION_TIMEOUT"},
		{name: "oversized retry delay", override: "IDENTITY_CACHE_REFRESH_RETRY_DELAY=24h1ns\n", want: "IDENTITY_CACHE_REFRESH_RETRY_DELAY"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadConfig(EnvDev, testConfigFile(t, base+test.override))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %s", err, test.want)
			}
		})
	}
}

func TestTerminalPurgeWorkerConfigFailsClosed(t *testing.T) {
	base := "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n"
	for _, test := range []struct {
		name     string
		override string
		want     string
	}{
		{name: "zero poll", override: "TERMINAL_PURGE_POLL_INTERVAL=0s\n", want: "TERMINAL_PURGE_POLL_INTERVAL"},
		{name: "zero component limit", override: "TERMINAL_PURGE_COMPONENT_LIMIT=0\n", want: "TERMINAL_PURGE_COMPONENT_LIMIT"},
		{name: "zero row batch", override: "TERMINAL_PURGE_ROW_BATCH_SIZE=0\n", want: "TERMINAL_PURGE_ROW_BATCH_SIZE"},
		{name: "zero lease", override: "TERMINAL_PURGE_LEASE_DURATION=0s\n", want: "TERMINAL_PURGE_LEASE_DURATION"},
		{name: "zero retry", override: "TERMINAL_PURGE_RETRY_DELAY=0s\n", want: "TERMINAL_PURGE_RETRY_DELAY"},
		{
			name:     "lease must exceed poll",
			override: "TERMINAL_PURGE_POLL_INTERVAL=30s\nTERMINAL_PURGE_LEASE_DURATION=30s\n",
			want:     "TERMINAL_PURGE_LEASE_DURATION must exceed TERMINAL_PURGE_POLL_INTERVAL",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadConfig(EnvDev, testConfigFile(t, base+test.override))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %s", err, test.want)
			}
		})
	}
}

func TestTapQuarantineReplayWorkerConfig(t *testing.T) {
	base := "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n"
	cfg, err := LoadConfig(EnvDev, testConfigFile(t, base))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.TapQuarantinePollInterval != time.Second ||
		cfg.TapQuarantineLeaseDuration != 30*time.Second ||
		cfg.TapQuarantineOperationTimeout != 10*time.Second ||
		cfg.TapQuarantineBatchSize != 8 {
		t.Fatalf("unexpected Tap quarantine defaults: %+v", cfg)
	}
	for _, test := range []struct {
		override string
		want     string
	}{
		{override: "TAP_QUARANTINE_POLL_INTERVAL=0s\n", want: "TAP_QUARANTINE_POLL_INTERVAL"},
		{override: "TAP_QUARANTINE_LEASE_DURATION=0s\n", want: "TAP_QUARANTINE_LEASE_DURATION"},
		{override: "TAP_QUARANTINE_OPERATION_TIMEOUT=0s\n", want: "TAP_QUARANTINE_OPERATION_TIMEOUT"},
		{override: "TAP_QUARANTINE_BATCH_SIZE=0\n", want: "TAP_QUARANTINE_BATCH_SIZE"},
		{
			override: "TAP_QUARANTINE_LEASE_DURATION=10s\nTAP_QUARANTINE_OPERATION_TIMEOUT=10s\n",
			want:     "TAP_QUARANTINE_OPERATION_TIMEOUT must be shorter than TAP_QUARANTINE_LEASE_DURATION",
		},
	} {
		t.Run(test.want, func(t *testing.T) {
			_, err := LoadConfig(EnvDev, testConfigFile(t, base+test.override))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want %s", err, test.want)
			}
		})
	}
}
