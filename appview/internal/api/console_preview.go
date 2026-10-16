package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/moderation"
	"social.craftsky/appview/internal/video"
)

type ConsolePreview struct {
	State          string                 `json:"state"`
	SubjectType    string                 `json:"subjectType"`
	SubjectDID     syntax.DID             `json:"subjectDid"`
	MediaAvailable bool                   `json:"mediaAvailable"`
	Title          string                 `json:"title,omitempty"`
	Text           string                 `json:"text,omitempty"`
	Author         *ConsolePreviewAuthor  `json:"author,omitempty"`
	Images         []PostImageView        `json:"images,omitempty"`
	CreatedAt      *time.Time             `json:"createdAt,omitempty"`
	Banner         string                 `json:"banner,omitempty"`
	Crafts         []string               `json:"crafts,omitempty"`
	Event          *BusinessEventResponse `json:"event,omitempty"`
	Video          *PostVideoView         `json:"video,omitempty"`
	External       *ExternalResponse      `json:"external,omitempty"`
	Project        *Project               `json:"project,omitempty"`
}

type ConsolePreviewAuthor struct {
	DID         syntax.DID `json:"did"`
	DisplayName *string    `json:"displayName,omitempty"`
	Avatar      *string    `json:"avatar,omitempty"`
}

// Preview uses the indexed version captured by the case. Restricted incidents,
// blocked image projections, terminal owners and changed records expose no media.
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
		var kind, uri, cid string
		var did syntax.DID
		var restricted, terminal bool
		err = pool.QueryRow(r.Context(), `SELECT subject_type,subject_did,COALESCE(subject_uri,''),COALESCE(subject_cid_snapshot,''),origin='systemDetected' OR EXISTS(SELECT 1 FROM safety_incident_subjects s WHERE s.subject_uri=c.subject_uri OR (c.subject_type='account' AND s.owner_did=c.subject_did)),appview_owner_is_terminal(owner_did) FROM moderation_cases c WHERE id=$1`, ref.UUID()).Scan(&kind, &did, &uri, &cid, &restricted, &terminal)
		if err != nil {
			consoleReadError(w, r, err)
			return
		}
		result := ConsolePreview{State: "unavailable", SubjectType: kind, SubjectDID: did}
		if restricted {
			result.State = "restricted"
			writeSafetyCommandJSON(w, 200, result)
			return
		}
		if terminal {
			writeSafetyCommandJSON(w, 200, result)
			return
		}
		err = pgx.ErrNoRows
		switch kind {
		case "post":
			row := PostRow{DID: did.String()}
			err = pool.QueryRow(r.Context(), `SELECT p.text,p.images,p.created_at,b.display_name,b.avatar_cid,b.avatar_mime,p.record->'embed',j.raw_project FROM craftsky_posts p LEFT JOIN bluesky_profiles b ON b.did=p.did LEFT JOIN craftsky_project_posts j ON j.uri=p.uri WHERE p.uri=$1 AND p.cid=$2 AND EXISTS(SELECT 1 FROM image_subject_states s WHERE s.subject_uri=p.uri AND s.source_cid=p.cid AND s.visibility_state='clear')`, uri, cid).Scan(&row.Text, &row.Images, &row.CreatedAt, &row.AuthorDisplayName, &row.AuthorAvatarCID, &row.AuthorAvatarMime, &row.RawEmbed, &row.RawProject)
			if err == nil {
				result.Text = row.Text
				row.CreatedAt = row.CreatedAt.UTC()
				result.CreatedAt = &row.CreatedAt
				result.Author = &ConsolePreviewAuthor{DID: did, DisplayName: row.AuthorDisplayName}
				if avatar := synthBlobURL("avatar", did.String(), row.AuthorAvatarCID, row.AuthorAvatarMime); avatar != "" {
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
				// Consumer caption routes use a separate session. Preview playback carries
				// the authored description without exposing unusable consumer caption URLs.
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
			}
		case "event":
			var raw json.RawMessage
			err = pool.QueryRow(r.Context(), `SELECT raw_record FROM craftsky_business_events e WHERE uri=$1 AND cid=$2 AND EXISTS(SELECT 1 FROM image_subject_states s WHERE s.subject_uri=e.uri AND s.source_cid=e.cid AND s.visibility_state='clear')`, uri, cid).Scan(&raw)
			if err == nil {
				view, hydrateErr := business.HydrateEvent(raw)
				if hydrateErr != nil {
					err = pgx.ErrNoRows
					break
				}
				view.DID = did
				view.URI = syntax.ATURI(uri)
				view.Rkey = view.URI.RecordKey()
				view.CID = syntax.CID(cid)
				response := BuildBusinessEventResponse(view)
				result.Event = &response
				result.Title, result.Text = view.Name, view.Summary
				if image := response.Image; image != nil {
					result.Images = []PostImageView{{Fullsize: image.Fullsize, Thumb: image.Thumb, Alt: image.Alt, AspectRatio: image.AspectRatio}}
				}
				result.MediaAvailable = len(result.Images) > 0
			}
		case "account":
			var avatarCID, avatarMIME, bannerCID, bannerMIME *string
			err = pool.QueryRow(r.Context(), `SELECT COALESCE(b.description,''),COALESCE(b.display_name,''),b.avatar_cid,b.avatar_mime,b.banner_cid,b.banner_mime,COALESCE(c.crafts,ARRAY[]::text[]) FROM bluesky_profiles b LEFT JOIN craftsky_profiles c ON c.did=b.did WHERE b.did=$1 AND ($2='' OR b.record_cid=$2)`, did, cid).Scan(&result.Text, &result.Title, &avatarCID, &avatarMIME, &bannerCID, &bannerMIME, &result.Crafts)
			if err == nil {
				result.Author = &ConsolePreviewAuthor{DID: did, DisplayName: &result.Title}
				if avatar := synthBlobURL("avatar", did.String(), avatarCID, avatarMIME); avatar != "" {
					result.Author.Avatar = &avatar
				}
				result.Banner = synthBlobURL("banner", did.String(), bannerCID, bannerMIME)
				result.MediaAvailable = result.Author.Avatar != nil || result.Banner != ""
			}
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			consoleReadError(w, r, err)
			return
		}
		if err == nil {
			result.State = "available"
		}
		writeSafetyCommandJSON(w, 200, result)
	})
}
