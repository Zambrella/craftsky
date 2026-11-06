package guestvisibility

import (
	"context"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"social.craftsky/appview/internal/guestvisibilitydb"
)

type EvidenceKind string

const (
	Source       EvidenceKind = "source"
	WriteReceipt EvidenceKind = "writeReceipt"
	Observation  EvidenceKind = "observation"
)

// Evidence carries a trusted source revision or an authoritative owner-operation
// observation bound. A CID is never used to order repository changes.
type Evidence struct {
	Revision syntax.TID
	CID      syntax.CID
	Kind     EvidenceKind
}

// ApplyTx returns false when older/conflicting evidence must not project any
// presentation. Conflicts persist an unknown state rather than preserve public
// visibility. The caller owns the transaction and lifecycle fence.
func ApplyTx(ctx context.Context, tx pgx.Tx, owner syntax.DID, preference Preference, evidence Evidence) (bool, error) {
	if owner == "" || evidence.Revision == "" || (evidence.Kind != Source && evidence.Kind != WriteReceipt && evidence.Kind != Observation) {
		return false, errors.New("guest visibility evidence is incomplete")
	}
	queries := guestvisibilitydb.New(tx)
	if err := queries.EnsureGuestVisibilityRow(ctx, guestvisibilitydb.EnsureGuestVisibilityRowParams{Did: owner.String(), RecordCid: ""}); err != nil {
		return false, err
	}
	current, err := queries.LockGuestVisibility(ctx, owner.String())
	if err != nil {
		return false, err
	}
	accepted := true
	cid := pgtype.Text{String: evidence.CID.String(), Valid: true}
	if current.GuestVisibilityRevision.Valid {
		if current.GuestVisibilityRevision.String > evidence.Revision.String() {
			// Repository-head observations can be newer than this profile's
			// actual write. A normal source event with the exact observed CID
			// and known preference may project its matching presentation while
			// the later visibility floor and provenance remain unchanged.
			sameKnownPreference := current.GuestVisibilityHidden.Valid && preference != Unknown && current.GuestVisibilityHidden.Bool == (preference == Hidden)
			return current.GuestVisibilityRecordCid.Valid && current.GuestVisibilityRecordCid.String == evidence.CID.String() && sameKnownPreference, nil
		}
		if current.GuestVisibilityRevision.String == evidence.Revision.String() {
			samePreference := (!current.GuestVisibilityHidden.Valid && preference == Unknown) ||
				(current.GuestVisibilityHidden.Valid && preference != Unknown && current.GuestVisibilityHidden.Bool == (preference == Hidden))
			if current.GuestVisibilityRecordCid.Valid && current.GuestVisibilityRecordCid.String == cid.String && samePreference {
				return true, nil
			}
			// Keep the conflicting revision floor. No same-revision delivery may
			// subsequently re-enable an account whose authority is uncertain.
			preference = Unknown
			cid = pgtype.Text{}
			accepted = false
		}
	}
	err = queries.SetGuestVisibilityEvidence(ctx, guestvisibilitydb.SetGuestVisibilityEvidenceParams{
		Did: owner.String(), GuestVisibilityHidden: pgtype.Bool{Bool: preference == Hidden, Valid: preference == Public || preference == Hidden},
		GuestVisibilityRevision:  pgtype.Text{String: evidence.Revision.String(), Valid: true},
		GuestVisibilityRecordCid: cid, GuestVisibilityEvidence: pgtype.Text{String: string(evidence.Kind), Valid: true},
	})
	return accepted, err
}
