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
	SuggestionDefaultLimit    = 5
	SuggestionMaxLimit        = 25
	RecentSearchMaxLabelRunes = 120
	RecentSearchMaxPayloadLen = 4096
)

type SearchSort string

const (
	SearchSortChronological SearchSort = "chronological"
	SearchSortPopular       SearchSort = "popular"
)

var ErrSearchValidation = errors.New("search validation error")

type SearchSuggestionType string

const (
	SearchSuggestionTypeProfiles SearchSuggestionType = "profiles"
	SearchSuggestionTypeHashtags SearchSuggestionType = "hashtags"
)

type SearchSuggestionsRequest struct {
	Query        string
	Types        map[SearchSuggestionType]bool
	ProfileLimit int
	HashtagLimit int
}

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

type HashtagSearchRequest struct {
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

type ProjectListRequest struct {
	CraftTypes []string
	Filters    map[string][]string
	Sort       SearchSort
	Limit      int
	Cursor     string
}

type TopHashtagsRequest struct {
	CraftTypes []string
	Limit      int
}

type ExactHashtagPostsRequest struct {
	Tag    string
	Sort   SearchSort
	Limit  int
	Cursor string
}

type SaveRecentSearchRequest struct {
	Type         string          `json:"type"`
	DisplayLabel string          `json:"displayLabel"`
	Payload      json.RawMessage `json:"payload"`

	NormalizedPayload []byte
	PayloadHash       string
}

func ParseSearchSuggestionsRequest(r *http.Request) (SearchSuggestionsRequest, error) {
	q := r.URL.Query()
	if q.Has("cursor") {
		return SearchSuggestionsRequest{}, ErrSearchValidation
	}
	query := strings.TrimSpace(q.Get("q"))
	if query == "" || utf8.RuneCountInString(query) > SearchMaxQueryLength {
		return SearchSuggestionsRequest{}, ErrSearchValidation
	}
	profileLimit, err := parseBoundedSearchLimit(q.Get("profileLimit"), SuggestionDefaultLimit, SuggestionMaxLimit)
	if err != nil {
		return SearchSuggestionsRequest{}, err
	}
	hashtagLimit, err := parseBoundedSearchLimit(q.Get("hashtagLimit"), SuggestionDefaultLimit, SuggestionMaxLimit)
	if err != nil {
		return SearchSuggestionsRequest{}, err
	}
	types, err := parseSearchSuggestionTypes(q.Get("types"))
	if err != nil {
		return SearchSuggestionsRequest{}, err
	}
	return SearchSuggestionsRequest{Query: query, Types: types, ProfileLimit: profileLimit, HashtagLimit: hashtagLimit}, nil
}

func ParseExactHashtagPostsRequest(r *http.Request) (ExactHashtagPostsRequest, error) {
	tag, err := NormalizeHashtagPathValue(r.PathValue("tag"))
	if err != nil {
		return ExactHashtagPostsRequest{}, err
	}
	sort, err := parseSearchSort(r.URL.Query().Get("sort"))
	if err != nil {
		return ExactHashtagPostsRequest{}, err
	}
	limit, err := parseBoundedSearchLimit(r.URL.Query().Get("limit"), SearchDefaultLimit, SearchMaxLimit)
	if err != nil {
		return ExactHashtagPostsRequest{}, err
	}
	cursor := r.URL.Query().Get("cursor")
	if _, err := envelope.DecodeCursor(cursor); err != nil {
		return ExactHashtagPostsRequest{}, err
	}
	return ExactHashtagPostsRequest{Tag: tag, Sort: sort, Limit: limit, Cursor: cursor}, nil
}

func parseSearchSuggestionTypes(raw string) (map[SearchSuggestionType]bool, error) {
	if strings.TrimSpace(raw) == "" {
		return map[SearchSuggestionType]bool{
			SearchSuggestionTypeProfiles: true,
			SearchSuggestionTypeHashtags: true,
		}, nil
	}
	out := map[SearchSuggestionType]bool{}
	for _, part := range strings.Split(raw, ",") {
		switch typ := SearchSuggestionType(strings.ToLower(strings.TrimSpace(part))); typ {
		case SearchSuggestionTypeProfiles, SearchSuggestionTypeHashtags:
			out[typ] = true
		default:
			return nil, ErrSearchValidation
		}
	}
	if len(out) == 0 {
		return nil, ErrSearchValidation
	}
	return out, nil
}

func DecodeSaveRecentSearchRequest(r *http.Request) (SaveRecentSearchRequest, error) {
	var req SaveRecentSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return SaveRecentSearchRequest{}, ErrSearchValidation
	}
	req.Type = strings.ToLower(strings.TrimSpace(req.Type))
	if req.Type != "query" && req.Type != "hashtag" && req.Type != "profile" && req.Type != "post" && req.Type != "project" {
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
	case "query":
		canonical, err = normalizeRecentQueryPayload(payload, false)
	case "hashtag":
		canonical, err = normalizeRecentHashtagPayload(payload)
	case "profile":
		canonical, err = normalizeRecentProfilePayload(payload)
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
	if !onlyRecentKeys(payload, "tag") {
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
	return map[string]any{"tag": normalizedTag}, nil
}

func normalizeRecentProfilePayload(payload map[string]json.RawMessage) (map[string]any, error) {
	if !onlyRecentKeys(payload, "did", "handle", "displayName", "avatar") {
		return nil, ErrSearchValidation
	}
	did, err := rawString(payload, "did", true)
	if err != nil {
		return nil, err
	}
	did = strings.TrimSpace(did)
	if did == "" || !strings.HasPrefix(did, "did:") || utf8.RuneCountInString(did) > SearchMaxQueryLength {
		return nil, ErrSearchValidation
	}
	handle, err := rawString(payload, "handle", true)
	if err != nil {
		return nil, err
	}
	handle = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(handle, "@")))
	if handle == "" || utf8.RuneCountInString(handle) > SearchMaxQueryLength {
		return nil, ErrSearchValidation
	}
	out := map[string]any{"did": did, "handle": handle}
	if displayName, err := rawOptionalTrimmedString(payload, "displayName", RecentSearchMaxLabelRunes); err != nil {
		return nil, err
	} else if displayName != "" {
		out["displayName"] = displayName
	}
	if avatar, err := rawOptionalTrimmedString(payload, "avatar", RecentSearchMaxPayloadLen); err != nil {
		return nil, err
	} else if avatar != "" {
		out["avatar"] = avatar
	}
	return out, nil
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

func rawOptionalTrimmedString(payload map[string]json.RawMessage, key string, maxRunes int) (string, error) {
	if _, ok := payload[key]; !ok {
		return "", nil
	}
	value, err := rawString(payload, key, false)
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) > maxRunes {
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
	crafts, err := CanonicalCraftTypes(q["craftTypes"], true)
	if err != nil {
		return TopHashtagsRequest{}, err
	}
	return TopHashtagsRequest{CraftTypes: crafts, Limit: limit}, nil
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
	"q": true, "limit": true, "cursor": true,
}

func ParseProjectSearchRequest(r *http.Request) (ProjectSearchRequest, error) {
	q := r.URL.Query()
	for key := range q {
		if !allowedProjectQueryKeys[key] {
			return ProjectSearchRequest{}, ErrSearchValidation
		}
	}
	query, err := parseSearchQuery(q, true)
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
	return ProjectSearchRequest{Query: query, Sort: SearchSortChronological, Limit: limit, Cursor: cursor, Filters: map[string][]string{}}, nil
}

func ParseProjectListRequest(r *http.Request) (ProjectListRequest, error) {
	q := r.URL.Query()
	for key := range q {
		if key != "sort" && key != "limit" && key != "cursor" && !allowedProjectFilterKeys[key] {
			return ProjectListRequest{}, ErrSearchValidation
		}
	}
	sort, err := parseSearchSort(q.Get("sort"))
	if err != nil {
		return ProjectListRequest{}, err
	}
	limit, err := parseBoundedSearchLimit(q.Get("limit"), SearchDefaultLimit, SearchMaxLimit)
	if err != nil {
		return ProjectListRequest{}, err
	}
	cursor := q.Get("cursor")
	if _, err := envelope.DecodeCursor(cursor); err != nil {
		return ProjectListRequest{}, err
	}
	filters, err := parseProjectBrowseFilters(q)
	if err != nil {
		return ProjectListRequest{}, err
	}
	return ProjectListRequest{CraftTypes: filters["craftType"], Filters: filters, Sort: sort, Limit: limit, Cursor: cursor}, nil
}

func parseProjectBrowseFilters(q url.Values) (map[string][]string, error) {
	filters := map[string][]string{}
	total := 0
	for key, values := range q {
		if key == "sort" || key == "limit" || key == "cursor" {
			continue
		}
		if !allowedProjectFilterKeys[key] || len(values) == 0 || len(values) > ProjectFilterMaxPerFamily {
			return nil, ErrSearchValidation
		}
		if key == "craftType" {
			crafts, err := CanonicalCraftTypes(values, false)
			if err != nil {
				return nil, err
			}
			filters[key] = crafts
			total += len(crafts)
			continue
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

func ParseHashtagSearchRequest(r *http.Request) (HashtagSearchRequest, error) {
	q := r.URL.Query()
	for key := range q {
		if key != "q" && key != "limit" && key != "cursor" {
			return HashtagSearchRequest{}, ErrSearchValidation
		}
	}
	query, err := parseSearchQuery(q, true)
	if err != nil {
		return HashtagSearchRequest{}, err
	}
	limit, err := parseBoundedSearchLimit(q.Get("limit"), SearchDefaultLimit, SearchMaxLimit)
	if err != nil {
		return HashtagSearchRequest{}, err
	}
	cursor := q.Get("cursor")
	if _, err := DecodeHashtagSearchCursor(cursor, normalizeHashtagSearchTerm(query)); err != nil {
		return HashtagSearchRequest{}, err
	}
	return HashtagSearchRequest{Query: query, Limit: limit, Cursor: cursor}, nil
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
