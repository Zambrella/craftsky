package moderation

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrCaseNotFound       = errors.New("moderation case not found")
	ErrInvalidQueueFilter = errors.New("invalid moderation queue filter")
	ErrInvalidQueueCursor = errors.New("invalid moderation queue cursor")
)

type QueueFilter struct {
	State, SubjectType, AppealStatus, Cursor string
	Limit                                    int
}

type QueueItem struct {
	CaseReference string      `json:"caseReference"`
	State         string      `json:"state"`
	SubjectType   SubjectType `json:"subjectType"`
	SubjectDID    syntax.DID  `json:"subjectDid"`
	ReportCount   int         `json:"reportCount"`
	Revision      int64       `json:"revision"`
	CreatedAt     time.Time   `json:"createdAt"`
}

type QueuePage struct {
	Items  []QueueItem `json:"items"`
	Cursor string      `json:"cursor,omitempty"`
}

type queueCursor struct {
	CreatedAt time.Time `json:"createdAt"`
	ID        uuid.UUID `json:"id"`
}

func (s *Store) Queue(ctx context.Context, filter QueueFilter) (QueuePage, error) {
	if filter.Limit == 0 {
		filter.Limit = 50
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		return QueuePage{}, ErrInvalidQueueFilter
	}
	if filter.State != "" && filter.State != "open" && filter.State != "resolved" {
		return QueuePage{}, ErrInvalidQueueFilter
	}
	if filter.SubjectType != "" && filter.SubjectType != string(SubjectPost) && filter.SubjectType != string(SubjectAccount) && filter.SubjectType != string(SubjectEvent) {
		return QueuePage{}, ErrInvalidQueueFilter
	}
	if filter.AppealStatus != "" && filter.AppealStatus != "none" && filter.AppealStatus != string(AppealStatusPending) && filter.AppealStatus != string(AppealStatusUpheld) && filter.AppealStatus != string(AppealStatusChanged) {
		return QueuePage{}, ErrInvalidQueueFilter
	}
	var cursor queueCursor
	if filter.Cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(filter.Cursor)
		if err != nil || decodeQueueCursor(raw, &cursor) != nil || cursor.ID == uuid.Nil || cursor.CreatedAt.IsZero() {
			return QueuePage{}, ErrInvalidQueueCursor
		}
	}
	if s == nil || s.pool == nil {
		return QueuePage{}, ErrInvalidCommand
	}
	rows, err := s.pool.Query(ctx, `SELECT c.id,c.state,c.subject_type,c.subject_did,count(r.report_id)::int,c.revision,c.created_at
		FROM moderation_cases c LEFT JOIN moderation_case_reports r ON r.case_id=c.id LEFT JOIN moderation_appeals a ON a.case_id=c.id
		WHERE ($1='' OR c.state=$1) AND ($2='' OR c.subject_type=$2) AND ($3='' OR COALESCE(a.status,'none')=$3)
		AND ($4::timestamptz IS NULL OR (c.created_at,c.id)<($4,$5))
		GROUP BY c.id HAVING true ORDER BY c.created_at DESC,c.id DESC LIMIT $6`, filter.State, filter.SubjectType, filter.AppealStatus, nullTime(cursor.CreatedAt), nullUUID(cursor.ID), filter.Limit+1)
	if err != nil {
		return QueuePage{}, fmt.Errorf("list moderation queue: %w", err)
	}
	defer rows.Close()
	page := QueuePage{Items: []QueueItem{}}
	var keys []queueCursor
	for rows.Next() {
		var item QueueItem
		var id uuid.UUID
		if err := rows.Scan(&id, &item.State, &item.SubjectType, &item.SubjectDID, &item.ReportCount, &item.Revision, &item.CreatedAt); err != nil {
			return QueuePage{}, err
		}
		item.CaseReference, _ = FormatCaseReference(id)
		page.Items = append(page.Items, item)
		keys = append(keys, queueCursor{item.CreatedAt, id})
	}
	if err := rows.Err(); err != nil {
		return QueuePage{}, err
	}
	if len(page.Items) > filter.Limit {
		page.Items = page.Items[:filter.Limit]
		keys = keys[:filter.Limit]
		raw, _ := json.Marshal(keys[len(keys)-1])
		page.Cursor = base64.RawURLEncoding.EncodeToString(raw)
	}
	return page, nil
}

