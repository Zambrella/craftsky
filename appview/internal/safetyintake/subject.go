package safetyintake

import (
	"net/url"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

// CanonicalSubject normalizes shared record links at the intake boundary.
// It never fetches a URL or resolves an account supplied by a reporter.
func CanonicalSubject(raw string, kind Kind) (string, error) {
	subject := strings.TrimSpace(raw)
	if _, err := syntax.ParseATURI(subject); err == nil {
		return subject, nil
	}
	if kind == KindAppeal && strings.HasPrefix(subject, "case:") {
		if _, err := uuid.Parse(strings.TrimPrefix(subject, "case:")); err == nil {
			return subject, nil
		}
	}
	u, err := url.Parse(subject)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Fragment != "" || u.Path == "" || u.Path == "/" || u.Port() != "" && u.Port() != "443" {
		return "", ErrInvalidSubject
	}
	host := strings.ToLower(u.Hostname())
	if host != "app.craftsky.social" && host != "craftsky.social" && host != "www.craftsky.social" {
		return "", ErrInvalidSubject
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) > 0 && (parts[0] == "posts" || parts[0] == "profiles" || parts[0] == "events") {
		collection, rkey := "", "self"
		switch parts[0] {
		case "profiles":
			if len(parts) != 2 {
				return "", ErrInvalidSubject
			}
			collection = "app.bsky.actor.profile"
		case "posts", "events":
			if len(parts) != 3 {
				return "", ErrInvalidSubject
			}
			collection = "social.craftsky.feed.post"
			if parts[0] == "events" {
				collection = "social.craftsky.business.event"
			}
			key, err := syntax.ParseRecordKey(parts[2])
			if err != nil {
				return "", ErrInvalidSubject
			}
			rkey = key.String()
		}
		did, err := syntax.ParseDID(parts[1])
		if err != nil {
			return "", ErrInvalidSubject
		}
		query, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return "", ErrInvalidSubject
		}
		// A shared reply points to its root thread with the actual reply in focus.
		if focus, exists := query["focus"]; exists {
			if parts[0] != "posts" || len(focus) != 1 {
				return "", ErrInvalidSubject
			}
			uri, err := syntax.ParseATURI(focus[0])
			if err != nil || uri.Collection().String() != "social.craftsky.feed.post" || uri.RecordKey() == "" {
				return "", ErrInvalidSubject
			}
			if _, err := syntax.ParseDID(uri.Authority().String()); err != nil {
				return "", ErrInvalidSubject
			}
			return uri.String(), nil
		}
		uri, err := syntax.ParseATURI("at://" + did.String() + "/" + collection + "/" + rkey)
		if err != nil {
			return "", ErrInvalidSubject
		}
		return uri.String(), nil
	}
	// Preserve the previously accepted marketing/share-site references. They may
	// be recorded as intake, but still need an exact record match to open a case.
	if host == "craftsky.social" || host == "www.craftsky.social" {
		return subject, nil
	}
	return "", ErrInvalidSubject
}

func validCanonicalSubject(subject string, kind Kind) bool {
	_, err := CanonicalSubject(subject, kind)
	return err == nil
}
