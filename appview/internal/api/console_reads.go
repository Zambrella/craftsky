package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/moderation"
	"social.craftsky/appview/internal/observability"
)

// Queries select an explicit wire allowlist. Storage rows must never be encoded
// wholesale: they also contain private object keys, credentials and raw routing.
var consoleReadQueries = map[string]string{
	"operations": `SELECT id,created_at,resource_type state,jsonb_build_object('id',id,'actorId',actor_id,'operation',operation,'resourceType',resource_type,'resourceId',resource_id,'status',response_status,'createdAt',created_at) item FROM operator_command_receipts`,

	"incidents":        `SELECT i.id,i.detected_at created_at,i.state,jsonb_build_object('id',i.id,'revision',i.revision,'kind',i.incident_kind,'state',i.state,'priority',i.priority,'owner',i.assigned_to,'deadlineAt',i.reporting_deadline,'detectedAt',i.detected_at,'cover',(SELECT 'operator:'||a.operator_id FROM safety_operator_assignments a WHERE a.incident_id=i.id AND a.assignment_role='cover' AND a.released_at IS NULL ORDER BY a.assigned_at DESC,a.id DESC LIMIT 1)) item FROM safety_incidents i`,
	"external-intakes": `SELECT id,created_at,intake_kind state,jsonb_build_object('id',id,'revision',revision,'reference',reference,'kind',intake_kind,'urgency',urgency,'subject',canonical_subject,'owner',owner_actor_id,'attachmentStatus',attachment_status,'receivedAt',received_at) item FROM external_safety_intakes`,
	"evidence":         `SELECT e.id,e.created_at,CASE WHEN e.deleted_at IS NULL THEN 'retained' ELSE 'deleted' END state,jsonb_build_object('id',e.id,'revision',e.revision,'incidentId',e.incident_id,'contentType',e.content_type,'byteSize',e.byte_size,'retentionExpiresAt',e.retention_expires_at,'deletedAt',e.deleted_at,'createdAt',e.created_at,'held',EXISTS(SELECT 1 FROM safety_legal_hold_evidence he JOIN safety_legal_holds h ON h.id=he.hold_id WHERE he.evidence_id=e.id AND h.released_at IS NULL AND h.expires_at>now())) item FROM safety_evidence e`,
	"holds":            `SELECT id,created_at,CASE WHEN released_at IS NOT NULL THEN 'released' WHEN expires_at<=now() THEN 'expired' ELSE 'active' END state,jsonb_build_object('id',id,'revision',revision,'incidentId',incident_id,'createdBy',created_by,'approvedBy',approved_by,'expiresAt',expires_at,'releasedAt',released_at,'createdAt',created_at) item FROM safety_legal_holds`,
	"workflows":        `SELECT id,created_at,state,jsonb_build_object('revision',revision,'id',id,'kind',workflow_class,'state',state,'urgency',urgency,'owner',responsible_actor,'receivedAt',received_at,'deadlineAt',deadline_at,'updatedAt',updated_at) item FROM safety_workflows`,
	"scans":            `SELECT r.id,r.created_at,r.state,jsonb_build_object('id',r.id,'state',r.state,'scannerId',r.scanner_id,'policyVersion',r.policy_version,'corpusVersion',r.corpus_version,'revision',j.revision,'version',r.current_version,'jobState',j.state,'attempts',j.attempts,'nextAttemptAt',j.next_attempt_at,'errorCategory',j.safe_error_category,'updatedAt',r.updated_at) item FROM image_scan_results r LEFT JOIN image_scan_jobs j ON j.scan_result_id=r.id`,
}

