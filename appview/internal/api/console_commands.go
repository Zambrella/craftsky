package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/moderation"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/safetyincident"
)

// ConsoleCommandHandler commits the existing domain operation and its replay
// receipt together. Authentication/capability checks must surround this handler,
// so replay never bypasses current operator authorization.
func ConsoleCommandHandler(pool *pgxpool.Pool, table, param, column string, handler func(pgx.Tx) http.Handler, rollback ...func(context.Context)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		actor, ok := ctxkeys.GetModerator(r.Context())
		if !ok {
			adminError(w, r, 401, "moderator_authentication_failed", "moderator authentication failed")
			return
		}
		key := r.Header.Get("Idempotency-Key")
		if len(r.Header.Values("Idempotency-Key")) != 1 || ValidateModerationIdempotencyKey(key) != nil {
			adminError(w, r, 400, "invalid_idempotency_key", "invalid Idempotency-Key")
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			adminError(w, r, 400, "invalid_request", "invalid command body")
			return
		}
		var fields map[string]json.RawMessage
		if decodeConsoleCommandObject(body, &fields) != nil || fields == nil || r.URL.RawQuery != "" {
			adminError(w, r, 400, "invalid_request", "invalid command body")
			return
		}
		canonical, _ := json.Marshal(fields)
		fingerprint := sha256.Sum256(append([]byte(r.Method+" "+r.URL.Path+"\n"), canonical...))
		keyHash := sha256.Sum256([]byte(key))
		var expected *int64
		if raw, exists := fields["expectedRevision"]; exists {
			if json.Unmarshal(raw, &expected) != nil || expected == nil || *expected < 0 {
				adminError(w, r, 400, "invalid_request", "invalid expected revision")
				return
			}
			delete(fields, "expectedRevision")
		}
		if table != "" && expected == nil {
			adminError(w, r, 400, "invalid_request", "expected revision required")
			return
		}
		if pool == nil {
			adminError(w, r, 503, "safety_command_unavailable", "safety command unavailable")
			return
		}
		tx, err := pool.Begin(r.Context())
		if err != nil {
			consoleCommandFailure(w, r, err)
			return
		}
		defer func() {
			if err := tx.Rollback(context.WithoutCancel(r.Context())); err == nil {
				for _, cleanup := range rollback {
					cleanup(context.WithoutCancel(r.Context()))
				}
			}
		}()
		// One key can create at most one outcome, including concurrent first attempts.
		if _, err = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, actor.ActorID+":"+key); err != nil {
			consoleCommandFailure(w, r, err)
			return
		}
		var priorHash, priorBody []byte
		var priorStatus int
		err = tx.QueryRow(r.Context(), `SELECT request_hash,response_status,response_body FROM operator_command_receipts WHERE actor_id=$1 AND key_hash=$2`, actor.ActorID, keyHash[:]).Scan(&priorHash, &priorStatus, &priorBody)
		if err == nil {
			if !bytes.Equal(priorHash, fingerprint[:]) {
				adminError(w, r, 409, "idempotency_conflict", "idempotency conflict")
				return
			}
			w.Header().Set("Idempotent-Replayed", "true")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(priorStatus)
			_, _ = w.Write(priorBody)
			return
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			consoleCommandFailure(w, r, err)
			return
		}
		var revision int64
		var id uuid.UUID
		var resourceKey any
		if table != "" {
			if table == "account_age_eligibility" {
				var raw string
				if json.Unmarshal(fields["accountDid"], &raw) != nil {
					adminError(w, r, 400, "invalid_request", "invalid account reference")
					return
				}
				did, parseErr := syntax.ParseDID(raw)
				if parseErr != nil {
					adminError(w, r, 400, "invalid_request", "invalid account reference")
					return
				}
				resourceKey = did
				if _, err = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "eligibility:"+did.String()); err != nil {
					consoleCommandFailure(w, r, err)
					return
				}
			} else {
				id, err = uuid.Parse(r.PathValue(param))
				if err != nil {
					adminError(w, r, 400, "invalid_request", "invalid resource reference")
					return
				}
				resourceKey = id
			}
			err = tx.QueryRow(r.Context(), `SELECT revision FROM `+pgx.Identifier{table}.Sanitize()+` WHERE `+pgx.Identifier{column}.Sanitize()+`=$1 FOR UPDATE`, resourceKey).Scan(&revision)
			if errors.Is(err, pgx.ErrNoRows) && table == "account_age_eligibility" {
				err = nil
				revision = 0
			}
			if errors.Is(err, pgx.ErrNoRows) {
				adminError(w, r, 404, "not_found", "resource not found")
				return
			}
			if err != nil {
				consoleCommandFailure(w, r, err)
				return
			}
			if revision != *expected {
				adminError(w, r, 409, "resource_revision_conflict", "resource revision conflict")
				return
			}
		}
		body, _ = json.Marshal(fields)
		request := r.Clone(r.Context())
		request.Body = io.NopCloser(bytes.NewReader(body))
		buffer := &consoleCommandResponse{header: make(http.Header)}
		handler(tx).ServeHTTP(buffer, request)
		if buffer.status >= 400 {
			for key, values := range buffer.header {
				w.Header()[key] = values
			}
			w.WriteHeader(buffer.status)
			_, _ = w.Write(buffer.body.Bytes())
			return
		}
		if buffer.status == 0 {
			buffer.status = 200
		}
		if table != "" {
			if err = tx.QueryRow(r.Context(), `SELECT revision FROM `+pgx.Identifier{table}.Sanitize()+` WHERE `+pgx.Identifier{column}.Sanitize()+`=$1`, resourceKey).Scan(&revision); err != nil {
				consoleCommandFailure(w, r, err)
				return
			}
		}
		if table != "" && revision == *expected {
			if err = tx.QueryRow(r.Context(), `UPDATE `+pgx.Identifier{table}.Sanitize()+` SET revision=revision WHERE `+pgx.Identifier{column}.Sanitize()+`=$1 RETURNING revision`, resourceKey).Scan(&revision); err != nil {
				consoleCommandFailure(w, r, err)
				return
			}
		}
		result := map[string]any{}
		if buffer.body.Len() > 0 && json.Unmarshal(buffer.body.Bytes(), &result) != nil {
			consoleCommandFailure(w, r, errors.New("invalid safe command response"))
			return
		}
		result["revision"] = revision
		response, _ := json.Marshal(result)
		response = append(response, '\n')
		status := buffer.status
		if status == 204 {
			status = 200
		}
		if _, err = tx.Exec(r.Context(), `INSERT INTO operator_command_receipts(actor_id,key_hash,request_hash,response_status,response_body,operation,resource_type,resource_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, actor.ActorID, keyHash[:], fingerprint[:], status, response, r.Pattern, table, consoleReceiptResource(id, result)); err != nil {
			consoleCommandFailure(w, r, err)
			return
		}
		if err = tx.Commit(r.Context()); err != nil {
			if errors.Is(err, pgx.ErrTxCommitRollback) {
				for _, cleanup := range rollback {
					cleanup(context.WithoutCancel(r.Context()))
				}
			}
			consoleCommandFailure(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(response)
	})
}

type consoleCommandResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (w *consoleCommandResponse) Header() http.Header { return w.header }
func (w *consoleCommandResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}
func (w *consoleCommandResponse) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	return w.body.Write(data)
}
func consoleCommandFailure(w http.ResponseWriter, r *http.Request, err error) {
	observability.ReportRequestFailure(r.Context(), err, "api.ConsoleCommandHandler", "transaction")
	adminError(w, r, 503, "safety_command_unavailable", "safety command unavailable")
}

func AtomicSafetyCommandHandler(pool *pgxpool.Pool, services SafetyCommandServices, operation, table, param, column string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var evidence *safetyincident.EvidenceService
		handler := ConsoleCommandHandler(pool, table, param, column, func(tx pgx.Tx) http.Handler {
			copy := services
			copy.Workflows = services.Workflows.InTransaction(tx)
			copy.CSEA = services.CSEA.InTransaction(tx)
			copy.Holds = services.Holds.InTransaction(tx)
			copy.Evidence = services.Evidence.InTransaction(tx)
			evidence = copy.Evidence
			return SafetyCommandHandler(copy, operation)
		}, func(ctx context.Context) {
			if evidence != nil {
				evidence.AbortPreservations(ctx)
			}
		})
		handler.ServeHTTP(w, r)
	})
}

func consoleReceiptResource(id uuid.UUID, result map[string]any) any {
	if id != uuid.Nil {
		return id
	}
	if raw, ok := result["id"].(string); ok {
		if parsed, err := uuid.Parse(raw); err == nil {
			return parsed
		}
	}
	if raw, ok := result["caseReference"].(string); ok {
		if parsed, err := moderation.ParseCaseReference(raw); err == nil {
			return parsed.UUID()
		}
	}
	return nil
}

func decodeConsoleCommandObject(data []byte, fields *map[string]json.RawMessage) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return errors.New("invalid command object")
	}
	result := map[string]json.RawMessage{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		if !ok {
			return errors.New("invalid command key")
		}
		if _, exists := result[key]; exists {
			return errors.New("duplicate command key")
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return err
		}
		result[key] = value
	}
	if _, err := decoder.Token(); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("trailing command value")
	}
	*fields = result
	return nil
}
