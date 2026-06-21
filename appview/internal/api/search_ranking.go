package api

import (
	"sort"
	"strings"
)

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

func RankHashtagResults(query string, rows []HashtagSuggestionRow) []HashtagSuggestionRow {
	q := normalizeHashtagSearchTerm(query)
	if q == "" {
		return []HashtagSuggestionRow{}
	}
	counts := map[string]int{}
	for _, row := range rows {
		tag := normalizeHashtagSearchTerm(row.Tag)
		if tag == "" || !strings.Contains(tag, q) {
			continue
		}
		count := row.PostsLast28Days
		if count < 0 {
			count = 0
		}
		counts[tag] += count
	}
	out := make([]HashtagSuggestionRow, 0, len(counts))
	for tag, count := range counts {
		out = append(out, HashtagSuggestionRow{Tag: tag, PostsLast28Days: count})
	}
	sort.SliceStable(out, func(i, j int) bool {
		ai := hashtagMatchRank(q, out[i].Tag)
		aj := hashtagMatchRank(q, out[j].Tag)
		if ai != aj {
			return ai < aj
		}
		if out[i].PostsLast28Days != out[j].PostsLast28Days {
			return out[i].PostsLast28Days > out[j].PostsLast28Days
		}
		return out[i].Tag < out[j].Tag
	})
	return out
}

func normalizeHashtagSearchTerm(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.TrimPrefix(value, "#")
	return strings.TrimSpace(value)
}

func hashtagMatchRank(query, tag string) int {
	switch {
	case tag == query:
		return 0
	case strings.HasPrefix(tag, query):
		return 1
	default:
		return 2
	}
}
