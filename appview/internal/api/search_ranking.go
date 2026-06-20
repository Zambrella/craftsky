package api

import "strings"

type ProfileRankTuple struct {
	FollowedRank  int
	RelevanceRank int
	HandleLower   string
	DID           string
}

func ProfileSearchRankTuple(viewerIsFollowing bool, relevanceRank int, handleLower, did string) ProfileRankTuple {
	followedRank := 1
	if viewerIsFollowing {
		followedRank = 0
	}
	return ProfileRankTuple{
		FollowedRank:  followedRank,
		RelevanceRank: relevanceRank,
		HandleLower:   strings.ToLower(handleLower),
		DID:           did,
	}
}

func (t ProfileRankTuple) Less(other ProfileRankTuple) bool {
	if t.FollowedRank != other.FollowedRank {
		return t.FollowedRank < other.FollowedRank
	}
	if t.RelevanceRank != other.RelevanceRank {
		return t.RelevanceRank < other.RelevanceRank
	}
	if t.HandleLower != other.HandleLower {
		return t.HandleLower < other.HandleLower
	}
	return t.DID < other.DID
}

func ProfileRelevanceRank(query, handle, displayName, description string) (int, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return 0, false
	}
	h := strings.ToLower(handle)
	switch {
	case h == q:
		return 0, true
	case strings.HasPrefix(h, q):
		return 1, true
	case strings.Contains(h, q):
		return 2, true
	case strings.Contains(strings.ToLower(displayName), q):
		return 3, true
	case strings.Contains(strings.ToLower(description), q):
		return 4, true
	default:
		return 0, false
	}
}
