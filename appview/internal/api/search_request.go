package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"social.craftsky/appview/internal/api/envelope"
)

const (
	SearchDefaultLimit        = 25
	SearchMaxLimit            = 100
	SearchMaxQueryLength      = 256
	SearchMaxHashtagLength    = 128
	ProjectFilterMaxPerFamily = 10
	ProjectFilterMaxTotal     = 50
	TopHashtagDefaultLimit    = 10
	TopHashtagMaxLimit        = 50
	RecentSearchMaxLabelRunes = 120
	RecentSearchMaxPayloadLen = 4096
)

type SearchSort string

const (
	SearchSortChronological SearchSort = "chronological"
	SearchSortPopular       SearchSort = "popular"
)

var ErrSearchValidation = errors.New("search validation error")

type PostSearchRequest struct {
	Query  string
	Sort   SearchSort
	Limit  int
	Cursor string
}

type ProfileSearchRequest struct {
	Query  string
	Limit  int
	Cursor string
}

type ProjectSearchRequest struct {
	Query   string
	Sort    SearchSort
	Limit   int
	Cursor  string
	Filters map[string][]string
}

type TopHashtagsRequest struct {
	CraftTypes []string
	Limit      int
}

type SaveRecentSearchRequest struct {
	Type         string          `json:"type"`
	DisplayLabel string          `json:"displayLabel"`
	Payload      json.RawMessage `json:"payload"`

	NormalizedPayload []byte
	PayloadHash       string
}

func DecodeSaveRecentSearchRequest(r *http.Request) (SaveRecentSearchRequest, error) {
	var req SaveRecentSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return SaveRecentSearchRequest{}, ErrSearchValidation
	}
	req.Type = strings.ToLower(strings.TrimSpace(req.Type))
	if req.Type != "hashtag" && req.Type != "profile" && req.Type != "post" && req.Type != "project" {
		return SaveRecentSearchRequest{}, ErrSearchValidation
	}
	req.DisplayLabel = strings.TrimSpace(req.DisplayLabel)
	if req.DisplayLabel == "" || utf8.RuneCountInString(req.DisplayLabel) > RecentSearchMaxLabelRunes {
		return SaveRecentSearchRequest{}, ErrSearchValidation
	}
	if len(req.Payload) == 0 || len(req.Payload) > RecentSearchMaxPayloadLen {
		return SaveRecentSearchRequest{}, ErrSearchValidation
	}
	normalized, err := normalizeRecentPayload(req.Type, req.Payload)
	if err != nil {
		return SaveRecentSearchRequest{}, err
	}
	req.NormalizedPayload = normalized
	sum := sha256.Sum256(normalized)
	req.PayloadHash = hex.EncodeToString(sum[:])
	return req, nil
}

func normalizeRecentPayload(searchType string, raw json.RawMessage) ([]byte, error) {
	var payload map[string]json.RawMessage
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&payload); err != nil || payload == nil {
		return nil, ErrSearchValidation
	}
	var canonical any
	var err error
	switch searchType {
	case "hashtag":
		canonical, err = normalizeRecentHashtagPayload(payload)
	case "profile":
		canonical, err = normalizeRecentQueryPayload(payload, false)
	case "post":
		canonical, err = normalizeRecentQueryPayload(payload, true)
	case "project":
		canonical, err = normalizeRecentProjectPayload(payload)
	default:
		err = ErrSearchValidation
	}
	if err != nil {
		return nil, err
	}
	normalized, err := json.Marshal(canonical)
	if err != nil {
		return nil, ErrSearchValidation
	}
	if len(normalized) > RecentSearchMaxPayloadLen {
		return nil, ErrSearchValidation
	}
	return normalized, nil
}

func normalizeRecentHashtagPayload(payload map[string]json.RawMessage) (map[string]any, error) {
	if !onlyRecentKeys(payload, "tag", "sort") {
		return nil, ErrSearchValidation
	}
	tag, err := rawString(payload, "tag", true)
	if err != nil {
		return nil, err
	}
	normalizedTag, err := NormalizeHashtagPathValue(tag)
	if err != nil {
		return nil, err
	}
	sortValue, err := rawOptionalSort(payload)
	if err != nil {
		return nil, err
	}
	return map[string]any{"tag": normalizedTag, "sort": string(sortValue)}, nil
}