func decodeQueueCursor(raw []byte, cursor *queueCursor) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(cursor); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("cursor must contain exactly one object")
	}
	return nil
}

type AdminReport struct {
	ID          string     `json:"id"`
	ReporterDID syntax.DID `json:"reporterDid"`
	Reason      string     `json:"reason"`
	Details     string     `json:"details,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}
type AdminEvent struct {
	Type         string    `json:"type"`
	ActorID      string    `json:"actorId"`
	SourceSystem string    `json:"sourceSystem"`
	Revision     int64     `json:"revision"`
	CreatedAt    time.Time `json:"createdAt"`
}
type AdminDecision struct {
	Disposition       Disposition `json:"disposition"`
	Reason            Reason      `json:"reason,omitempty"`
	EvidenceNotes     string      `json:"evidenceNotes,omitempty"`
	UserSafeDetail    string      `json:"userSafeDetail,omitempty"`
	SeverityRationale string      `json:"severityRationale,omitempty"`
	CreatedAt         time.Time   `json:"createdAt"`
}
type AdminEffect struct {
	EffectID  string     `json:"effectId"`
	Type      EffectType `json:"type"`
	Action    string     `json:"action"`
	Rationale string     `json:"rationale,omitempty"`
	Active    bool       `json:"active"`
	CreatedAt time.Time  `json:"createdAt"`
}
type AdminStrikeContext struct {
	LogicalEffectID string     `json:"logicalEffectId"`
	IssuedAt        time.Time  `json:"issuedAt"`
	DueAt           time.Time  `json:"dueAt"`
	ExpiredAt       *time.Time `json:"expiredAt,omitempty"`
	OverturnedAt    *time.Time `json:"overturnedAt,omitempty"`
	Active          bool       `json:"active"`
}
type AdminDetail struct {
	QueueItem
	SubjectCollection  string              `json:"subjectCollection,omitempty"`
	SubjectRkey        string              `json:"subjectRkey,omitempty"`
	SubjectURI         syntax.ATURI        `json:"subjectUri,omitempty"`
	SubjectCIDSnapshot string              `json:"subjectCidSnapshot,omitempty"`
	OwnerDID           syntax.DID          `json:"ownerDid"`
	SafeSnapshot       SubjectSnapshot     `json:"safeSnapshot"`
	Reports            []AdminReport       `json:"reports"`
	Events             []AdminEvent        `json:"events"`
	Decisions          []AdminDecision     `json:"decisions"`
	Effects            []AdminEffect       `json:"effects"`
	StrikeContext      *AdminStrikeContext `json:"strikeContext,omitempty"`
	Standing           OwnerStanding       `json:"standing"`
	AppealStatus       AppealStatus        `json:"appealStatus,omitempty"`
}

func (s *Store) AdminDetail(ctx context.Context, reference string) (AdminDetail, error) {
	parsed, err := ParseCaseReference(reference)
	if err != nil {
		return AdminDetail{}, err
	}
	var detail AdminDetail
	var id uuid.UUID
	var rawSnapshot json.RawMessage
	err = s.pool.QueryRow(ctx, `SELECT c.id,c.state,c.subject_type,c.subject_did,COALESCE(c.subject_collection,''),COALESCE(c.subject_rkey,''),COALESCE(c.subject_uri,''),COALESCE(c.subject_cid_snapshot,''),c.owner_did,c.revision,c.created_at,c.safe_snapshot,COALESCE((SELECT status FROM moderation_appeals WHERE case_id=c.id),'') FROM moderation_cases c WHERE c.id=$1`, parsed.UUID()).Scan(&id, &detail.State, &detail.SubjectType, &detail.SubjectDID, &detail.SubjectCollection, &detail.SubjectRkey, &detail.SubjectURI, &detail.SubjectCIDSnapshot, &detail.OwnerDID, &detail.Revision, &detail.CreatedAt, &rawSnapshot, &detail.AppealStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminDetail{}, ErrCaseNotFound
	}
	if err != nil {
		return AdminDetail{}, err
	}
	detail.CaseReference, _ = FormatCaseReference(id)
	detail.SafeSnapshot, err = presentSubjectSnapshot(rawSnapshot, SubjectSnapshot{
		Type: detail.SubjectType, DID: detail.SubjectDID, Collection: detail.SubjectCollection,
		Rkey: detail.SubjectRkey, URI: detail.SubjectURI, CID: detail.SubjectCIDSnapshot,
	})
	if err != nil {
		return AdminDetail{}, fmt.Errorf("present moderation subject snapshot: %w", err)
	}
	rows, err := s.pool.Query(ctx, `SELECT r.id,r.reporter_did,r.reason_type,COALESCE(r.details,''),r.created_at FROM moderation_reports r JOIN moderation_case_reports cr ON cr.report_id=r.id WHERE cr.case_id=$1 ORDER BY r.created_at,r.id`, id)
	if err != nil {
		return AdminDetail{}, err
	}
	defer rows.Close()
	detail.Reports = []AdminReport{}
	for rows.Next() {
		var report AdminReport
		if err := rows.Scan(&report.ID, &report.ReporterDID, &report.Reason, &report.Details, &report.CreatedAt); err != nil {
			return AdminDetail{}, err
		}
		detail.Reports = append(detail.Reports, report)
	}
	detail.ReportCount = len(detail.Reports)
	events, err := s.pool.Query(ctx, `SELECT event_type,actor_id,source_system,result_revision,created_at FROM moderation_case_events WHERE case_id=$1 ORDER BY created_at,id`, id)
	if err != nil {
		return AdminDetail{}, err
	}
	defer events.Close()
	detail.Events = []AdminEvent{}
	for events.Next() {
		var event AdminEvent
		if err := events.Scan(&event.Type, &event.ActorID, &event.SourceSystem, &event.Revision, &event.CreatedAt); err != nil {
			return AdminDetail{}, err
		}
		detail.Events = append(detail.Events, event)
	}
	decisions, err := s.pool.Query(ctx, `SELECT disposition,COALESCE(reason,''),COALESCE(internal_evidence_notes,''),COALESCE(user_safe_detail,''),COALESCE(severity_rationale,''),created_at FROM moderation_decisions WHERE case_id=$1 ORDER BY created_at,id`, id)
	if err != nil {
		return AdminDetail{}, err
	}
	defer decisions.Close()
	detail.Decisions = []AdminDecision{}
	for decisions.Next() {
		var decision AdminDecision
		if err := decisions.Scan(&decision.Disposition, &decision.Reason, &decision.EvidenceNotes, &decision.UserSafeDetail, &decision.SeverityRationale, &decision.CreatedAt); err != nil {
			return AdminDetail{}, err
		}
		detail.Decisions = append(detail.Decisions, decision)
	}
	effects, err := s.pool.Query(ctx, `SELECT e.logical_effect_id,e.effect_type,e.action,COALESCE(e.rationale,''),a.logical_effect_id IS NOT NULL,e.created_at FROM moderation_effect_events e LEFT JOIN moderation_active_case_effects a ON a.case_id=e.case_id AND a.logical_effect_id=e.logical_effect_id WHERE e.case_id=$1 ORDER BY e.created_at,e.id`, id)
	if err != nil {
		return AdminDetail{}, err
	}
	defer effects.Close()
	detail.Effects = []AdminEffect{}
	for effects.Next() {
		var effect AdminEffect
		if err := effects.Scan(&effect.EffectID, &effect.Type, &effect.Action, &effect.Rationale, &effect.Active, &effect.CreatedAt); err != nil {
			return AdminDetail{}, err
		}
		detail.Effects = append(detail.Effects, effect)
	}
	var strike AdminStrikeContext
	err = s.pool.QueryRow(ctx, `SELECT s.logical_effect_id,s.issued_at,s.due_at,s.expired_at,s.overturned_at,EXISTS(SELECT 1 FROM moderation_active_case_effects a WHERE a.case_id=s.case_id AND a.logical_effect_id=s.logical_effect_id) FROM moderation_case_strikes s WHERE s.case_id=$1`, id).Scan(&strike.LogicalEffectID, &strike.IssuedAt, &strike.DueAt, &strike.ExpiredAt, &strike.OverturnedAt, &strike.Active)
	if err == nil {
		detail.StrikeContext = &strike
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return AdminDetail{}, err
	}
	detail.Standing, err = s.Standing(ctx, detail.OwnerDID)
	if err != nil {
		return AdminDetail{}, err
	}
	return detail, nil
}

type OwnerStanding struct {
	ActiveStrikeCount  int  `json:"activeStrikeCount"`
	StrikeThreshold    int  `json:"strikeThreshold"`
	ThresholdSuspended bool `json:"thresholdSuspended"`
	SevereSuspended    bool `json:"severeSuspended"`
	Suspended          bool `json:"suspended"`
}

func (s *Store) Standing(ctx context.Context, owner syntax.DID) (OwnerStanding, error) {
	var out OwnerStanding
	out.StrikeThreshold = 3
	err := s.pool.QueryRow(ctx, `SELECT active_strike_count,threshold_suspended,severe_suspended,effective_suspended FROM moderation_account_standings WHERE owner_did=$1`, owner).Scan(&out.ActiveStrikeCount, &out.ThresholdSuspended, &out.SevereSuspended, &out.Suspended)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	return out, err
}
func (s *Store) IsSuspended(ctx context.Context, owner syntax.DID) (bool, error) {
	standing, err := s.Standing(ctx, owner)
	return standing.Suspended, err
}

type OwnerHistoryItem struct {
	CaseReference  string          `json:"caseReference"`
	EventType      string          `json:"eventType"`
	Reason         Reason          `json:"reason,omitempty"`
	UserSafeDetail string          `json:"userSafeDetail,omitempty"`
	Effects        []OwnerEffect   `json:"effects"`
	SafeSnapshot   SubjectSnapshot `json:"safeSnapshot"`
	AppealStatus   AppealStatus    `json:"appealStatus,omitempty"`
	OccurredAt     time.Time       `json:"occurredAt"`
}
type OwnerEffect struct {
	Type   EffectType `json:"type"`
	Action string     `json:"action"`
	DueAt  *time.Time `json:"dueAt,omitempty"`
}
type OwnerHistoryPage struct {
	Items  []OwnerHistoryItem `json:"items"`
	Cursor string             `json:"cursor,omitempty"`
}

func (s *Store) OwnerHistory(ctx context.Context, owner syntax.DID, cursor string, limit int) (OwnerHistoryPage, error) {
	return s.ownerHistory(ctx, owner, cursor, limit, nil)
}

func (s *Store) ownerHistory(ctx context.Context, owner syntax.DID, cursor string, limit int, caseID *uuid.UUID) (OwnerHistoryPage, error) {
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 100 {
		return OwnerHistoryPage{}, ErrInvalidCommand
	}
	var after queueCursor
	if cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil || decodeQueueCursor(raw, &after) != nil || after.ID == uuid.Nil || after.CreatedAt.IsZero() {
			return OwnerHistoryPage{}, ErrInvalidQueueCursor
		}
	}
	rows, err := s.pool.Query(ctx, `SELECT c.id,e.id,e.event_type,e.created_at,c.safe_snapshot,c.subject_type,c.subject_did,COALESCE(c.subject_collection,''),COALESCE(c.subject_rkey,''),COALESCE(c.subject_uri,''),COALESCE(c.subject_cid_snapshot,''),COALESCE(d.reason,''),COALESCE(d.user_safe_detail,''),COALESCE(a.status,''),COALESCE(jsonb_agg(jsonb_build_object('type',ee.effect_type,'action',ee.action) ORDER BY ee.effect_type) FILTER(WHERE ee.effect_type IS NOT NULL),'[]')
		FROM moderation_case_events e JOIN moderation_cases c ON c.id=e.case_id LEFT JOIN moderation_decisions d ON d.case_event_id=e.id LEFT JOIN moderation_effect_events ee ON ee.case_event_id=e.id LEFT JOIN moderation_appeals a ON a.case_id=c.id
		WHERE c.owner_did=$1 AND (d.disposition='violation' OR ee.id IS NOT NULL) AND ($2::timestamptz IS NULL OR (e.created_at,e.id)<($2,$3)) AND ($5::uuid IS NULL OR c.id=$5) GROUP BY c.id,e.id,d.reason,d.user_safe_detail,a.status ORDER BY e.created_at DESC,e.id DESC LIMIT $4`, owner, nullTime(after.CreatedAt), nullUUID(after.ID), limit+1, caseID)
	if err != nil {
		return OwnerHistoryPage{}, err
	}
	defer rows.Close()
	page := OwnerHistoryPage{Items: []OwnerHistoryItem{}}
	var keys []queueCursor
	for rows.Next() {
		var item OwnerHistoryItem
		var caseID, eventID uuid.UUID
		var effectsJSON []byte
		var rawSnapshot json.RawMessage
		var fallback SubjectSnapshot
		if err := rows.Scan(&caseID, &eventID, &item.EventType, &item.OccurredAt, &rawSnapshot, &fallback.Type, &fallback.DID, &fallback.Collection, &fallback.Rkey, &fallback.URI, &fallback.CID, &item.Reason, &item.UserSafeDetail, &item.AppealStatus, &effectsJSON); err != nil {
			return OwnerHistoryPage{}, err
		}
		item.SafeSnapshot, err = presentSubjectSnapshot(rawSnapshot, fallback)
		if err != nil {
			return OwnerHistoryPage{}, fmt.Errorf("present owner moderation subject snapshot: %w", err)
		}
		if err := json.Unmarshal(effectsJSON, &item.Effects); err != nil {
			return OwnerHistoryPage{}, err
		}
		for index := range item.Effects {
			if item.Effects[index].Type == EffectStrike && item.Effects[index].Action == string(EffectApply) {
				dueAt := StrikeDeadline(item.OccurredAt)
				item.Effects[index].DueAt = &dueAt
			}
		}
		item.CaseReference, _ = FormatCaseReference(caseID)
		page.Items = append(page.Items, item)
		keys = append(keys, queueCursor{item.OccurredAt, eventID})
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		keys = keys[:limit]
		raw, _ := json.Marshal(keys[len(keys)-1])
		page.Cursor = base64.RawURLEncoding.EncodeToString(raw)
	}
	return page, rows.Err()
}
func (s *Store) OwnerEntry(ctx context.Context, owner syntax.DID, reference string) (OwnerHistoryPage, error) {
	parsed, err := ParseCaseReference(reference)
	if err != nil {
		return OwnerHistoryPage{}, err
	}
	caseID := parsed.UUID()
	page, err := s.ownerHistory(ctx, owner, "", 100, &caseID)
	if err != nil {
		return page, err
	}
	if len(page.Items) == 0 {
		return OwnerHistoryPage{}, ErrCaseNotFound
	}
	page.Cursor = ""
	return page, nil
}

func nullTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}
func nullUUID(value uuid.UUID) any {
	if value == uuid.Nil {
		return nil
	}
	return value
}
