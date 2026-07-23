package instagram

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PrivateDataExport is a request-lifetime owner snapshot. It intentionally
// omits HMAC/digest material, webhook work, rate buckets, operator evidence,
// PDS operation internals, delivery routing, and other importers' facts.
type PrivateDataExport struct {
	Version              int                         `json:"version"`
	OwnerDID             syntax.DID                  `json:"ownerDid"`
	VerificationAttempts []PrivateVerificationExport `json:"verificationAttempts"`
	AccountLinks         []PrivateAccountLinkExport  `json:"accountLinks"`
	Imports              []PrivateImportExport       `json:"imports"`
	Suggestions          []PrivateSuggestionExport   `json:"suggestions"`
	MatchNotifications   []PrivateMatchEventExport   `json:"matchNotifications"`
	MatchPushEnabled     *bool                       `json:"matchPushEnabled,omitempty"`
}

func (PrivateDataExport) String() string     { return "Instagram private-data export [REDACTED]" }
func (e PrivateDataExport) GoString() string { return e.String() }
func (e PrivateDataExport) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, e.String())
}

type PrivateVerificationExport struct {
	ID                       uuid.UUID                `json:"verificationId"`
	State                    VerificationAttemptState `json:"state"`
	CandidateInstagramUserID string                   `json:"candidateInstagramUserId,omitempty"`
	CandidateUsername        string                   `json:"candidateUsername,omitempty"`
	RetryCode                AttemptRetryCode         `json:"retryCode,omitempty"`
	ExpiresAt                time.Time                `json:"expiresAt"`
	ProcessingStartedAt      *time.Time               `json:"processingStartedAt,omitempty"`
	TerminalAt               *time.Time               `json:"terminalAt,omitempty"`
	CreatedAt                time.Time                `json:"createdAt"`
	UpdatedAt                time.Time                `json:"updatedAt"`
}

func (PrivateVerificationExport) String() string {
	return "Instagram private verification export [REDACTED]"
}
func (e PrivateVerificationExport) GoString() string { return e.String() }
func (e PrivateVerificationExport) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, e.String())
}

type PrivateAccountLinkExport struct {
	ID                      uuid.UUID          `json:"linkId"`
	State                   InstagramLinkState `json:"state"`
	InstagramUserID         string             `json:"instagramUserId,omitempty"`
	Username                string             `json:"username,omitempty"`
	Discoverable            bool               `json:"discoverable"`
	ConflictPending         bool               `json:"conflictPending"`
	VerifiedAt              time.Time          `json:"verifiedAt"`
	MembershipInactiveAt    *time.Time         `json:"membershipInactiveAt,omitempty"`
	RevokedAt               *time.Time         `json:"revokedAt,omitempty"`
	SupersededAt            *time.Time         `json:"supersededAt,omitempty"`
	RawIdentityScheduledFor *time.Time         `json:"rawIdentityScheduledFor,omitempty"`
	CreatedAt               time.Time          `json:"createdAt"`
	UpdatedAt               time.Time          `json:"updatedAt"`
}

func (PrivateAccountLinkExport) String() string {
	return "Instagram private account-link export [REDACTED]"
}
func (e PrivateAccountLinkExport) GoString() string { return e.String() }
func (e PrivateAccountLinkExport) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, e.String())
}

type PrivateImportExport struct {
	ID                   uuid.UUID             `json:"importId"`
	State                InstagramImportState  `json:"state"`
	SourceType           ImportSourceType      `json:"sourceType"`
	MembershipInactiveAt *time.Time            `json:"membershipInactiveAt,omitempty"`
	FollowingCount       int                   `json:"followingCount"`
	CreatedAt            time.Time             `json:"createdAt"`
	UpdatedAt            time.Time             `json:"updatedAt"`
	RetainedEntries      []PrivateHandleExport `json:"retainedEntries"`
}

func (PrivateImportExport) String() string     { return "Instagram private import export [REDACTED]" }
func (e PrivateImportExport) GoString() string { return e.String() }
func (e PrivateImportExport) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, e.String())
}

type PrivateHandleExport struct {
	Username  string    `json:"username"`
	Matched   bool      `json:"matched"`
	CreatedAt time.Time `json:"createdAt"`
}

func (PrivateHandleExport) String() string     { return "Instagram private handle export [REDACTED]" }
func (e PrivateHandleExport) GoString() string { return e.String() }
func (e PrivateHandleExport) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, e.String())
}

type PrivateSuggestionExport struct {
	ID         uuid.UUID                 `json:"suggestionId"`
	TargetDID  syntax.DID                `json:"targetDid"`
	State      InstagramSuggestionState  `json:"state"`
	Reason     InstagramSuggestionReason `json:"reason"`
	CreatedAt  time.Time                 `json:"createdAt"`
	UpdatedAt  time.Time                 `json:"updatedAt"`
	TerminalAt *time.Time                `json:"terminalAt,omitempty"`
}