var consoleScopedReads = map[string]struct{ query, column string }{
	"case-provenance":       {`SELECT id,created_at,'' state,case_id::text scope,jsonb_build_object('id',id,'intakeId',intake_id,'actorId',actor_id,'sourceSystem',source_system,'reason',reason,'createdAt',created_at) item FROM moderation_case_provenance`, "caseReference"},
	"incident-subjects":     {`SELECT md5(incident_id::text||subject_uri||image_slot)::uuid id,linked_at created_at,'' state,incident_id::text scope,jsonb_build_object('subjectUri',subject_uri,'subjectCid',source_cid,'kind',subject_kind,'imageSlot',image_slot) item FROM safety_incident_subjects`, "incidentReference"},
	"incident-reports":      {`SELECT id,submitted_at created_at,report_kind state,incident_id::text scope,jsonb_build_object('id',id,'kind',report_kind,'submittedBy',submitted_by,'submittedAt',submitted_at,'deadlineMet',deadline_met,'retentionUntil',retention_until) item FROM safety_authority_reports`, "incidentReference"},
	"information-requests":  {`SELECT id,received_at created_at,state,incident_id::text scope,jsonb_build_object('id',id,'revision',revision,'state',state,'dueAt',due_at,'receivedAt',received_at,'respondedAt',responded_at) item FROM safety_authority_information_requests`, "incidentReference"},
	"workflow-events":       {`SELECT id,occurred_at created_at,event_type state,workflow_id::text scope,jsonb_build_object('id',id,'sequence',sequence,'type',event_type,'actorId',actor_id,'detailCode',detail_code,'occurredAt',occurred_at) item FROM safety_workflow_events`, "workflowId"},
	"incident-events":       {`SELECT id,created_at,event_type state,incident_id::text scope,jsonb_build_object('id',id,'type',event_type,'actorId',actor_id,'sourceSystem',source_system,'createdAt',created_at) item FROM safety_incident_events`, "incidentReference"},
	"intake-correspondence": {`SELECT id,created_at,'' state,intake_id::text scope,jsonb_build_object('id',id,'attachmentStatus',attachment_status,'receivedAt',received_at,'createdAt',created_at) item FROM external_safety_correspondence`, "intakeId"},
	"account-history":       {`SELECT id,created_at,state,owner_did scope,jsonb_build_object('caseReference','MOD-'||id::text,'state',state,'revision',revision,'subjectType',subject_type,'origin',origin,'createdAt',created_at) item FROM moderation_cases`, "did"},
}

type consoleReadCursor struct {
	Scope    string    `json:"scope"`
	Resource string    `json:"resource"`
	State    string    `json:"state"`
	Kind     string    `json:"kind"`
	JobState string    `json:"jobState"`
	At       time.Time `json:"at"`
	ID       uuid.UUID `json:"id"`
}

