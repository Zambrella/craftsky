package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"social.craftsky/appview/internal/lexicon/schema"
	"social.craftsky/appview/internal/middleware"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/profilelinks"
	"social.craftsky/appview/internal/recordjson"
)

type ProfileLinkInput struct {
	Rkey      syntax.RecordKey `json:"rkey"`
	CID       syntax.CID       `json:"cid,omitempty"`
	URL       string           `json:"url"`
	Title     string           `json:"title,omitempty"`
	Icon      map[string]any   `json:"icon,omitempty"`
	CreatedAt string           `json:"createdAt"`
}

func buildProfileLinkRecords(owner syntax.DID, inputs []ProfileLinkInput, current []pdscommands.AuthoritativeRecord) ([]json.RawMessage, []profilelinks.StrongRef, error) {
	records := make([]json.RawMessage, len(inputs))
	refs := make([]profilelinks.StrongRef, len(inputs))
	for i, input := range inputs {
		before := current[i]
		record := map[string]any{}
		if len(before.Record) > 0 {
			if err := recordjson.Unmarshal(before.Record, &record); err != nil {
				return nil, nil, err
			}
		}
		record["$type"] = profilelinks.Collection.String()
		record["url"] = input.URL
		record["createdAt"] = input.CreatedAt
		delete(record, "title")
		if input.Title != "" {
			record["title"] = input.Title
		}
		delete(record, "icon")
		if input.Icon != nil {
			record["icon"] = input.Icon
		}
		raw, err := json.Marshal(record)
		if err != nil {
			return nil, nil, err
		}
		if err := schema.ValidateCraftskyRecord(raw, profilelinks.Collection.String()); err != nil {
			return nil, nil, err
		}
		recordCID, err := profilelinks.RecordCID(raw)
		if err != nil {
			return nil, nil, err
		}
		if input.CID != before.CID && before.CID != recordCID {
			return nil, nil, pdscommands.ErrRepositoryConflict
		}
		records[i] = raw
		refs[i] = profilelinks.StrongRef{URI: syntax.ATURI("at://" + owner.String() + "/" + profilelinks.Collection.String() + "/" + input.Rkey.String()), CID: recordCID}
	}
	return records, refs, nil
}

func (s *ProfileStore) readProfileLinks(ctx context.Context, owner syntax.DID, viewer string) ([]profilelinks.View, []profilelinks.View, error) {
	rows, err := s.pool.Query(ctx, `
  SELECT link.uri,link.cid,link.record,
 NOT EXISTS(SELECT 1 FROM image_subject_states state WHERE state.subject_uri=link.uri AND state.visibility_state<>'clear') AS image_clear,
 NOT EXISTS(
  SELECT 1 FROM moderation_outputs mo
  WHERE mo.subject_type='profileLink' AND mo.subject_uri=link.uri AND mo.action='apply'
   AND mo.value IN ('hide','takedown','warn') AND NOT appview_owner_is_terminal(mo.source_did)
   AND (mo.expires_at IS NULL OR mo.expires_at>now())
   AND NOT EXISTS(SELECT 1 FROM moderation_outputs neg WHERE neg.subject_type=mo.subject_type
    AND neg.subject_uri=mo.subject_uri AND neg.source_did=mo.source_did AND neg.value=mo.value
    AND neg.action='negate' AND neg.indexed_at>mo.indexed_at AND (neg.expires_at IS NULL OR neg.expires_at>now()))
 ) AS unmoderated
  FROM bluesky_profiles profile
  CROSS JOIN LATERAL jsonb_array_elements(profile.links) WITH ORDINALITY ref(value,position)
  JOIN bluesky_profile_links link ON link.uri=ref.value->>'uri' AND link.did=profile.did
  WHERE profile.did=$1 AND NOT appview_owner_is_terminal(link.did)
  ORDER BY ref.position LIMIT 10`, owner)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	links := []profilelinks.View{}
	editable := []profilelinks.View{}
	seen := map[string]bool{}
	for rows.Next() {
		var view profilelinks.View
		var raw json.RawMessage
		var record profilelinks.Record
		var clear, unmoderated bool
		if err := rows.Scan(&view.URI, &view.CID, &raw, &clear, &unmoderated); err != nil {
			return nil, nil, err
		}
		if err := json.Unmarshal(raw, &record); err != nil {
			return nil, nil, err
		}
		normalized, err := profilelinks.Normalize(record.URL)
		if err != nil || seen[normalized] || !profilelinks.OwnRef(owner, view.URI) {
			continue
		}
		seen[normalized] = true
		view.URL = record.URL
		view.Title = profilelinks.Title(record.Title, 30)
		if record.Icon != nil && clear {
			cid, mime := record.Icon.Ref.String(), record.Icon.MimeType
			view.Icon = synthBlobURL("avatar", owner.String(), &cid, &mime)
		}
		if viewer == owner.String() {
			view.IconBlob = record.Icon
			view.CreatedAt = record.CreatedAt
			editable = append(editable, view)
		}
		var labels struct {
			Values []struct {
				Val string `json:"val"`
			} `json:"values"`
		}
		if len(record.Labels) > 0 && json.Unmarshal(record.Labels, &labels) != nil {
			continue
		}
		if !clear || !unmoderated || len(labels.Values) > 0 {
			continue
		}
		links = append(links, view)
	}
	return links, editable, rows.Err()
}

