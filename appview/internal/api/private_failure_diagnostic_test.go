package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/relationships"
	"social.craftsky/appview/internal/scheduledposts"
)

const auditPrivateCanary = "PRIVATE_PAYLOAD_TOKEN_RECIPIENT_CANARY"
const auditRequestID = "10000000-0000-4000-8000-000000000001"

func auditPrivateFailure() error {
	return observability.WrapError("private operation failed", &pgconn.PgError{Code: "08006", Message: auditPrivateCanary, Detail: auditPrivateCanary})
}

func auditPrivateRequest(t *testing.T, method, body string) (*http.Request, *slog.Logger, *bytes.Buffer, *observability.Observer, *sentry.MockTransport) {
	t.Helper()
	local := &bytes.Buffer{}
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(local, nil)))
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{Env: "test", Logger: logger, LogsEnabled: true, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	request := httptest.NewRequest(method, "/v1/private/"+auditPrivateCanary, strings.NewReader(body))
	ctx := middleware.WithDID(request.Context(), syntax.DID("did:plc:initiating-account"))
	ctx = middleware.WithDeviceID(ctx, auditPrivateCanary)
	ctx = ctxkeys.WithRunID(ctx, auditRequestID)
	ctx = observability.WithRequestDiagnosticContext(ctx, observability.RequestDiagnosticContext{Path: "/v1/private/[REDACTED]"})
	ctx = observability.WithRequestObserver(observability.WithCaptureMarker(ctx), observer)
	return request.WithContext(ctx), logger, local, observer, transport
}

func assertAuditPrivateDiagnostics(t *testing.T, local *bytes.Buffer, observer *observability.Observer, transport *sentry.MockTransport, response *httptest.ResponseRecorder, operation, stage string, expectedIssues int) {
	t.Helper()
	if !observer.Flush(time.Second) {
		t.Fatal("flush returned false")
	}
	issues := 0
	for _, event := range transport.Events() {
		if len(event.Exception) == 0 {
			continue
		}
		issues++
		if event.Tags["operation"] != operation || event.Tags["failure_stage"] != stage {
			t.Fatalf("wrong issue operation/stage: %+v", event.Tags)
		}
		if event.Contexts["correlation"]["run_id"] != auditRequestID {
			t.Fatalf("request ID lost: %+v", event.Contexts)
		}
		if event.Contexts["diagnostic"]["operation_account_did"] != "did:plc:initiating-account" {
			t.Fatalf("initiating account lost: %+v", event.Contexts)
		}
	}
	if issues != expectedIssues {
		t.Fatalf("issues=%d, want %d", issues, expectedIssues)
	}
	serialized, err := json.Marshal(transport.Events())
	if err != nil {
		t.Fatal(err)
	}
	if expectedIssues > 0 {
		for _, sink := range []string{local.String(), string(serialized)} {
			for _, positive := range []string{"pgconn.PgError", "08006", operation, stage, auditRequestID} {
				if !strings.Contains(sink, positive) {
					t.Fatalf("diagnostics missing %q: %s", positive, sink)
				}
			}
		}
	}
	if strings.Contains(local.String()+string(serialized)+response.Body.String(), auditPrivateCanary) {
		t.Fatal("private dependency prose or routing escaped")
	}
}

func TestPrivateAPIErrorWritersRetainOriginalCause(t *testing.T) {
	cases := []struct {
		name, operation, stage string
		status                 int
		write                  func(http.ResponseWriter, *http.Request, *slog.Logger, error)
	}{
		{"instagram_import", "instagram.import", "store", 500, writeInstagramImportError},
		{"instagram_suggestion", "instagram.suggestion", "store_or_effect", 503, writeInstagramSuggestionError},
		{"instagram_verification", "instagram.verification", "store", 500, writeInstagramVerificationError},
		{"profile_pin", "profile.pin.mutation", "store", 500, func(w http.ResponseWriter, r *http.Request, _ *slog.Logger, err error) {
			writeProfilePinMutationError(w, r, err)
		}},
		{"safety_intake", "safety.intake", "store", 503, func(w http.ResponseWriter, r *http.Request, _ *slog.Logger, err error) {
			writeSafetyIntakeError(w, r, err)
		}},
		{"moderation_command", "moderation.command", "store", 500, func(w http.ResponseWriter, r *http.Request, _ *slog.Logger, err error) {
			writeModerationCommandError(w, r, err)
		}},
		{"account_deletion", "account.deletion.accept", "store", 500, func(w http.ResponseWriter, r *http.Request, _ *slog.Logger, err error) {
			writeAccountDeletionError(w, r, err, "account.deletion.accept")
		}},
		{"billing_account", "billing.account", "store", 500, func(w http.ResponseWriter, r *http.Request, _ *slog.Logger, err error) {
			writeBillingAccountError(w, r, err)
		}},
		{"subscription_assignment", "subscription.assignment", "store", 500, func(w http.ResponseWriter, r *http.Request, _ *slog.Logger, err error) {
			writeAssignmentError(w, r, err)
		}},
		{"post_interaction_list", "post.interactions.likes", "store", 500, func(w http.ResponseWriter, r *http.Request, _ *slog.Logger, err error) {
			writePostInteractionListError(w, r, err, "post.interactions.likes")
		}},
		{"scheduled_post", "scheduled.post", "request", 500, func(w http.ResponseWriter, r *http.Request, _ *slog.Logger, err error) {
			writeScheduledPostError(w, r, err, "scheduled.post", "request")
		}},
		{"scheduled_media", "scheduled.media", "request", 500, func(w http.ResponseWriter, r *http.Request, _ *slog.Logger, err error) {
			writeScheduledMediaError(w, r, err, "scheduled.media", "request")
		}},
		{"scheduled_object_store", "scheduled.media", "object_store", 503, func(w http.ResponseWriter, r *http.Request, _ *slog.Logger, err error) {
			writeScheduledMediaError(w, r, errors.Join(scheduledposts.ErrPrivateObjectStoreUnavailable, err), "scheduled.media", "store")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request, logger, local, observer, transport := auditPrivateRequest(t, http.MethodPost, "")
			response := httptest.NewRecorder()
			tc.write(response, request, logger, auditPrivateFailure())
			if response.Code != tc.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			assertAuditPrivateDiagnostics(t, local, observer, transport, response, tc.operation, tc.stage, 1)
		})
	}
}

