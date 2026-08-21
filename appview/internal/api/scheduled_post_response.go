package api

import (
	"time"

	"social.craftsky/appview/internal/scheduledposts"
)

type scheduledPostResponse struct {
	ID          string                 `json:"id"`
	OperationID string                 `json:"operationId"`
	Status      scheduledposts.Status  `json:"status"`
	ScheduledAt time.Time              `json:"scheduledAt"`
	Payload     scheduledposts.Payload `json:"payload"`
}

type scheduledPostSummaryResponse struct {
	ID                      string                  `json:"id"`
	Status                  scheduledposts.Status   `json:"status"`
	ScheduledAt             time.Time               `json:"scheduledAt"`
	Kind                    scheduledposts.PostKind `json:"kind"`
	ProjectTitle            string                  `json:"projectTitle,omitempty"`
	TextPreview             string                  `json:"textPreview"`
	FirstMediaID            string                  `json:"firstMediaId,omitempty"`
	NeedsAttentionExpiresAt *time.Time              `json:"needsAttentionExpiresAt,omitempty"`
}

type scheduledPostListResponse struct {
	Items               []scheduledPostSummaryResponse `json:"items"`
	Count               int                            `json:"count"`
	NeedsAttentionCount int                            `json:"needsAttentionCount"`
}
