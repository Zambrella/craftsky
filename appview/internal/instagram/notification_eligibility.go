package instagram

import (
	"context"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/notifications"
)

// NotificationEligibilityService applies the shared suggestion policy at the
// last notification boundaries. Definitively invalid support is retracted in
// one transaction; unavailable safety data is hidden without destroying the
// support so a later retry can recover.
type NotificationEligibilityService struct {
	pool          *pgxpool.Pool
	policy        InstagramSuggestionEligibilityPolicy
	notifications InstagramMatchNotificationService
}

func NewNotificationEligibilityService(pool *pgxpool.Pool, policy InstagramSuggestionEligibilityPolicy, notificationService InstagramMatchNotificationService) (*NotificationEligibilityService, error) {
	if pool == nil || policy == nil || notificationService == nil {
		return nil, errors.New("Instagram notification eligibility dependencies are required")
	}
	return &NotificationEligibilityService{pool: pool, policy: policy, notifications: notificationService}, nil
}

func (s *NotificationEligibilityService) RevalidateNotification(ctx context.Context, notificationID uuid.UUID, stage EligibilityStage) (bool, error) {
	if s == nil || s.pool == nil || s.policy == nil || s.notifications == nil || notificationID == uuid.Nil || !stage.Valid() {
		return false, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT suggestion.id, suggestion.importer_did, suggestion.target_did,
		       COALESCE(evidence.username_normalized,'')
		FROM instagram_notification_suggestions support
		JOIN instagram_follow_suggestions suggestion ON suggestion.id=support.suggestion_id
		LEFT JOIN LATERAL (
			SELECT handle.username_normalized
			FROM instagram_suggestion_sources source
			JOIN instagram_graph_imports graph_import
			  ON graph_import.id=source.import_id
			 AND graph_import.owner_did=suggestion.importer_did
			 AND graph_import.state='active'
			JOIN instagram_graph_handles handle
			  ON handle.import_id=graph_import.id AND handle.matched
			JOIN instagram_account_links link
			  ON link.owner_did=suggestion.target_did
			 AND link.username_normalized=handle.username_normalized
			WHERE source.suggestion_id=suggestion.id
			ORDER BY source.created_at,handle.id
			LIMIT 1
		) evidence ON true
		WHERE support.notification_id=$1
		ORDER BY suggestion.id
	`, notificationID)
	if err != nil {
		return false, err
	}
	type supportedSuggestion struct {
		id      uuid.UUID
		request SuggestionEligibilityRequest
	}
	items := make([]supportedSuggestion, 0)
	for rows.Next() {
		var item supportedSuggestion
		var importer, target string
		if err := rows.Scan(&item.id, &importer, &target, &item.request.ImportedUsername); err != nil {
			rows.Close()
			return false, err
		}
		item.request.ImporterDID = syntax.DID(importer)
		item.request.TargetDID = syntax.DID(target)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return false, err
	}
	rows.Close()

	eligible := false
	invalid := make([]uuid.UUID, 0)
	for _, item := range items {
		decision, err := s.policy.Evaluate(ctx, stage, item.request)
		if err != nil {
			return false, err
		}
		if decision.Eligible {
			eligible = true
			continue
		}
		if decision.Reason != EligibilitySafetyUnavailable {
			invalid = append(invalid, item.id)
		}
	}
	if len(invalid) == 0 {
		return eligible, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	for _, suggestionID := range invalid {
		if err := s.notifications.RetractInstagramMatch(ctx, tx, notifications.InstagramMatchRetraction{
			SuggestionID: suggestionID,
			Reason:       "eligibility_changed",
		}); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return eligible, nil
}