func TestPrivateAPIExpectedFailuresDoNotCaptureIssues(t *testing.T) {
	cases := []struct {
		name  string
		write func(http.ResponseWriter, *http.Request, *slog.Logger)
	}{
		{"import_not_found", func(w http.ResponseWriter, r *http.Request, l *slog.Logger) {
			writeInstagramImportError(w, r, l, instagram.ErrInstagramResourceNotFound)
		}},
		{"suggestion_ineligible", func(w http.ResponseWriter, r *http.Request, l *slog.Logger) {
			writeInstagramSuggestionError(w, r, l, instagram.ErrSuggestionIneligible)
		}},
		{"verification_conflict", func(w http.ResponseWriter, r *http.Request, l *slog.Logger) {
			writeInstagramVerificationError(w, r, l, instagram.ErrInstagramLinkConflict)
		}},
		{"scheduled_capacity", func(w http.ResponseWriter, r *http.Request, _ *slog.Logger) {
			writeScheduledPostError(w, r, scheduledposts.ErrCapacityReached, "scheduled.post", "request")
		}},
		{"scheduled_media_not_found", func(w http.ResponseWriter, r *http.Request, _ *slog.Logger) {
			writeScheduledMediaError(w, r, scheduledposts.ErrScheduledMediaNotFound, "scheduled.media", "request")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request, logger, local, observer, transport := auditPrivateRequest(t, http.MethodPost, "")
			response := httptest.NewRecorder()
			tc.write(response, request, logger)
			if response.Code < 400 || response.Code >= 500 {
				t.Fatalf("expected client error, got %d", response.Code)
			}
			assertAuditPrivateDiagnostics(t, local, observer, transport, response, "", "", 0)
		})
	}
}

type auditPreferenceStore struct {
	err   error
	calls int
}

func (s *auditPreferenceStore) NotificationPreferences(context.Context, string) (map[notifications.Category]notifications.Preference, error) {
	s.calls++
	return nil, s.err
}
func (s *auditPreferenceStore) PatchNotificationPreferences(context.Context, string, map[notifications.Category]notifications.PreferencePatch) (map[notifications.Category]notifications.Preference, error) {
	s.calls++
	return nil, s.err
}

type auditDeviceStore struct{ err error }

func (s auditDeviceStore) RegisterNotificationDevice(context.Context, string, string, string, string) (string, error) {
	return "", s.err
}
func (s auditDeviceStore) RemoveNotificationSubscription(context.Context, string, string, string) error {
	return s.err
}

func TestNotificationAPIStoreFailuresAreServerFailuresWithOriginalCause(t *testing.T) {
	cases := []struct {
		name, method, body, operation string
		handler                       func(error, *slog.Logger) http.Handler
	}{
		{"preferences_read", http.MethodGet, "", "notification.preferences.read", func(err error, l *slog.Logger) http.Handler {
			return GetNotificationPreferencesHandler(&auditPreferenceStore{err: err}, l)
		}},
		{"preferences_patch", http.MethodPatch, `{"preferences":{"like":{"pushEnabled":false}}}`, "notification.preferences.update", func(err error, l *slog.Logger) http.Handler {
			return PatchNotificationPreferencesHandler(&auditPreferenceStore{err: err}, l)
		}},
		{"device_register", http.MethodPost, `{"platform":"ios","token":"` + auditPrivateCanary + `"}`, "notification.device.register", func(err error, l *slog.Logger) http.Handler {
			return RegisterNotificationDeviceHandler(auditDeviceStore{err}, l)
		}},
		{"device_remove", http.MethodDelete, "", "notification.device.remove", func(err error, l *slog.Logger) http.Handler {
			return RemoveNotificationDeviceHandler(auditDeviceStore{err}, l)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request, logger, local, observer, transport := auditPrivateRequest(t, tc.method, tc.body)
			request.SetPathValue("accountSubscriptionId", auditPrivateCanary)
			response := httptest.NewRecorder()
			tc.handler(auditPrivateFailure(), logger).ServeHTTP(response, request)
			if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "internal_error") {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			assertAuditPrivateDiagnostics(t, local, observer, transport, response, tc.operation, "store", 1)
		})
	}
}

