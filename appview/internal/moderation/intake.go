package moderation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrInvalidAcceptedReport = errors.New("invalid accepted moderation report")

type safeSubjectSnapshot struct {
	Type            SubjectType `json:"type"`
	DID             string      `json:"did"`
	Collection      string      `json:"collection,omitempty"`
	Rkey            string      `json:"rkey,omitempty"`
	URI             string      `json:"uri,omitempty"`
	CID             string      `json:"cid,omitempty"`
	SubmittedHandle string      `json:"submittedHandle,omitempty"`
}

func (s *Store) AttachAcceptedReportTx(ctx context.Context, tx pgx.Tx, report AcceptedReport) (Case, error) {
	subjectKey, err := canonicalSubjectKey(report)
	if err != nil {
		return Case{}, err
	}
	snapshot, err := json.Marshal(safeSubjectSnapshot{
		Type: report.SubjectType, DID: report.SubjectDID.String(),
		Collection: report.SubjectCollection, Rkey: report.SubjectRkey,
		URI: report.SubjectURI.String(), CID: report.SubjectCIDSnapshot,
		SubmittedHandle: report.SubmittedHandleSnapshot,
	})
	if err != nil {
		return Case{}, fmt.Errorf("marshal moderation case snapshot: %w", err)
	}

	created := Case{}
	err = tx.QueryRow(ctx, `
		INSERT INTO moderation_cases(
			id,subject_key,subject_type,subject_did,subject_collection,subject_rkey,
			subject_uri,subject_cid_snapshot,owner_did,safe_snapshot,created_at,updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)
		ON CONFLICT (subject_key) WHERE state='open' DO NOTHING
		RETURNING id,subject_key,state,revision,created_at
	`, uuid.New(), subjectKey, report.SubjectType, report.SubjectDID,
		nullIfEmpty(report.SubjectCollection), nullIfEmpty(report.SubjectRkey),
		nullIfEmpty(report.SubjectURI.String()), nullIfEmpty(report.SubjectCIDSnapshot),
		report.SubjectDID, snapshot, report.CreatedAt.UTC()).Scan(
		&created.ID, &created.SubjectKey, &created.State, &created.Revision, &created.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			SELECT id,subject_key,state,revision,created_at
			FROM moderation_cases
			WHERE subject_key=$1 AND state='open'
		`, subjectKey).Scan(
			&created.ID, &created.SubjectKey, &created.State, &created.Revision, &created.CreatedAt,
		)
	}
	if err != nil {
		return Case{}, fmt.Errorf("find or create moderation case: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO moderation_case_reports(case_id,report_id,attached_at)
		VALUES ($1,$2,$3)
	`, created.ID, report.ID, report.CreatedAt.UTC()); err != nil {
		return Case{}, fmt.Errorf("attach moderation report to case: %w", err)
	}
	return created, nil
}

func canonicalSubjectKey(report AcceptedReport) (string, error) {
	if report.ID == "" || report.SubjectDID == "" || report.CreatedAt.IsZero() {
		return "", ErrInvalidAcceptedReport
	}
	switch report.SubjectType {
	case SubjectAccount:
		if report.SubjectCollection != "" || report.SubjectRkey != "" || report.SubjectURI != "" {
			return "", ErrInvalidAcceptedReport
		}
		return "account:" + report.SubjectDID.String(), nil
	case SubjectPost, SubjectEvent:
		if report.SubjectCollection == "" || report.SubjectRkey == "" || report.SubjectURI == "" {
			return "", ErrInvalidAcceptedReport
		}
		return string(report.SubjectType) + ":" + report.SubjectURI.String(), nil
	default:
		return "", ErrInvalidAcceptedReport
	}
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
