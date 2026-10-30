package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/moderation"
	"social.craftsky/appview/internal/profilelinks"
	"social.craftsky/appview/internal/video"
)

type ConsolePreview struct {
	State          string                   `json:"state"`
	SubjectType    string                   `json:"subjectType"`
	SubjectDID     syntax.DID               `json:"subjectDid"`
	MediaAvailable bool                     `json:"mediaAvailable"`
	Title          string                   `json:"title,omitempty"`
	Text           string                   `json:"text,omitempty"`
	Author         *ConsolePreviewAuthor    `json:"author,omitempty"`
	Images         []PostImageView          `json:"images,omitempty"`
	CreatedAt      *time.Time               `json:"createdAt,omitempty"`
	Banner         string                   `json:"banner,omitempty"`
	Crafts         []string                 `json:"crafts,omitempty"`
	Event          *BusinessEventResponse   `json:"event,omitempty"`
	Video          *PostVideoView           `json:"video,omitempty"`
	External       *ExternalResponse        `json:"external,omitempty"`
	Project        *Project                 `json:"project,omitempty"`
	Context        []ConsolePreviewContext  `json:"context,omitempty"`
	Business       *BusinessProfileResponse `json:"business,omitempty"`
	BusinessState  string                   `json:"businessState,omitempty"`
	Version        *ConsolePreviewVersion   `json:"version,omitempty"`
	Current        *ConsolePreview          `json:"current,omitempty"`
}

type ConsolePreviewAuthor struct {
	DID         syntax.DID `json:"did"`
	DisplayName *string    `json:"displayName,omitempty"`
	Avatar      *string    `json:"avatar,omitempty"`
}

type ConsolePreviewContext struct {
	Kind    string         `json:"kind"`
	Preview ConsolePreview `json:"preview"`
}
type consolePreviewReference struct{ kind, uri, cid string }
type consolePreviewReader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type ConsolePreviewVersion struct {
	Basis        string                 `json:"basis"`
	Status       string                 `json:"status"`
	CapturedCID  string                 `json:"capturedCid,omitempty"`
	CurrentCID   string                 `json:"currentCid,omitempty"`
	CaseOpenedAt *time.Time             `json:"caseOpenedAt,omitempty"`
	Records      []ConsolePreviewRecord `json:"records,omitempty"`
}
type ConsolePreviewRecord struct {
	Collection string `json:"collection"`
	CID        string `json:"cid"`
}

func compareConsoleVersion(expected, current string) *ConsolePreviewVersion {
	v := &ConsolePreviewVersion{Basis: "reference", CapturedCID: expected, CurrentCID: current, Status: "matches"}
	if expected == "" {
		v.Basis = "current"
		v.Status = "current"
	} else if current == "" {
		v.Status = "missing"
	} else if expected != current {
		v.Status = "changed"
	}
	return v
}

// All case and context reads share one database snapshot. Each subject applies
// its own incident, owner lifecycle, image-safety and exact-version checks.
func ConsoleCasePreviewHandler(pool *pgxpool.Pool, playback PlaybackURLBuilder) http.Handler {
	if playback == nil {
		playback, _ = video.NewPlaybackURLBuilder(video.PlaybackConfig{})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ref, err := moderation.ParseCaseReference(r.PathValue("caseReference"))
		if err != nil {
			adminError(w, r, 400, "invalid_request", "invalid case reference")
			return
		}
		if pool == nil {
			adminError(w, r, 503, "admin_read_unavailable", "admin read unavailable")
			return
		}
		tx, err := pool.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
		if err != nil {
			consoleReadError(w, r, err)
			return
		}
		defer tx.Rollback(r.Context())
		result, err := readConsoleCase(r.Context(), tx, ref.UUID(), playback)
		if err == nil {
			err = tx.Commit(r.Context())
		}
		if err != nil {
			consoleReadError(w, r, err)
			return
		}
		writeSafetyCommandJSON(w, 200, result)
	})
}

