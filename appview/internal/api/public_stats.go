package api

import (
	"context"
	"net/http"
	"time"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

// PublicStats contains aggregate counts only. Projects are post snapshots, not
// distinct physical projects; a project can appear in more than one post.
type PublicStats struct {
	GeneratedAt          time.Time          `json:"generatedAt"`
	Accounts             int64              `json:"accounts"`
	Posts                int64              `json:"posts"`
	ProjectPosts         int64              `json:"projectPosts"`
	FinishedProjectPosts int64              `json:"finishedProjectPosts"`
	ProjectsByCraft      []PublicCraftCount `json:"projectsByCraft"`
}

type PublicCraftCount struct {
	Craft string `json:"craft"`
	Count int64  `json:"count"`
}

type PublicStatsReader interface {
	PublicStats(context.Context) (*PublicStats, error)
}

func PublicStatsHandler(store PublicStatsReader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		stats, err := store.PublicStats(ctx)
		if err != nil {
			observability.ReportRequestFailure(ctx, err, "api.PublicStatsHandler", "read")
			envelope.WriteError(w, http.StatusServiceUnavailable, "unavailable", "statistics temporarily unavailable", middleware.GetRunID(ctx), nil)
			return
		}
		if stats.ProjectsByCraft == nil {
			stats.ProjectsByCraft = []PublicCraftCount{}
		}
		writeJSON(w, http.StatusOK, stats)
	})
}
