package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

type RecentMembersReader interface {
	ListRecentMembers(context.Context, syntax.DID) ([]RecentMemberRow, error)
}

// Dedicated allowlist: extending full profile summaries cannot expand discovery.
type recentMemberSummary struct {
	DID               syntax.DID    `json:"did"`
	Handle            syntax.Handle `json:"handle"`
	IsCraftskyProfile bool          `json:"isCraftskyProfile"`
	DisplayName       *string       `json:"displayName,omitempty"`
	Avatar            string        `json:"avatar,omitempty"`
}

func GetRecentMembersHandler(reader RecentMembersReader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		viewer, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "authenticated owner unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		ctx := observability.WithPrivateDiagnosticContext(r.Context(), viewer, "")
		rows, err := reader.ListRecentMembers(ctx, viewer)
		if err != nil {
			observability.ReportRequestFailure(ctx, err, "api.GetRecentMembersHandler", "query")
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "Unable to load recent members", middleware.GetRunID(r.Context()), nil)
			return
		}
		items := make([]recentMemberSummary, 0, len(rows))
		for _, row := range rows {
			var displayName *string
			if row.DisplayName != nil {
				name := strings.TrimSpace(*row.DisplayName)
				if name != "" {
					displayName = &name
				}
			}
			items = append(items, recentMemberSummary{
				DID:               row.DID,
				Handle:            row.Handle,
				IsCraftskyProfile: true,
				DisplayName:       displayName,
				Avatar:            synthBlobURL("avatar", row.DID.String(), row.AvatarCID, row.AvatarMime),
			})
		}
		w.Header().Set("Cache-Control", "no-store")
		envelope.WriteJSON(w, http.StatusOK, struct {
			Items []recentMemberSummary `json:"items"`
		}{Items: items})
	})
}