func readConsoleCase(ctx context.Context, db consolePreviewReader, id uuid.UUID, playback PlaybackURLBuilder) (ConsolePreview, error) {
	var kind, uri, cid string
	var did syntax.DID
	var restricted, terminal bool
	var openedAt time.Time
	err := db.QueryRow(ctx, `SELECT subject_type,subject_did,COALESCE(subject_uri,''),COALESCE(subject_cid_snapshot,''),origin='systemDetected' OR EXISTS(SELECT 1 FROM safety_incident_subjects s WHERE s.subject_uri=c.subject_uri OR (c.subject_type='account' AND s.owner_did=c.subject_did)),appview_owner_is_terminal(owner_did),created_at FROM moderation_cases c WHERE id=$1`, id).Scan(&kind, &did, &uri, &cid, &restricted, &terminal, &openedAt)
	result := ConsolePreview{State: "unavailable", SubjectType: kind, SubjectDID: did}
	if err != nil {
		return result, err
	}
	if restricted {
		result.State = "restricted"
		return result, nil
	}
	if terminal {
		return result, nil
	}
	result, err = readConsoleSubject(ctx, db, kind, did, uri, cid, playback)
	if err != nil || result.State == "restricted" {
		return result, err
	}
	if result.Version != nil {
		result.Version.Basis = "case"
		openedAt = openedAt.UTC()
		result.Version.CaseOpenedAt = &openedAt
		if cid == "" {
			result.Version.Status = "untracked"
		}
		if result.Version.Status == "changed" {
			current, readErr := readConsoleSubject(ctx, db, kind, did, uri, "", playback)
			if readErr != nil {
				return result, readErr
			}
			result.Current = &current
		}
	}
	return result, nil
}
func readConsoleSubject(ctx context.Context, db consolePreviewReader, kind string, did syntax.DID, uri, cid string, playback PlaybackURLBuilder) (ConsolePreview, error) {
	switch kind {
	case "post":
		result, references, err := readConsolePost(ctx, db, uri, cid, playback)
		if result.SubjectDID == "" {
			result.SubjectDID = did
		}
		if err == nil && result.State == "available" {
			for _, reference := range references {
				related, _, readErr := readConsolePost(ctx, db, reference.uri, reference.cid, playback)
				if readErr != nil {
					return result, readErr
				}
				result.Context = append(result.Context, ConsolePreviewContext{Kind: reference.kind, Preview: related})
			}
		}
		return result, err
	case "account":
		return readConsoleAccount(ctx, db, did, cid)
	case "event":
		return readConsoleEvent(ctx, db, did, uri, cid)
	case "profileLink":
		result := ConsolePreview{State: "unavailable", SubjectType: kind, SubjectDID: did, Version: compareConsoleVersion(cid, "")}
		var raw json.RawMessage
		var currentCID string
		err := db.QueryRow(ctx, `SELECT cid,record FROM bluesky_profile_links WHERE uri=$1 AND did=$2`, uri, did).Scan(&currentCID, &raw)
		if errors.Is(err, pgx.ErrNoRows) {
			return result, nil
		}
		if err != nil {
			return result, err
		}
		result.Version = compareConsoleVersion(cid, currentCID)
		if result.Version.Status == "changed" {
			return result, nil
		}
		var record profilelinks.Record
		if err := json.Unmarshal(raw, &record); err != nil {
			return result, err
		}
		result.State = "available"
		result.Title = record.Title
		result.Text = record.URL
		return result, nil
	}
	return ConsolePreview{State: "unavailable", SubjectType: kind, SubjectDID: did}, nil
}

func readConsoleEvent(ctx context.Context, db consolePreviewReader, did syntax.DID, uri, expectedCID string) (ConsolePreview, error) {
	result := ConsolePreview{State: "unavailable", SubjectType: "event", SubjectDID: did, Version: compareConsoleVersion(expectedCID, "")}
	var raw json.RawMessage
	var currentCID string
	var restricted, terminal, clear bool
	err := db.QueryRow(ctx, `SELECT e.raw_record,e.cid,EXISTS(SELECT 1 FROM safety_incident_subjects s WHERE s.subject_uri=e.uri),appview_owner_is_terminal(e.owner_did),appview_image_subject_is_clear(e.uri,e.cid) FROM craftsky_business_events e WHERE e.uri=$1`, uri).Scan(&raw, &currentCID, &restricted, &terminal, &clear)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	if restricted || terminal {
		result.Version = nil
		if restricted {
			result.State = "restricted"
		}
		return result, nil
	}
	result.Version = compareConsoleVersion(expectedCID, currentCID)
	if !clear || expectedCID != "" && expectedCID != currentCID {
		return result, nil
	}
	view, err := business.HydrateEvent(raw)
	if err != nil {
		return result, nil
	}
	view.DID = did
	view.URI = syntax.ATURI(uri)
	view.Rkey = view.URI.RecordKey()
	view.CID = syntax.CID(currentCID)
	response := BuildBusinessEventResponse(view)
	result.Event = &response
	result.State = "available"
	result.Title, result.Text = view.Name, view.Summary
	if image := response.Image; image != nil {
		result.Images = []PostImageView{{Fullsize: image.Fullsize, Thumb: image.Thumb, Alt: image.Alt, AspectRatio: image.AspectRatio}}
	}
	result.MediaAvailable = len(result.Images) > 0
	return result, nil
}

