package observability

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

func (o *Observer) ObserveAuthorityVerification(operation, result, reason string, duration time.Duration) {
	if o == nil {
		return
	}
	o.metricRecorder.AuthorityVerification(context.Background(), operation, result, reason, duration)
}

func (o *Observer) ObserveOAuthMetadataCache(ctx context.Context, stage, result string) {
	if o == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	o.metricRecorder.OAuthMetadataCache(ctx, safeOAuthMetadataStage(stage), safeOAuthMetadataCacheResult(result))
}

func (o *Observer) ObserveOAuthCleanup(
	ctx context.Context,
	credentialKind string,
	result string,
	reason string,
	duration time.Duration,
	attempt int,
) {
	if o == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	credentialKind = safeCleanupCredentialKind(credentialKind)
	result = safeCleanupResult(result)
	reason = safeCleanupReason(reason)
	o.metricRecorder.OAuthCleanup(ctx, credentialKind, result, reason, duration, attempt)
	level := slog.LevelInfo
	if result != "success" {
		level = slog.LevelWarn
	}
	logResult := result
	if logResult == "retry" || logResult == "discarded" {
		logResult = "error"
	}
	o.Log(ctx, level, "OAuth credential cleanup completed", EventContext{
		"operation": "oauth_cleanup",
		"result":    logResult,
		"reason":    reason,
		"retryable": result == "retry",
	})
}

func (o *Observer) ObserveRepositoryRepair(jobKind, result, reason string, duration time.Duration, attempt int) {
	if o == nil {
		return
	}
	o.metricRecorder.RepositoryRepair(context.Background(), jobKind, result, reason, duration, attempt)
}

func (o *Observer) ObserveRepositoryRepairQueue(pending int, oldestAge time.Duration, maxAttempts int, alert bool) {
	if o == nil {
		return
	}
	o.metricRecorder.RepositoryRepairQueue(context.Background(), pending, oldestAge, maxAttempts, alert)
}

func (o *Observer) ObserveRepositorySnapshotVerification(result, reason string, duration time.Duration) {
	if o == nil {
		return
	}
	o.metricRecorder.RepositorySnapshotVerification(context.Background(), result, reason, duration)
}