func (PrivateSuggestionExport) String() string {
	return "Instagram private suggestion export [REDACTED]"
}
func (e PrivateSuggestionExport) GoString() string { return e.String() }
func (e PrivateSuggestionExport) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, e.String())
}

type PrivateMatchEventExport struct {
	ID          uuid.UUID  `json:"notificationId"`
	State       string     `json:"state"`
	Count       int        `json:"count"`
	CountCapped bool       `json:"countCapped"`
	Destination string     `json:"destination"`
	ActivityAt  time.Time  `json:"activityAt"`
	IndexedAt   time.Time  `json:"indexedAt"`
	RetractedAt *time.Time `json:"retractedAt,omitempty"`
}

// ExportOwnerData builds a repeatable-read snapshot and never persists an
// export blob. The transaction is rolled back after the snapshot is materialized.
func (s *PrivateDataService) ExportOwnerData(ctx context.Context, owner syntax.DID) (PrivateDataExport, error) {
	if err := s.validateOwner(owner); err != nil {
		return PrivateDataExport{}, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return PrivateDataExport{}, fmt.Errorf("begin Instagram private export: %w", err)
	}
	defer tx.Rollback(ctx)

	export := PrivateDataExport{
		Version:              1,
		OwnerDID:             owner,
		VerificationAttempts: make([]PrivateVerificationExport, 0),
		AccountLinks:         make([]PrivateAccountLinkExport, 0),
		Imports:              make([]PrivateImportExport, 0),
		Suggestions:          make([]PrivateSuggestionExport, 0),
		MatchNotifications:   make([]PrivateMatchEventExport, 0),
	}
	if err := loadPrivateVerificationExport(ctx, tx, owner, &export); err != nil {
		return PrivateDataExport{}, err
	}
	if err := loadPrivateLinkExport(ctx, tx, owner, &export); err != nil {
		return PrivateDataExport{}, err
	}
	if err := loadPrivateImportExport(ctx, tx, owner, &export); err != nil {
		return PrivateDataExport{}, err
	}
	if err := loadPrivateSuggestionExport(ctx, tx, owner, &export); err != nil {
		return PrivateDataExport{}, err
	}
	if err := loadPrivateNotificationExport(ctx, tx, owner, &export); err != nil {
		return PrivateDataExport{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PrivateDataExport{}, fmt.Errorf("finish Instagram private export snapshot: %w", err)
	}
	return export, nil
}

func loadPrivateVerificationExport(ctx context.Context, tx pgx.Tx, owner syntax.DID, export *PrivateDataExport) error {
	rows, err := tx.Query(ctx, `
		SELECT id, state, candidate_igsid, candidate_username, retry_code,
		       expires_at, processing_started_at, terminal_at, created_at, updated_at
		FROM instagram_verification_attempts
		WHERE owner_did=$1
		ORDER BY created_at, id
	`, owner)
	if err != nil {
		return fmt.Errorf("read Instagram private attempts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item PrivateVerificationExport
		var candidateID, username, retry sql.NullString
		if err := rows.Scan(
			&item.ID, &item.State, &candidateID, &username, &retry,
			&item.ExpiresAt, &item.ProcessingStartedAt, &item.TerminalAt,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return err
		}
		if !item.State.Valid() {
			return ErrInvalidInstagramState
		}
		item.CandidateInstagramUserID = candidateID.String
		item.CandidateUsername = username.String
		if retry.Valid {
			item.RetryCode = AttemptRetryCode(retry.String)
			if !item.RetryCode.Valid() {
				return errors.New("invalid stored Instagram export retry code")
			}
		}
		export.VerificationAttempts = append(export.VerificationAttempts, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate Instagram private attempts: %w", err)
	}
	return nil
}

func loadPrivateLinkExport(ctx context.Context, tx pgx.Tx, owner syntax.DID, export *PrivateDataExport) error {
	rows, err := tx.Query(ctx, `
		SELECT id, state, igsid, username, discoverable, conflict_pending,
		       verified_at, membership_inactive_at, revoked_at, superseded_at,
		       raw_identity_purge_at, created_at, updated_at
		FROM instagram_account_links
		WHERE owner_did=$1
		ORDER BY created_at, id
	`, owner)
	if err != nil {
		return fmt.Errorf("read Instagram private links: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item PrivateAccountLinkExport
		var igsid, username sql.NullString
		if err := rows.Scan(
			&item.ID, &item.State, &igsid, &username, &item.Discoverable,
			&item.ConflictPending, &item.VerifiedAt, &item.MembershipInactiveAt,
			&item.RevokedAt, &item.SupersededAt, &item.RawIdentityScheduledFor,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return err
		}
		if !item.State.Valid() {
			return ErrInvalidInstagramState
		}
		item.InstagramUserID = igsid.String
		item.Username = username.String
		export.AccountLinks = append(export.AccountLinks, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate Instagram private links: %w", err)
	}
	return nil
}

func loadPrivateImportExport(ctx context.Context, tx pgx.Tx, owner syntax.DID, export *PrivateDataExport) error {
	rows, err := tx.Query(ctx, `
		SELECT id, state, source_type, membership_inactive_at, following_count,
		       created_at, updated_at
		FROM instagram_graph_imports
		WHERE owner_did=$1
		ORDER BY created_at, id
	`, owner)
	if err != nil {
		return fmt.Errorf("read Instagram private imports: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item PrivateImportExport
		if err := rows.Scan(
			&item.ID, &item.State, &item.SourceType, &item.MembershipInactiveAt,
			&item.FollowingCount,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return err
		}
		if !item.State.Valid() || !item.SourceType.Valid() {
			rows.Close()
			return ErrInvalidInstagramState
		}
		item.RetainedEntries = make([]PrivateHandleExport, 0)
		export.Imports = append(export.Imports, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate Instagram private imports: %w", err)
	}
	rows.Close()

	for index := range export.Imports {
		handleRows, err := tx.Query(ctx, `
			SELECT username_normalized, matched, created_at
			FROM instagram_graph_handles
			WHERE import_id=$1
			ORDER BY id
		`, export.Imports[index].ID)
		if err != nil {
			return fmt.Errorf("read Instagram private retained entries: %w", err)
		}
		for handleRows.Next() {
			var handle PrivateHandleExport
			if err := handleRows.Scan(
				&handle.Username, &handle.Matched, &handle.CreatedAt,
			); err != nil {
				handleRows.Close()
				return err
			}
			export.Imports[index].RetainedEntries = append(export.Imports[index].RetainedEntries, handle)
		}
		if err := handleRows.Err(); err != nil {
			handleRows.Close()
			return fmt.Errorf("iterate Instagram private retained entries: %w", err)
		}
		handleRows.Close()
	}
	return nil
}

func loadPrivateSuggestionExport(ctx context.Context, tx pgx.Tx, owner syntax.DID, export *PrivateDataExport) error {
	rows, err := tx.Query(ctx, `
		SELECT id, target_did, state, reason, created_at, updated_at, terminal_at
		FROM instagram_follow_suggestions
		WHERE importer_did=$1
		ORDER BY created_at, id
	`, owner)
	if err != nil {
		return fmt.Errorf("read Instagram private suggestions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item PrivateSuggestionExport
		var target string
		if err := rows.Scan(
			&item.ID, &target, &item.State, &item.Reason,
			&item.CreatedAt, &item.UpdatedAt, &item.TerminalAt,
		); err != nil {
			return err
		}
		if !item.State.Valid() || item.Reason != SuggestionReasonVerifiedInstagramFollow {
			return ErrInvalidInstagramState
		}
		item.TargetDID = syntax.DID(target)
		export.Suggestions = append(export.Suggestions, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate Instagram private suggestions: %w", err)
	}
	return nil
}

func loadPrivateNotificationExport(ctx context.Context, tx pgx.Tx, owner syntax.DID, export *PrivateDataExport) error {
	rows, err := tx.Query(ctx, `
		SELECT id, state, system_count, system_count_capped,
		       system_destination, activity_at, indexed_at, retracted_at
		FROM notification_events
		WHERE recipient_did=$1 AND kind='system' AND category='instagramMatch'
		ORDER BY activity_at, id
	`, owner)
	if err != nil {
		return fmt.Errorf("read Instagram private notifications: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item PrivateMatchEventExport
		if err := rows.Scan(
			&item.ID, &item.State, &item.Count, &item.CountCapped,
			&item.Destination, &item.ActivityAt, &item.IndexedAt,
			&item.RetractedAt,
		); err != nil {
			return err
		}
		export.MatchNotifications = append(export.MatchNotifications, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate Instagram private notifications: %w", err)
	}
	var pushEnabled bool
	err = tx.QueryRow(ctx, `
		SELECT push_enabled FROM notification_preferences
		WHERE account_did=$1 AND category='instagramMatch'
	`, owner).Scan(&pushEnabled)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("read Instagram private notification preference: %w", err)
	}
	if err == nil {
		export.MatchPushEnabled = &pushEnabled
	}
	return nil
}