func readConsolePost(ctx context.Context, db consolePreviewReader, uri, cid string, playback PlaybackURLBuilder) (ConsolePreview, []consolePreviewReference, error) {
	result := ConsolePreview{State: "unavailable", SubjectType: "post", Version: compareConsoleVersion(cid, "")}
	parsed, parseErr := syntax.ParseATURI(uri)
	if parseErr != nil || parsed.Collection() != "social.craftsky.feed.post" {
		return result, nil, nil
	}
	row := PostRow{URI: uri}
	var restricted, terminal, clear bool
	err := db.QueryRow(ctx, `SELECT p.did,p.cid,p.text,p.images,p.created_at,b.display_name,b.avatar_cid,b.avatar_mime,p.record->'embed',j.raw_project,p.quote_uri,p.quote_cid,p.reply_parent_uri,p.reply_parent_cid,p.reply_root_uri,p.reply_root_cid,
 EXISTS(SELECT 1 FROM safety_incident_subjects s WHERE s.subject_uri=p.uri),appview_owner_is_terminal(p.did),appview_image_subject_is_clear(p.uri,p.cid)
 FROM craftsky_posts p LEFT JOIN bluesky_profiles b ON b.did=p.did LEFT JOIN craftsky_project_posts j ON j.uri=p.uri WHERE p.uri=$1`, uri).Scan(&row.DID, &row.CID, &row.Text, &row.Images, &row.CreatedAt, &row.AuthorDisplayName, &row.AuthorAvatarCID, &row.AuthorAvatarMime, &row.RawEmbed, &row.RawProject, &row.QuoteURI, &row.QuoteCID, &row.ReplyParentURI, &row.ReplyParentCID, &row.ReplyRootURI, &row.ReplyRootCID, &restricted, &terminal, &clear)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, nil, nil
	}
	if err != nil {
		return result, nil, err
	}
	result.SubjectDID = syntax.DID(row.DID)
	if restricted {
		result.State = "restricted"
		result.Version = nil
		return result, nil, nil
	}
	if terminal {
		result.Version = nil
		return result, nil, nil
	}
	result.Version = compareConsoleVersion(cid, row.CID)
	if !clear || cid != "" && cid != row.CID {
		return result, nil, nil
	}
	result.State = "available"
	result.Text = row.Text
	row.CreatedAt = row.CreatedAt.UTC()
	result.CreatedAt = &row.CreatedAt
	result.Author = &ConsolePreviewAuthor{DID: result.SubjectDID, DisplayName: row.AuthorDisplayName}
	if avatar := synthBlobURL("avatar", row.DID, row.AuthorAvatarCID, row.AuthorAvatarMime); avatar != "" {
		result.Author.Avatar = &avatar
	}
	result.Images = buildPostImageViews(&row)
	for i := range result.Images {
		if name, ok := strings.CutPrefix(result.Images[i].CID, "devmedia:"); ok && validDevMediaName(name) {
			result.Images[i].Fullsize = "/v1/dev/media/" + name
			result.Images[i].Thumb = result.Images[i].Fullsize
		}
	}
	result.Video = buildPostVideoView(&row, playback)
	if result.Video != nil {
		result.Video.Captions = nil
	}
	if len(result.Images) == 0 {
		result.External = buildExternalResponse(&row)
	}
	if len(row.RawProject) > 0 {
		var project Project
		if json.Unmarshal(row.RawProject, &project) == nil {
			result.Project = &project
		}
	}
	result.MediaAvailable = len(result.Images) > 0 || result.Author.Avatar != nil || result.Video != nil || result.External != nil && result.External.Thumb != nil
	var references []consolePreviewReference
	for _, reference := range []struct {
		kind     string
		uri, cid *string
	}{{"quote", row.QuoteURI, row.QuoteCID}, {"parent", row.ReplyParentURI, row.ReplyParentCID}, {"root", row.ReplyRootURI, row.ReplyRootCID}} {
		if reference.uri == nil || reference.cid == nil || *reference.uri == uri {
			continue
		}
		if reference.kind == "root" && row.ReplyParentURI != nil && *reference.uri == *row.ReplyParentURI {
			continue
		}
		references = append(references, consolePreviewReference{reference.kind, *reference.uri, *reference.cid})
	}
	return result, references, nil
}