func acceptedLinkViews(records []pdscommands.AuthoritativeRecord) ([]profilelinks.View, error) {
	views := []profilelinks.View{}
	for _, saved := range records {
		var record profilelinks.Record
		if err := json.Unmarshal(saved.Record, &record); err != nil {
			return nil, err
		}
		if record.Type != profilelinks.Collection.String() {
			return nil, errors.New("unexpected profile link record")
		}
		// Icon URL is read-owned until image scanning has cleared the blob.
		views = append(views, profilelinks.View{URI: saved.URI, CID: saved.CID, URL: record.URL, Title: record.Title, IconBlob: record.Icon, CreatedAt: record.CreatedAt})
	}
	return views, nil
}

// Save acknowledgements must not bypass moderation or expose a pending icon
// through the optimistic client overlay. Original blobs remain editor-owned.
func acceptedPublicProfileLinks(records []pdscommands.AuthoritativeRecord, editable []profilelinks.View, inputs []ProfileLinkInput, prior *ProfileRow) []profilelinks.View {
	result := []profilelinks.View{}
	previous := map[syntax.ATURI]profilelinks.View{}
	if prior != nil {
		for _, view := range prior.Links {
			previous[view.URI] = view
		}
	}
	for i, view := range editable {
		var record profilelinks.Record
		if json.Unmarshal(records[i].Record, &record) != nil {
			continue
		}
		var labels struct {
			Values []json.RawMessage `json:"values"`
		}
		if len(record.Labels) > 0 && (json.Unmarshal(record.Labels, &labels) != nil || len(labels.Values) > 0) {
			continue
		}
		old, visible := previous[view.URI]
		if inputs[i].CID != "" && !visible {
			continue
		}
		if view.IconBlob != nil {
			if !visible || old.IconBlob == nil || old.IconBlob.Ref != view.IconBlob.Ref {
				continue
			}
			view.Icon = old.Icon
		}
		result = append(result, view)
	}
	return result
}

// Only links actually visible on the resolved profile can be reported.
type profileLinkReportResolver struct{ store *ProfileStore }

func (resolver profileLinkReportResolver) ResolvePostReportTarget(ctx context.Context, owner syntax.DID, rkey syntax.RecordKey) (*PostReportTarget, error) {
	viewer, _ := middleware.GetDID(ctx)
	profile, err := resolver.store.Read(ctx, owner.String(), viewer.String())
	if err != nil {
		return nil, err
	}
	uri := "at://" + owner.String() + "/app.bsky.actor.link/" + rkey.String()
	for _, link := range profile.Links {
		if link.URI.String() == uri {
			return &PostReportTarget{DID: owner.String(), Rkey: rkey.String(), URI: uri, CIDSnapshot: link.CID.String()}, nil
		}
	}
	return nil, ErrPostNotFound
}
func ReportProfileLinkHandler(store *ProfileStore, reports ReportCreator, forwarder ReportForwarder, logger *slog.Logger) http.Handler {
	return reportRecordHandler(profileLinkReportResolver{store}, reports, forwarder, logger, ReportSubjectProfileLink, "app.bsky.actor.link")
}