func TestNotificationPreferenceInputValidationPrecedesStorage(t *testing.T) {
	for _, patch := range []string{`{"unknown":{"pushEnabled":false}}`, `{"like":{"scope":"invalid"}}`, `{"moderation":{"scope":"everyone"}}`} {
		request, logger, local, observer, transport := auditPrivateRequest(t, http.MethodPatch, `{"preferences":`+patch+`}`)
		store := &auditPreferenceStore{err: auditPrivateFailure()}
		response := httptest.NewRecorder()
		PatchNotificationPreferencesHandler(store, logger).ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "invalid_preferences") || store.calls != 0 {
			t.Fatalf("validation status=%d body=%s storage calls=%d", response.Code, response.Body.String(), store.calls)
		}
		assertAuditPrivateDiagnostics(t, local, observer, transport, response, "", "", 0)
	}
}

type auditRejectedSetCommands struct{ cause error }

func (commands auditRejectedSetCommands) Execute(_ context.Context, request pdscommands.SetCommandRequest) (pdscommands.CommandResult, error) {
	return pdscommands.CommandResult{TerminalResult: request.Rejected(commands.cause)}, nil
}

func TestSetCommandRejectionsRetainOriginalCauseBeforePersistence(t *testing.T) {
	owner, target := syntax.DID("did:plc:initiating-account"), syntax.DID("did:plc:public-target")
	key := uuid.MustParse("018f4d5c-7a61-7d40-a1a2-0123456789ab")
	ref := &PostTargetRef{URI: "at://did:plc:public-target/social.craftsky.feed.post/one", CID: "bafyPost"}
	cases := []struct {
		name    string
		execute func(*http.Request, SetCommandExecutor, bool) (pdscommands.CommandResult, error)
	}{
		{"like", func(r *http.Request, commands SetCommandExecutor, active bool) (pdscommands.CommandResult, error) {
			return executeLikeCommand(r, commands, owner, 1, target, ref, key, "", active, auditRequestID)
		}},
		{"repost", func(r *http.Request, commands SetCommandExecutor, active bool) (pdscommands.CommandResult, error) {
			return executeRepostCommand(r, commands, owner, 1, target, ref, key, "", active, auditRequestID)
		}},
		{"follow", func(r *http.Request, commands SetCommandExecutor, active bool) (pdscommands.CommandResult, error) {
			return executeFollowCommand(r, commands, owner, 1, target, key, "", active, json.RawMessage(`{}`), auditRequestID)
		}},
		{"block", func(r *http.Request, commands SetCommandExecutor, active bool) (pdscommands.CommandResult, error) {
			return executeBlockCommand(r, commands, owner, 1, target, key, "", active, relationships.State{}, auditRequestID)
		}},
	}
	for _, tc := range cases {
		for _, active := range []bool{false, true} {
			for _, captured := range []bool{false, true} {
				name := tc.name
				if !active {
					name = "un" + name
				}
				if captured {
					name += "_already_owned"
				}
				t.Run(name, func(t *testing.T) {
					request, _, local, observer, transport := auditPrivateRequest(t, http.MethodPost, "")
					if captured {
						observability.MarkCaptured(request.Context())
					}
					result, err := tc.execute(request, auditRejectedSetCommands{cause: errors.Join(pdscommands.ErrDispatchRejected, auditPrivateFailure())}, active)
					if err != nil || result.State != pdscommands.CommandRejected || result.HTTPStatus != http.StatusBadGateway {
						t.Fatalf("rejection changed: %+v err=%v", result, err)
					}
					response := httptest.NewRecorder()
					WriteCommandResponse(response, CommandResultFromStored(result))
					wantIssues := 1
					if captured {
						wantIssues = 0
					}
					assertAuditPrivateDiagnostics(t, local, observer, transport, response, "pds.command", "rejection", wantIssues)
				})
			}
		}
	}
}

func TestScheduledPublicationMalformedInputIsAClientError(t *testing.T) {
	for _, body := range []string{`{"payload":`, `{"payload":{"kind":"standard","text":"` + auditPrivateCanary + `"}}`} {
		request, logger, local, observer, transport := auditPrivateRequest(t, http.MethodPost, body)
		request.SetPathValue("id", "00000000-0000-4000-8000-000000000641")
		publisher := &recordingManualScheduledPublisher{}
		response := httptest.NewRecorder()
		PublishScheduledPostHandler(publisher, DefaultMediaLimits(), time.Now, logger).ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "malformed_body") || publisher.params.ID != uuid.Nil {
			t.Fatalf("malformed publication status=%d body=%s params=%+v", response.Code, response.Body.String(), publisher.params)
		}
		assertAuditPrivateDiagnostics(t, local, observer, transport, response, "", "", 0)
	}
}
