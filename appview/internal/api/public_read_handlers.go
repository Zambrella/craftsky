package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

type PublicBrowsingReader interface {
	PublicPageReader
	PublicTagPosts(context.Context, string, int, string) (*PublicPostPage, error)
	PublicEvent(context.Context, syntax.DID, syntax.RecordKey) (*PublicEvent, error)
	PublicEvents(context.Context, syntax.DID, int, string) (*PublicEventPage, error)
	PublicInteractions(context.Context, syntax.DID, syntax.RecordKey, PostInteractionKind, int, string) (any, error)
	PublicProjects(context.Context, ProjectListRequest) (*PublicPostPage, error)
	PublicSearch(context.Context, string, string, int, string) (any, error)
	PublicProfilePosts(context.Context, syntax.DID, string, int, string) (*PublicPostPage, error)
	PublicThread(context.Context, syntax.DID, syntax.RecordKey, int, string) (*PublicThread, error)
}

func PublicBrowsingHandler(store PublicBrowsingReader, resolver HandleResolver, resource string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		fail := func(status int, code, message string) {
			envelope.WriteError(w, status, code, message, middleware.GetRunID(ctx), nil)
		}
		query := r.URL.Query()
		limit := 20
		var result any
		var err error
		if resource == "projectFeed" {
			if len(query["limit"]) > 1 || len(query["cursor"]) > 1 || len(query["sort"]) > 1 || len(query.Get("cursor")) > 2048 {
				fail(400, "invalid_request", "invalid public request")
				return
			}
			limit, err = parseBoundedSearchLimit(query.Get("limit"), 20, 20)
			if err != nil {
				fail(400, "invalid_request", "invalid public request")
				return
			}
			request, parseErr := ParseProjectListRequest(r)
			if parseErr != nil {
				fail(400, "invalid_request", "invalid public request")
				return
			}
			request.Limit = limit
			result, err = store.PublicProjects(ctx, request)
		} else {
			allowed := []string{"limit", "cursor"}
			if resource == "event" {
				allowed = nil
			}
			if strings.HasPrefix(resource, "search/") {
				allowed = append(allowed, "q")
			}
			limit, err = validatePublicQuery(query, allowed...)
			if err != nil {
				fail(400, "invalid_request", "invalid public request")
				return
			}
			if strings.HasPrefix(resource, "search/") {
				if _, parseErr := parseSearchQuery(query, true); parseErr != nil {
					fail(400, "invalid_request", "enter a search query")
					return
				}
				result, err = store.PublicSearch(ctx, strings.TrimPrefix(resource, "search/"), query.Get("q"), limit, query.Get("cursor"))
			} else if resource == "tagPosts" {
				tag, parseErr := NormalizeHashtagPathValue(r.PathValue("tag"))
				if parseErr != nil {
					fail(400, "invalid_request", "invalid hashtag")
					return
				}
				result, err = store.PublicTagPosts(ctx, tag, limit, query.Get("cursor"))
			} else {
				postResource := resource == "thread" || resource == "event" || strings.HasPrefix(resource, "interactions/")
				raw := strings.TrimPrefix(r.PathValue("handleOrDid"), "@")
				if postResource {
					raw = r.PathValue("did")
				}
				if raw == "me" || raw == "recent" {
					fail(404, "not_found", "page not found")
					return
				}
				did, resolveErr := resolveToDID(ctx, raw, resolver)
				if resolveErr != nil {
					if errors.Is(resolveErr, errInvalidIdentifier) {
						fail(400, "invalid_identifier", "invalid profile identifier")
					} else {
						fail(503, "identity_unavailable", "identity temporarily unavailable")
					}
					return
				}
				if postResource {
					key, parseErr := syntax.ParseRecordKey(r.PathValue("rkey"))
					if parseErr != nil {
						fail(400, "invalid_identifier", "invalid post identifier")
						return
					}
					switch resource {
					case "thread":
						result, err = store.PublicThread(ctx, did, key, limit, query.Get("cursor"))
					case "event":
						result, err = store.PublicEvent(ctx, did, key)
					default:
						result, err = store.PublicInteractions(ctx, did, key, PostInteractionKind(strings.TrimPrefix(resource, "interactions/")), limit, query.Get("cursor"))
					}
				} else if resource == "events" {
					result, err = store.PublicEvents(ctx, did, limit, query.Get("cursor"))
				} else {
					result, err = store.PublicProfilePosts(ctx, did, resource, limit, query.Get("cursor"))
				}
			}
		}
		if err != nil {
			writePublicReadError(w, r, err)
			return
		}
		writeJSON(w, 200, result)
	})
}

func writePublicReadError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := 503, "unavailable", "page temporarily unavailable"
	switch {
	case errors.Is(err, ErrPublicAuthenticationRequired):
		status, code, message = 403, "authentication_required", "Sign in to view this content."
	case errors.Is(err, ErrProfileNotFound) || errors.Is(err, ErrPostNotFound):
		status, code, message = 404, "not_found", "page not found"
	case errors.Is(err, ErrSearchValidation) || errors.Is(err, envelope.ErrInvalidCursor):
		status, code, message = 400, "invalid_request", "invalid public request"
	default:
		observability.ReportRequestFailure(r.Context(), err, "api.PublicBrowsingHandler", "read")
	}
	envelope.WriteError(w, status, code, message, middleware.GetRunID(r.Context()), nil)
}
