// Package retoolproof hosts only synthetic fixtures for the Retool integration
// experiment. It is not registered in the AppView production application.
package retoolproof

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/moderation"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/safetyincident"
)

//go:embed openapi.json
var specification []byte

type Fixture struct {
	IncidentID    uuid.UUID
	CaseReference string
	EvidenceID    uuid.UUID
	Evidence      *safetyincident.EvidenceService
}

// Seed requires a freshly migrated, isolated database/schema. It never resets
// existing records. Credentials are supplied by the caller and stored as digests.
func Seed(ctx context.Context, pool *pgxpool.Pool, alphaToken, betaToken string) (Fixture, error) {
	if alphaToken == "" || betaToken == "" || alphaToken == betaToken {
		return Fixture{}, errors.New("distinct proof credentials required")
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	caseID := uuid.New()
	reference, _ := moderation.FormatCaseReference(caseID)
	incidentID, scanID := uuid.New(), uuid.New()
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		for _, operator := range []struct{ actor, token string }{{"proof-alpha", alphaToken}, {"proof-beta", betaToken}} {
			id := uuid.New()
			digest := sha256.Sum256([]byte(operator.token))
			role := safetyincident.RoleModerator
			var retoolID *int64
			if operator.actor == "proof-alpha" {
				role = safetyincident.RoleSafetyAdministrator
				binding := int64(2691189)
				retoolID = &binding
			}
			if _, err := tx.Exec(ctx, `INSERT INTO safety_operators(id,actor_id,role,retool_user_id,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$5)`, id, operator.actor, string(role), retoolID, now); err != nil {
				return err
			}
			for _, p := range []string{"moderation.read", "moderation.manage"} {
				if _, err := tx.Exec(ctx, `INSERT INTO safety_operator_permissions(operator_id,permission,granted_by,granted_at) VALUES($1,$2,'proof-fixture',$3)`, id, p, now); err != nil {
					return err
				}
			}
			if _, err := tx.Exec(ctx, `INSERT INTO safety_operator_tokens(id,operator_id,token_digest,expires_at,created_at) VALUES($1,$2,$3,$4,$5)`, uuid.New(), id, digest[:], now.Add(24*time.Hour), now); err != nil {
				return err
			}
			if operator.actor == "proof-alpha" {
				if _, err := tx.Exec(ctx, `INSERT INTO safety_operator_permissions(operator_id,permission,granted_by,granted_at) VALUES($1,'operator.manage','proof-fixture',$2)`, id, now); err != nil {
					return err
				}
				if _, err := tx.Exec(ctx, `INSERT INTO safety_operator_permissions(operator_id,permission,granted_by,granted_at) VALUES($1,'evidence.access','proof-fixture',$2)`, id, now); err != nil {
					return err
				}
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO moderation_reports(id,reporter_did,subject_type,subject_did,reason_type,details,forwarding_status,forwarding_prepared_at,created_at) VALUES('proof-report','did:example:synthetic-reporter','account','did:example:synthetic-owner','spam','Synthetic fixture only','prepared_not_submitted',$1,$1)`, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO moderation_cases(id,subject_key,subject_type,subject_did,owner_did,safe_snapshot,created_at,updated_at) VALUES($1,'account:did:example:synthetic-owner','account','did:example:synthetic-owner','did:example:synthetic-owner','{"type":"account","did":"did:example:synthetic-owner"}',$2,$2)`, caseID, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO moderation_case_reports(case_id,report_id,attached_at) VALUES($1,'proof-report',$2)`, caseID, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO image_scan_results(id,blob_cid,scanner_id,policy_version,corpus_version,state,created_at,updated_at) VALUES($1,'synthetic-fixture','proof-scanner','proof-policy','proof-corpus','match',$2,$2)`, scanID, now); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO safety_incidents(id,scan_result_id,provider_reference,integrity_metadata_reference,detected_at,updated_at) VALUES($1,$2,'synthetic-provider-reference','synthetic-integrity-reference',$3,$3)`, incidentID, scanID, now)
		return err
	})
	if err != nil {
		return Fixture{}, err
	}
	objects := safetyincident.NewMemoryEvidenceStore()
	evidence := safetyincident.NewEvidenceService(pool, objects, time.Now)
	// A benign one-pixel PNG, never real restricted content.
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+j0ioAAAAASUVORK5CYII=")
	if err != nil {
		return Fixture{}, err
	}
	ref, err := evidence.Preserve(ctx, safetyincident.Actor{ID: "proof-fixture", Role: safetyincident.RoleSafetyAdministrator}, safetyincident.PreserveCommand{IncidentID: incidentID, Bytes: png, SHA256: sha256.Sum256(png), ContentType: "image/png", Reason: "Benign synthetic transport fixture", ExpiresAt: now.Add(24 * time.Hour)})
	if err != nil {
		return Fixture{}, err
	}
	return Fixture{CaseReference: reference, EvidenceID: ref.ID, Evidence: evidence, IncidentID: incidentID}, nil
}

func New(pool *pgxpool.Pool, fixture Fixture, publicOrigin string) (http.Handler, error) {
	fencer, err := ownerlifecycle.NewFencer(pool, 5*time.Second)
	if err != nil {
		return nil, err
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		return nil, err
	}
	outputs, err := api.NewModerationStore(pool, lifecycles)
	if err != nil {
		return nil, err
	}
	store := moderation.NewStore(pool)
	commands := moderation.NewService(store, outputs, time.Now)
	// The normal middleware emits only its fixed authentication-failure message.
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	auth := middleware.ModeratorDatabaseAuthentication(middleware.NewPostgresModeratorAuthenticator(pool, "retool-proof", time.Now), logger)
	mux := http.NewServeMux()
	// Retool's generated OpenAPI methods omit header parameters. These two
	// proof-only adapters accept the key in JSON and forward it to the unchanged
	// real admin handler. No production route or idempotency rule is relaxed.
	for _, command := range []struct{ path, kind string }{{"decisions", "decision"}, {"effect-changes", "effectChange"}} {
		handler := api.RequireOperatorPermission(safetyincident.PermissionModerationManage)(api.ModerationAdminCommandHandler(commands, syntax.DID("did:example:synthetic-labeler"), command.kind))
		mux.Handle("POST /v1/admin/proof/"+command.path, auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var request struct {
				ExpectedRevision *int64 `json:"expectedRevision"`
				IdempotencyKey   string `json:"idempotencyKey"`
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if decoder.Decode(&request) != nil || decoder.Decode(&struct{}{}) != io.EOF || request.ExpectedRevision == nil {
				envelope.WriteError(w, 400, "invalid_request", "invalid proof command", ctxkeys.GetRunID(r.Context()), nil)
				return
			}
			body := map[string]any{"expectedRevision": *request.ExpectedRevision}
			if command.kind == "decision" {
				body["disposition"] = "violation"
				body["reason"] = "spam"
				body["evidenceNotes"] = "Synthetic fixture review"
				body["consequences"] = []string{"visibilityHide", "strike"}
			} else {
				body["negateEffects"] = []string{"visibilityHide", "strike"}
				body["rationale"] = "Synthetic reversal"
			}
			encoded, _ := json.Marshal(body)
			r.Body = io.NopCloser(bytes.NewReader(encoded))
			r.Header.Set("Idempotency-Key", request.IdempotencyKey)
			r.SetPathValue("caseReference", fixture.CaseReference)
			handler.ServeHTTP(w, r)
		})))
	}
	mux.Handle("GET /v1/admin/identity", auth(api.OperatorIdentityHandler()))
	mux.Handle("GET /v1/admin/moderation/cases", auth(api.RequireOperatorPermission(safetyincident.PermissionModerationRead)(api.ModerationAdminQueueHandler(store))))
	mux.Handle("GET /v1/admin/moderation/cases/{caseReference}", auth(api.RequireOperatorPermission(safetyincident.PermissionModerationRead)(api.ModerationAdminDetailHandler(store))))
	mux.Handle("GET /v1/admin/safety/incidents/{incidentReference}", auth(api.RequireOperatorPermission(safetyincident.PermissionIncidentReadSafe)(api.RequireSafetyPermission(safetyincident.PermissionIncidentReadSafe)(api.SafetyIncidentDetailHandler(safetyincident.NewStore(pool, time.Now))))))
	for _, command := range []struct{ path, kind string }{{"decisions", "decision"}, {"effect-changes", "effectChange"}} {
		mux.Handle("POST /v1/admin/moderation/cases/{caseReference}/"+command.path, auth(api.RequireOperatorPermission(safetyincident.PermissionModerationManage)(api.ModerationAdminCommandHandler(commands, syntax.DID("did:example:synthetic-labeler"), command.kind))))
	}
	mux.Handle("GET /v1/admin/proof/identity", auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, _ := ctxkeys.GetModerator(r.Context())
		envelope.WriteJSON(w, 200, map[string]any{"actorId": actor.ActorID, "role": actor.Role, "synthetic": true, "caseReference": fixture.CaseReference, "evidenceId": fixture.EvidenceID, "permissions": actor.Permissions})
	})))
	mux.Handle("POST /v1/admin/safety/evidence/{evidenceId}/access", auth(api.SafetyCommandHandler(api.SafetyCommandServices{Evidence: fixture.Evidence}, "accessEvidence")))
	mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, r *http.Request) {
		var spec map[string]any
		if json.Unmarshal(specification, &spec) != nil {
			envelope.WriteError(w, 500, "internal_error", "specification unavailable", ctxkeys.GetRunID(r.Context()), nil)
			return
		}
		spec["servers"] = []map[string]string{{"url": publicOrigin}}
		envelope.WriteJSON(w, 200, spec)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		envelope.WriteError(w, 404, "not_found", "proof endpoint not found", ctxkeys.GetRunID(r.Context()), nil)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		r = r.WithContext(ctxkeys.WithRunID(r.Context(), uuid.NewString()))
		mux.ServeHTTP(w, r)
	}), nil
}
