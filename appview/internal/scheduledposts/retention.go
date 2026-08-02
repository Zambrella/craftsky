package scheduledposts

import "time"

const (
	UnclaimedMediaRetention = 24 * time.Hour
	PrivateContentRetention = 30 * 24 * time.Hour
	TombstoneRetention      = 30 * 24 * time.Hour
)

func UnclaimedMediaExpiresAt(createdAt time.Time) time.Time {
	return createdAt.UTC().Add(UnclaimedMediaRetention)
}

func NeedsAttentionExpiresAt(transitionedAt time.Time) time.Time {
	return transitionedAt.UTC().Add(PrivateContentRetention)
}

func PublicationTombstoneExpiresAt(publishedAt time.Time) time.Time {
	return publishedAt.UTC().Add(TombstoneRetention)
}

func SuccessfulPublicationCleanupAt(publishedAt time.Time) time.Time {
	return publishedAt.UTC()
}