func ConsoleListHandler(pool *pgxpool.Pool, resource string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := 50
		for key, values := range r.URL.Query() {
			if (key != "limit" && key != "cursor" && key != "state" && key != "kind" && key != "incidentId" && key != "jobState") || len(values) != 1 {
				adminError(w, r, 400, "invalid_request", "invalid list parameters")
				return
			}
		}
		if raw := r.URL.Query().Get("limit"); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value < 1 || value > 100 {
				adminError(w, r, 400, "invalid_request", "invalid limit")
				return
			}
			limit = value
		}
		scope := ""
		scoped, hasScope := consoleScopedReads[resource]
		if hasScope {
			scope = r.PathValue(scoped.column)
			if scoped.column == "caseReference" {
				reference, err := moderation.ParseCaseReference(scope)
				if err != nil {
					adminError(w, r, 400, "invalid_request", "invalid case reference")
					return
				}
				scope = reference.UUID().String()
			} else if scoped.column == "did" {
				if _, err := syntax.ParseDID(scope); err != nil {
					adminError(w, r, 400, "invalid_request", "invalid account reference")
					return
				}
			} else {
				parsed, err := uuid.Parse(scope)
				if err != nil {
					adminError(w, r, 400, "invalid_request", "invalid resource reference")
					return
				}
				scope = parsed.String()
			}
		}

		kind := r.URL.Query().Get("kind")
		jobState := r.URL.Query().Get("jobState")
		if len(kind) > 32 || len(jobState) > 32 || kind != "" && resource != "workflows" && resource != "external-intakes" && resource != "incidents" || jobState != "" && resource != "scans" {
			adminError(w, r, 400, "invalid_request", "invalid list filter")
			return
		}
		if raw := r.URL.Query().Get("incidentId"); raw != "" {
			id, err := uuid.Parse(raw)
			if err != nil || (resource != "evidence" && resource != "holds") {
				adminError(w, r, 400, "invalid_request", "invalid incident filter")
				return
			}
			scope = id.String()
		}
		state := r.URL.Query().Get("state")
		if len(state) > 32 {
			adminError(w, r, 400, "invalid_request", "invalid state")
			return
		}
		var cursor consoleReadCursor
		if raw := r.URL.Query().Get("cursor"); raw != "" {
			data, err := base64.RawURLEncoding.DecodeString(raw)
			decoder := json.NewDecoder(bytes.NewReader(data))
			decoder.DisallowUnknownFields()
			if err != nil || len(raw) > 1024 || decoder.Decode(&cursor) != nil || cursor.Resource != resource || cursor.Scope != scope || cursor.Kind != kind || cursor.JobState != jobState || cursor.State != state || cursor.At.IsZero() || cursor.ID == uuid.Nil || !errors.Is(decoder.Decode(&struct{}{}), io.EOF) {
				adminError(w, r, 400, "invalid_cursor", "invalid cursor")
				return
			}
		}
		query, ok := consoleReadQueries[resource]
		if hasScope {
			query = scoped.query
			ok = true
		} else {
			query = `SELECT *,item->>'incidentId' scope FROM (` + query + `) unscoped`
		}
		if pool == nil || !ok {
			adminError(w, r, 503, "admin_read_unavailable", "admin read unavailable")
			return
		}
		var at any
		if !cursor.At.IsZero() {
			at = cursor.At
		}
		rows, err := pool.Query(r.Context(), `SELECT id,created_at,item FROM (`+query+`) q WHERE ($5='' OR scope=$5) AND ($6='' OR item->>'kind'=$6) AND ($7='' OR item->>'jobState'=$7) AND ($1='' OR state=$1) AND ($2::timestamptz IS NULL OR (created_at,id)<($2,$3)) ORDER BY created_at DESC,id DESC LIMIT $4`, state, at, cursor.ID, limit+1, scope, kind, jobState)
		if err != nil {
			consoleReadError(w, r, err)
			return
		}
		defer rows.Close()
		items := []json.RawMessage{}
		keys := []consoleReadCursor{}
		for rows.Next() {
			var key consoleReadCursor
			var item json.RawMessage
			if err := rows.Scan(&key.ID, &key.At, &item); err != nil {
				consoleReadError(w, r, err)
				return
			}
			key.Kind = kind
			key.JobState = jobState
			key.Scope = scope
			key.Resource = resource
			key.State = state
			keys = append(keys, key)
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			consoleReadError(w, r, err)
			return
		}
		next := ""
		if len(items) > limit {
			items = items[:limit]
			data, _ := json.Marshal(keys[limit-1])
			next = base64.RawURLEncoding.EncodeToString(data)
		}
		writeSafetyCommandJSON(w, 200, struct {
			Items  []json.RawMessage `json:"items"`
			Cursor string            `json:"cursor,omitempty"`
		}{items, next})
	})
}

func ConsoleDetailHandler(pool *pgxpool.Pool, resource, param string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue(param))
		if err != nil {
			adminError(w, r, 400, "invalid_request", "invalid resource reference")
			return
		}
		query, ok := consoleReadQueries[resource]
		if pool == nil || !ok {
			adminError(w, r, 503, "admin_read_unavailable", "admin read unavailable")
			return
		}
		var item json.RawMessage
		err = pool.QueryRow(r.Context(), `SELECT item FROM (`+query+`) q WHERE id=$1`, id).Scan(&item)
		if err != nil {
			consoleReadError(w, r, err)
			return
		}
		writeSafetyCommandJSON(w, 200, item)
	})
}

func consoleReadError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		adminError(w, r, 404, "not_found", "resource not found")
		return
	}
	observability.ReportRequestFailure(r.Context(), fmt.Errorf("read operator resource: %w", err), "api.ConsoleReadHandler", "query")
	adminError(w, r, 503, "admin_read_unavailable", "admin read unavailable")
}