func ConsoleAccountPreviewHandler(pool *pgxpool.Pool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		did, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			adminError(w, r, 400, "invalid_request", "invalid account reference")
			return
		}
		if pool == nil {
			adminError(w, r, 503, "admin_read_unavailable", "admin read unavailable")
			return
		}
		result, err := readConsoleAccount(r.Context(), pool, did, "")
		if err != nil {
			consoleReadError(w, r, err)
			return
		}
		writeSafetyCommandJSON(w, 200, result)
	})
}

func readConsoleAccount(ctx context.Context, db consolePreviewReader, did syntax.DID, expectedCID string) (ConsolePreview, error) {
	result := ConsolePreview{State: "unavailable", SubjectType: "account", SubjectDID: did}
	var profileCID, craftCID, businessCID *string
	var displayName, description, avatarCID, avatarMIME, bannerCID, bannerMIME *string
	var crafts []string
	var raw json.RawMessage
	var terminal, restricted, businessClear bool
	err := db.QueryRow(ctx, `SELECT b.record_cid,c.record_cid,j.cid,b.display_name,b.description,b.avatar_cid,b.avatar_mime,b.banner_cid,b.banner_mime,COALESCE(c.crafts,ARRAY[]::text[]),j.raw_record,
 appview_owner_is_terminal(a.did),EXISTS(SELECT 1 FROM safety_incident_subjects s WHERE s.owner_did=a.did),COALESCE(appview_image_subject_is_clear(j.uri,j.cid),false)
 FROM (SELECT $1::text did) a LEFT JOIN bluesky_profiles b ON b.did=a.did LEFT JOIN craftsky_profiles c ON c.did=a.did LEFT JOIN craftsky_business_profiles j ON j.owner_did=a.did`, did).Scan(&profileCID, &craftCID, &businessCID, &displayName, &description, &avatarCID, &avatarMIME, &bannerCID, &bannerMIME, &crafts, &raw, &terminal, &restricted, &businessClear)
	if err != nil {
		return result, err
	}
	if restricted {
		result.State = "restricted"
		return result, nil
	}
	if terminal || profileCID == nil && craftCID == nil && businessCID == nil {
		return result, nil
	}
	result.Version = compareConsoleVersion(expectedCID, derefOrEmpty(profileCID))
	for _, record := range []struct {
		collection string
		cid        *string
	}{{"app.bsky.actor.profile", profileCID}, {"social.craftsky.actor.profile", craftCID}, {"social.craftsky.business.profile", businessCID}} {
		if record.cid != nil {
			result.Version.Records = append(result.Version.Records, ConsolePreviewRecord{Collection: record.collection, CID: *record.cid})
		}
	}
	if expectedCID != "" && (profileCID == nil || *profileCID != expectedCID) {
		return result, nil
	}
	result.State = "available"
	result.Title = derefOrEmpty(displayName)
	result.Text = derefOrEmpty(description)
	result.Crafts = crafts
	result.Author = &ConsolePreviewAuthor{DID: did, DisplayName: displayName}
	if avatar := synthBlobURL("avatar", did.String(), avatarCID, avatarMIME); avatar != "" {
		result.Author.Avatar = &avatar
	}
	result.Banner = synthBlobURL("banner", did.String(), bannerCID, bannerMIME)
	result.MediaAvailable = result.Author.Avatar != nil || result.Banner != ""
	if businessCID != nil {
		result.BusinessState = "unavailable"
		if businessClear {
			view, hydrateErr := business.HydrateProfile(raw)
			if hydrateErr == nil {
				view.CID = syntax.CID(*businessCID)
				result.Business = BuildBusinessProfileResponse(did, &view)
				result.BusinessState = "available"
				for _, product := range result.Business.Products {
					if product.Image != nil {
						result.MediaAvailable = true
					}
				}
			}
		}
	}
	return result, nil
}