func normalizeRecentQueryPayload(payload map[string]json.RawMessage, allowSort bool) (map[string]any, error) {
	allowed := []string{"q"}
	if allowSort {
		allowed = append(allowed, "sort")
	}
	if !onlyRecentKeys(payload, allowed...) {
		return nil, ErrSearchValidation
	}
	q, err := rawString(payload, "q", true)
	if err != nil {
		return nil, err
	}
	q = strings.TrimSpace(q)
	if q == "" || utf8.RuneCountInString(q) > SearchMaxQueryLength {
		return nil, ErrSearchValidation
	}
	out := map[string]any{"q": q}
	if allowSort {
		sortValue, err := rawOptionalSort(payload)
		if err != nil {
			return nil, err
		}
		out["sort"] = string(sortValue)
	}
	return out, nil
}

func normalizeRecentProjectPayload(payload map[string]json.RawMessage) (map[string]any, error) {
	if !onlyRecentKeys(payload, "q", "sort", "filters") {
		return nil, ErrSearchValidation
	}
	out := map[string]any{}
	if _, ok := payload["q"]; ok {
		q, err := rawString(payload, "q", false)
		if err != nil {
			return nil, err
		}
		q = strings.TrimSpace(q)
		if utf8.RuneCountInString(q) > SearchMaxQueryLength {
			return nil, ErrSearchValidation
		}
		if q != "" {
			out["q"] = q
		}
	}
	sortValue, err := rawOptionalSort(payload)
	if err != nil {
		return nil, err
	}
	out["sort"] = string(sortValue)
	filters, err := normalizeRecentProjectFilters(payload["filters"])
	if err != nil {
		return nil, err
	}
	if len(filters) > 0 {
		out["filters"] = filters
	}
	return out, nil
}

func normalizeRecentProjectFilters(raw json.RawMessage) (map[string][]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return map[string][]string{}, nil
	}
	var input map[string][]string
	if err := json.Unmarshal(raw, &input); err != nil || input == nil {
		return nil, ErrSearchValidation
	}
	filters := map[string][]string{}
	total := 0
	for key, values := range input {
		if !allowedProjectFilterKeys[key] || len(values) == 0 || len(values) > ProjectFilterMaxPerFamily {
			return nil, ErrSearchValidation
		}
		seen := map[string]bool{}
		for _, value := range values {
			normalized := strings.ToLower(strings.TrimSpace(value))
			if normalized == "" || utf8.RuneCountInString(normalized) > SearchMaxQueryLength {
				return nil, ErrSearchValidation
			}
			if !seen[normalized] {
				filters[key] = append(filters[key], normalized)
				seen[normalized] = true
				total++
			}
		}
		sort.Strings(filters[key])
	}
	if total > ProjectFilterMaxTotal {
		return nil, ErrSearchValidation
	}
	return filters, nil
}

func rawOptionalSort(payload map[string]json.RawMessage) (SearchSort, error) {
	if _, ok := payload["sort"]; !ok {
		return SearchSortChronological, nil
	}
	s, err := rawString(payload, "sort", false)
	if err != nil {
		return "", err
	}
	return parseSearchSort(s)
}

func rawString(payload map[string]json.RawMessage, key string, required bool) (string, error) {
	raw, ok := payload[key]
	if !ok {
		if required {
			return "", ErrSearchValidation
		}
		return "", nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", ErrSearchValidation
	}
	return value, nil
}

func onlyRecentKeys(payload map[string]json.RawMessage, allowed ...string) bool {
	set := map[string]bool{}
	for _, key := range allowed {
		set[key] = true
	}
	for key := range payload {
		if !set[key] {
			return false
		}
	}
	return true
}

func ParseTopHashtagsRequest(r *http.Request) (TopHashtagsRequest, error) {
	q := r.URL.Query()
	limit, err := parseBoundedSearchLimit(q.Get("limit"), TopHashtagDefaultLimit, TopHashtagMaxLimit)
	if err != nil {
		return TopHashtagsRequest{}, err
	}
	crafts := q["craftTypes"]
	out := make([]string, 0, len(crafts))
	for _, craft := range crafts {
		normalized := strings.ToLower(strings.TrimSpace(craft))
		if normalized == "" || utf8.RuneCountInString(normalized) > SearchMaxQueryLength {
			return TopHashtagsRequest{}, ErrSearchValidation
		}
		out = append(out, normalized)
	}
	return TopHashtagsRequest{CraftTypes: out, Limit: limit}, nil
}

var allowedProjectFilterKeys = map[string]bool{
	"craftType":         true,
	"projectType":       true,
	"patternDifficulty": true,
	"color":             true,
	"material":          true,
	"designTag":         true,
	"projectTag":        true,
}

var allowedProjectQueryKeys = map[string]bool{
	"q": true, "sort": true, "limit": true, "cursor": true,
}