func (r *InMemoryMetricRecorder) AuthorityVerification(_ context.Context, operation, result, reason string, duration time.Duration) {
	attrs := authorityVerificationAttributes(operation, result, reason)
	r.record(MetricCall{Name: "craftsky_appview_authority_verifications_total", Kind: MetricKindCounter, Value: 1, Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_authority_verification_duration_seconds", Kind: MetricKindDistribution, Unit: "second", Value: nonNegativeDuration(duration).Seconds(), Attributes: attrs})
}

func (r *InMemoryMetricRecorder) OAuthMetadataCache(_ context.Context, stage, result string) {
	r.record(MetricCall{
		Name: "craftsky_appview_oauth_metadata_cache_requests_total", Kind: MetricKindCounter, Value: 1,
		Attributes: oauthMetadataCacheAttributes(stage, result),
	})
}

func (r *InMemoryMetricRecorder) OAuthCleanup(_ context.Context, credentialKind, result, reason string, duration time.Duration, attempt int) {
	attrs := oauthCleanupAttributes(credentialKind, result, reason)
	r.record(MetricCall{Name: "craftsky_appview_oauth_cleanups_total", Kind: MetricKindCounter, Value: 1, Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_oauth_cleanup_duration_seconds", Kind: MetricKindDistribution, Unit: "second", Value: nonNegativeDuration(duration).Seconds(), Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_oauth_cleanup_attempt", Kind: MetricKindDistribution, Unit: "attempt", Value: float64(max(attempt, 0)), Attributes: attrs})
}

func (r *InMemoryMetricRecorder) RepositoryRepair(_ context.Context, jobKind, result, reason string, duration time.Duration, attempt int) {
	attrs := repositoryRepairAttributes(jobKind, result, reason)
	r.record(MetricCall{Name: "craftsky_appview_repository_repairs_total", Kind: MetricKindCounter, Value: 1, Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_repository_repair_duration_seconds", Kind: MetricKindDistribution, Unit: "second", Value: nonNegativeDuration(duration).Seconds(), Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_repository_repair_attempt", Kind: MetricKindDistribution, Unit: "attempt", Value: float64(max(attempt, 0)), Attributes: attrs})
}

func (r *InMemoryMetricRecorder) RepositoryRepairQueue(_ context.Context, pending int, oldestAge time.Duration, maxAttempts int, alert bool) {
	attrs := map[string]string{"alert": strconv.FormatBool(alert)}
	r.record(MetricCall{Name: "craftsky_appview_repository_repair_pending", Kind: MetricKindGauge, Unit: "job", Value: float64(max(pending, 0)), Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_repository_repair_oldest_age_seconds", Kind: MetricKindGauge, Unit: "second", Value: nonNegativeDuration(oldestAge).Seconds(), Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_repository_repair_max_attempts", Kind: MetricKindGauge, Unit: "attempt", Value: float64(max(maxAttempts, 0)), Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_repository_repair_alert", Kind: MetricKindGauge, Value: boolMetricValue(alert), Attributes: attrs})
}

func (r *InMemoryMetricRecorder) RepositorySnapshotVerification(_ context.Context, result, reason string, duration time.Duration) {
	attrs := repositorySnapshotAttributes(result, reason)
	r.record(MetricCall{Name: "craftsky_appview_repository_snapshot_verifications_total", Kind: MetricKindCounter, Value: 1, Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_repository_snapshot_verification_duration_seconds", Kind: MetricKindDistribution, Unit: "second", Value: nonNegativeDuration(duration).Seconds(), Attributes: attrs})
}

func (r *sentryMetricRecorder) AuthorityVerification(ctx context.Context, operation, result, reason string, duration time.Duration) {
	attrs := authorityVerificationAttributes(operation, result, reason)
	r.count(ctx, "craftsky_appview_authority_verifications_total", 1, "", attrs)
	r.distribution(ctx, "craftsky_appview_authority_verification_duration_seconds", nonNegativeDuration(duration).Seconds(), "second", attrs)
}

func (r *sentryMetricRecorder) OAuthMetadataCache(ctx context.Context, stage, result string) {
	r.count(ctx, "craftsky_appview_oauth_metadata_cache_requests_total", 1, "", oauthMetadataCacheAttributes(stage, result))
}

func (r *sentryMetricRecorder) OAuthCleanup(ctx context.Context, credentialKind, result, reason string, duration time.Duration, attempt int) {
	attrs := oauthCleanupAttributes(credentialKind, result, reason)
	r.count(ctx, "craftsky_appview_oauth_cleanups_total", 1, "", attrs)
	r.distribution(ctx, "craftsky_appview_oauth_cleanup_duration_seconds", nonNegativeDuration(duration).Seconds(), "second", attrs)
	r.distribution(ctx, "craftsky_appview_oauth_cleanup_attempt", float64(max(attempt, 0)), "attempt", attrs)
}

func (r *sentryMetricRecorder) RepositoryRepair(ctx context.Context, jobKind, result, reason string, duration time.Duration, attempt int) {
	attrs := repositoryRepairAttributes(jobKind, result, reason)
	r.count(ctx, "craftsky_appview_repository_repairs_total", 1, "", attrs)
	r.distribution(ctx, "craftsky_appview_repository_repair_duration_seconds", nonNegativeDuration(duration).Seconds(), "second", attrs)
	r.distribution(ctx, "craftsky_appview_repository_repair_attempt", float64(max(attempt, 0)), "attempt", attrs)
}

func (r *sentryMetricRecorder) RepositoryRepairQueue(ctx context.Context, pending int, oldestAge time.Duration, maxAttempts int, alert bool) {
	attrs := map[string]string{"alert": strconv.FormatBool(alert)}
	r.gauge(ctx, "craftsky_appview_repository_repair_pending", float64(max(pending, 0)), "job", attrs)
	r.gauge(ctx, "craftsky_appview_repository_repair_oldest_age_seconds", nonNegativeDuration(oldestAge).Seconds(), "second", attrs)
	r.gauge(ctx, "craftsky_appview_repository_repair_max_attempts", float64(max(maxAttempts, 0)), "attempt", attrs)
	r.gauge(ctx, "craftsky_appview_repository_repair_alert", boolMetricValue(alert), "", attrs)
}

func (r *sentryMetricRecorder) RepositorySnapshotVerification(ctx context.Context, result, reason string, duration time.Duration) {
	attrs := repositorySnapshotAttributes(result, reason)
	r.count(ctx, "craftsky_appview_repository_snapshot_verifications_total", 1, "", attrs)
	r.distribution(ctx, "craftsky_appview_repository_snapshot_verification_duration_seconds", nonNegativeDuration(duration).Seconds(), "second", attrs)
}

func authorityVerificationAttributes(operation, result, reason string) map[string]string {
	return map[string]string{
		"operation": safeAuthorityOperation(operation),
		"result":    safeAuthorityResult(result),
		"reason":    safeAuthorityReason(reason),
	}
}

func oauthMetadataCacheAttributes(stage, result string) map[string]string {
	return map[string]string{
		"stage":  safeOAuthMetadataStage(stage),
		"result": safeOAuthMetadataCacheResult(result),
	}
}

func oauthCleanupAttributes(credentialKind, result, reason string) map[string]string {
	return map[string]string{
		"credential_kind": safeCleanupCredentialKind(credentialKind),
		"result":          safeCleanupResult(result),
		"reason":          safeCleanupReason(reason),
	}
}

func repositoryRepairAttributes(jobKind, result, reason string) map[string]string {
	return map[string]string{
		"job_kind": safeRepositoryJobKind(jobKind),
		"result":   safeRepositoryResult(result),
		"reason":   safeRepositoryReason(reason),
	}
}

func repositorySnapshotAttributes(result, reason string) map[string]string {
	return map[string]string{
		"result": safeSnapshotResult(result),
		"reason": safeRepositoryReason(reason),
	}
}

func safeAuthorityOperation(value string) string {
	switch strings.TrimSpace(value) {
	case "callback", "session_select", "write":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeOAuthMetadataStage(value string) string {
	switch strings.TrimSpace(value) {
	case "protected_resource", "authorization_server":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeOAuthMetadataCacheResult(value string) string {
	switch strings.TrimSpace(value) {
	case "hit", "miss", "coalesced", "error":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeAuthorityResult(value string) string {
	switch strings.TrimSpace(value) {
	case "success", "mismatch", "error":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeAuthorityReason(value string) string {
	switch strings.TrimSpace(value) {
	case "none", "did_mismatch", "pds_changed", "issuer_changed", "resolve_failed", "metadata_invalid", "already_stale":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeCleanupCredentialKind(value string) string {
	switch strings.TrimSpace(value) {
	case "parent", "callback":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeCleanupResult(value string) string {
	switch strings.TrimSpace(value) {
	case "success", "retry", "discarded", "error":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeCleanupReason(value string) string {
	switch strings.TrimSpace(value) {
	case "none", "dependency_unavailable", "timeout", "canceled", "invalid_credential",
		"attempts_exhausted", "retention_expired", "store_failed":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeRepositoryJobKind(value string) string {
	switch strings.TrimSpace(value) {
	case "tap_add_repo", "pds_reconcile":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeRepositoryResult(value string) string {
	switch strings.TrimSpace(value) {
	case "success", "retry", "error":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeSnapshotResult(value string) string {
	switch strings.TrimSpace(value) {
	case "success", "error":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeRepositoryReason(value string) string {
	switch strings.TrimSpace(value) {
	case "none", "remote_unavailable", "invalid_request", "source_unavailable", "source_invalid",
		"request_invalid", "download_failed", "download_incomplete", "download_oversized", "source_changed",
		"car_malformed", "car_ambiguous_root", "repository_invalid", "commit_wrong_did", "commit_invalid",
		"signing_key_invalid", "signature_invalid", "mst_invalid", "commit_wrong_root", "registry_invalid",
		"lease_lost", "store_failed":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeMigrationReason(value string) string {
	if reason := safeAuthorityReason(value); reason != "unknown" {
		return reason
	}
	if reason := safeCleanupReason(value); reason != "unknown" {
		return reason
	}
	return safeRepositoryReason(value)
}

func boolMetricValue(value bool) float64 {
	if value {
		return 1
	}
	return 0
}
