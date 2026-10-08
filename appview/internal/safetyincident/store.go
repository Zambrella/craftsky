package safetyincident

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/imagesafety"
)

type WorkItem struct {
	Reference string
	Kind      string
	State     string
	Priority  int
	CreatedAt time.Time
	Deadline  *time.Time
	Owner     string
	Cover     string
	Alert     bool
	Sensitive RestrictedFields
}

type SafeDetail struct {
	Reference                  string         `json:"reference"`
	Kind                       string         `json:"kind"`
	State                      string         `json:"state"`
	ProviderReference          string         `json:"providerReference"`
	IntegrityMetadataReference string         `json:"integrityMetadataReference"`
	ScannerID                  string         `json:"scannerId"`
	PolicyVersion              string         `json:"policyVersion"`
	CorpusVersion              string         `json:"corpusVersion"`
	DetectedAt                 time.Time      `json:"detectedAt"`
	SubjectCounts              map[string]int `json:"subjectCounts"`
}

type Store struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewStore(pool *pgxpool.Pool, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{pool: pool, now: now}
}

func (store *Store) RecordMatchTx(ctx context.Context, tx pgx.Tx, detection imagesafety.MatchDetection) error {
	if store == nil || detection.ResultID == uuid.Nil || detection.ProviderReference == "" ||
		detection.IntegrityMetadataReference == "" || detection.DetectedAt.IsZero() {
		return errors.New("invalid restricted image match")
	}
	incidentID := uuid.New()
	if _, err := tx.Exec(ctx, `
		INSERT INTO safety_incidents(
			id,scan_result_id,provider_reference,integrity_metadata_reference,detected_at,updated_at
		) VALUES($1,$2,$3,$4,$5,$5)
		ON CONFLICT (scan_result_id) DO NOTHING
	`, incidentID, detection.ResultID, detection.ProviderReference,
		detection.IntegrityMetadataReference, detection.DetectedAt.UTC()); err != nil {
		return fmt.Errorf("insert safety incident: %w", err)
	}
	if err := tx.QueryRow(ctx, `SELECT id FROM safety_incidents WHERE scan_result_id=$1`, detection.ResultID).Scan(&incidentID); err != nil {
		return fmt.Errorf("resolve safety incident: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO safety_incident_subjects(
			incident_id,owner_did,subject_uri,source_cid,subject_kind,image_slot,linked_at
		)
		SELECT $1,split_part(subject_uri,'/',3),subject_uri,source_cid,subject_kind,image_slot,$3
		FROM image_subject_requirements WHERE scan_result_id=$2
		ON CONFLICT DO NOTHING
	`, incidentID, detection.ResultID, detection.DetectedAt.UTC()); err != nil {
		return fmt.Errorf("link safety incident subjects: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO safety_incident_subjects(
			incident_id,owner_did,subject_uri,source_cid,subject_kind,image_slot,linked_at
		)
		SELECT $1,profile_did,'at://' || profile_did || '/app.bsky.actor.profile/self',source_cid,
		       'profile',slot,$3
		FROM profile_image_candidates WHERE scan_result_id=$2
		ON CONFLICT DO NOTHING
	`, incidentID, detection.ResultID, detection.DetectedAt.UTC()); err != nil {
		return fmt.Errorf("link safety incident profile subjects: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO safety_incident_events(id,incident_id,event_type,created_at)
		VALUES($1,$2,'detected',$3)
		ON CONFLICT (incident_id,event_type) DO NOTHING
	`, uuid.New(), incidentID, detection.DetectedAt.UTC()); err != nil {
		return fmt.Errorf("record safety incident detection: %w", err)
	}
	return nil
}

func (store *Store) SafeWork(ctx context.Context) ([]WorkItem, error) {
	if store == nil || store.pool == nil {
		return nil, errors.New("safety incident store unavailable")
	}
	rows, err := store.pool.Query(ctx, `
		SELECT id,state,detected_at
		FROM safety_incidents
		WHERE state IN ('detected','confirmed')
		ORDER BY detected_at,id
	`)
	if err != nil {
		return nil, fmt.Errorf("read safe incident work: %w", err)
	}
	defer rows.Close()
	items := make([]WorkItem, 0)
	for rows.Next() {
		var id uuid.UUID
		var item WorkItem
		if err := rows.Scan(&id, &item.State, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Reference = id.String()
		item.Kind = "imageMatch"
		item.Priority = PriorityFor(PriorityInput{Detected: true}, store.now().UTC())
		items = append(items, item)
	}
	return items, rows.Err()
}

func (store *Store) SafeDetail(ctx context.Context, incidentID uuid.UUID) (SafeDetail, error) {
	if store == nil || store.pool == nil || incidentID == uuid.Nil {
		return SafeDetail{}, errors.New("invalid safety incident reference")
	}
	detail := SafeDetail{Reference: incidentID.String(), SubjectCounts: make(map[string]int)}
	if err := store.pool.QueryRow(ctx, `
		SELECT incident.incident_kind,incident.state,incident.provider_reference,
		       incident.integrity_metadata_reference,result.scanner_id,
		       result.policy_version,result.corpus_version,incident.detected_at
		FROM safety_incidents incident
		JOIN image_scan_results result ON result.id=incident.scan_result_id
		WHERE incident.id=$1
	`, incidentID).Scan(&detail.Kind, &detail.State, &detail.ProviderReference,
		&detail.IntegrityMetadataReference, &detail.ScannerID, &detail.PolicyVersion,
		&detail.CorpusVersion, &detail.DetectedAt); err != nil {
		return SafeDetail{}, err
	}
	rows, err := store.pool.Query(ctx, `
		SELECT subject_kind,count(*) FROM safety_incident_subjects
		WHERE incident_id=$1 GROUP BY subject_kind ORDER BY subject_kind
	`, incidentID)
	if err != nil {
		return SafeDetail{}, fmt.Errorf("read safety incident subject context: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var kind string
		var count int
		if err := rows.Scan(&kind, &count); err != nil {
			return SafeDetail{}, err
		}
		detail.SubjectCounts[kind] = count
	}
	return detail, rows.Err()
}