func ParseProjectSearchRequest(r *http.Request) (ProjectSearchRequest, error) {
	q := r.URL.Query()
	query, err := parseSearchQuery(q, false)
	if err != nil {
		return ProjectSearchRequest{}, err
	}
	sort, err := parseSearchSort(q.Get("sort"))
	if err != nil {
		return ProjectSearchRequest{}, err
	}
	limit, err := parseBoundedSearchLimit(q.Get("limit"), SearchDefaultLimit, SearchMaxLimit)
	if err != nil {
		return ProjectSearchRequest{}, err
	}
	cursor := q.Get("cursor")
	if _, err := envelope.DecodeCursor(cursor); err != nil {
		return ProjectSearchRequest{}, err
	}
	filters := map[string][]string{}
	total := 0
	for key, values := range q {
		if allowedProjectQueryKeys[key] {
			continue
		}
		if !allowedProjectFilterKeys[key] {
			return ProjectSearchRequest{}, ErrSearchValidation
		}
		if len(values) > ProjectFilterMaxPerFamily {
			return ProjectSearchRequest{}, ErrSearchValidation
		}
		for _, value := range values {
			normalized := strings.ToLower(strings.TrimSpace(value))
			if normalized == "" || utf8.RuneCountInString(normalized) > SearchMaxQueryLength {
				return ProjectSearchRequest{}, ErrSearchValidation
			}
			filters[key] = append(filters[key], normalized)
			total++
		}
	}
	if total > ProjectFilterMaxTotal {
		return ProjectSearchRequest{}, ErrSearchValidation
	}
	return ProjectSearchRequest{Query: query, Sort: sort, Limit: limit, Cursor: cursor, Filters: filters}, nil
}

func NormalizeHashtagPathValue(raw string) (string, error) {
	tag := strings.TrimSpace(raw)
	tag = strings.TrimPrefix(tag, "#")
	tag = strings.TrimSpace(tag)
	if tag == "" || utf8.RuneCountInString(tag) > SearchMaxHashtagLength {
		return "", ErrSearchValidation
	}
	for _, r := range tag {
		if r == '#' || r == '/' || unicode.IsSpace(r) || unicode.IsControl(r) {
			return "", ErrSearchValidation
		}
	}
	return strings.ToLower(tag), nil
}

func ParsePostSearchRequest(r *http.Request) (PostSearchRequest, error) {
	q := r.URL.Query()
	query, err := parseSearchQuery(q, true)
	if err != nil {
		return PostSearchRequest{}, err
	}
	sort, err := parseSearchSort(q.Get("sort"))
	if err != nil {
		return PostSearchRequest{}, err
	}
	limit, err := parseBoundedSearchLimit(q.Get("limit"), SearchDefaultLimit, SearchMaxLimit)
	if err != nil {
		return PostSearchRequest{}, err
	}
	cursor := q.Get("cursor")
	if _, err := envelope.DecodeCursor(cursor); err != nil {
		return PostSearchRequest{}, err
	}
	return PostSearchRequest{Query: query, Sort: sort, Limit: limit, Cursor: cursor}, nil
}

func ParseProfileSearchRequest(r *http.Request) (ProfileSearchRequest, error) {
	q := r.URL.Query()
	if q.Has("sort") {
		return ProfileSearchRequest{}, ErrSearchValidation
	}
	query, err := parseSearchQuery(q, true)
	if err != nil {
		return ProfileSearchRequest{}, err
	}
	limit, err := parseBoundedSearchLimit(q.Get("limit"), SearchDefaultLimit, SearchMaxLimit)
	if err != nil {
		return ProfileSearchRequest{}, err
	}
	cursor := q.Get("cursor")
	if _, err := envelope.DecodeCursor(cursor); err != nil {
		return ProfileSearchRequest{}, err
	}
	return ProfileSearchRequest{Query: query, Limit: limit, Cursor: cursor}, nil
}

func parseSearchQuery(values url.Values, required bool) (string, error) {
	query := strings.TrimSpace(values.Get("q"))
	if required && query == "" {
		return "", ErrSearchValidation
	}
	if utf8.RuneCountInString(query) > SearchMaxQueryLength {
		return "", ErrSearchValidation
	}
	return query, nil
}

func parseSearchSort(raw string) (SearchSort, error) {
	switch SearchSort(strings.TrimSpace(raw)) {
	case "", SearchSortChronological:
		return SearchSortChronological, nil
	case SearchSortPopular:
		return SearchSortPopular, nil
	default:
		return "", ErrSearchValidation
	}
}

func parseBoundedSearchLimit(raw string, defaultLimit, maxLimit int) (int, error) {
	if raw == "" {
		return defaultLimit, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 || limit > maxLimit {
		return 0, ErrSearchValidation
	}
	return limit, nil
}
