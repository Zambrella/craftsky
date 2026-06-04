package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

const (
	facetDefaultLimit   = 10
	facetMaxLimit       = 25
	facetMaxQueryLength = 64
)

var ErrFacetValidation = errors.New("facet validation")

type FacetSuggestionRequest struct {
	Query string
	Limit int
}

func ParseFacetSuggestionRequest(r *http.Request) (FacetSuggestionRequest, error) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(query)) > facetMaxQueryLength {
		return FacetSuggestionRequest{}, ErrFacetValidation
	}
	limit := facetDefaultLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > facetMaxLimit {
			return FacetSuggestionRequest{}, ErrFacetValidation
		}
		limit = parsed
	}
	return FacetSuggestionRequest{Query: query, Limit: limit}, nil
}

func ParseFacetMentionHandle(r *http.Request) (syntax.Handle, error) {
	raw := strings.TrimPrefix(strings.TrimSpace(r.URL.Query().Get("handle")), "@")
	if raw == "" {
		return "", ErrMentionNotFound
	}
	handle, err := syntax.ParseHandle(raw)
	if err != nil {
		return "", ErrMentionNotFound
	}
	return handle, nil
}
