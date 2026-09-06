// appview/internal/api/post_create.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/postrecord"
	"social.craftsky/appview/internal/postutil"
	"social.craftsky/appview/internal/relationships"
	"social.craftsky/appview/internal/video"
)

type postAuthorReader interface {
	ReadAuthor(context.Context, string) (*PostAuthorRow, error)
}

type createPostStore interface {
	DirectedInteractionAuthorizer
	shareTargetReader
	postAuthorReader
	postQuoteHydrationStore
}

type VideoCompletionVerifier interface {
	Verify(context.Context, syntax.DID, string, video.Blob) (video.Blob, error)
}

type CreatePostHandlerOptions struct {
	VideoCompletionVerifier VideoCompletionVerifier
}

// CreatePostHandler serves POST /v1/posts.
func CreatePostHandler(
	store createPostStore,
	newEffects pdseffects.ExecutorFactory,
	resolver HandleResolver,
	limits MediaLimits,
	logger *slog.Logger,
	options ...CreatePostHandlerOptions,
) http.Handler {
	limits = normalizeMediaLimits(limits)
	var videoVerifier VideoCompletionVerifier
	if len(options) > 0 {
		videoVerifier = options[0].VideoCompletionVerifier
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		ownerGeneration, ok := requirePDSEffectGeneration(w, r, runID)
		if !ok {
			return
		}
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		logger.Debug("post create: request started",
			pdsLogAttrs(runID, pdsOperationPostCreate, pdsStageRequestBuild)...)

		req, err := DecodePostCreate(r.Body)
		if err != nil {
			fe, isFE := err.(*FieldError)
			switch {
			case isFE && fe.Code == "malformed_body":
				envelope.WriteError(w, http.StatusBadRequest,
					"malformed_body", "could not parse body", runID, fe.Fields)
			case isFE:
				envelope.WriteError(w, http.StatusBadRequest,
					fe.Code, "request body rejected", runID, fe.Fields)
			default:
				envelope.WriteError(w, http.StatusBadRequest,
					"malformed_body", "could not parse body", runID, nil)
			}
			return
		}
		if err := ValidatePostCreateWithLimits(req, limits); err != nil {
			fe, isFE := err.(*FieldError)
			if isFE {
				status := http.StatusUnprocessableEntity
				if _, hasLanguageError := fe.Fields["langs"]; hasLanguageError {
					status = http.StatusBadRequest
				}
				envelope.WriteError(w, status,
					fe.Code, "validation failed", runID, fe.Fields)
				return
			}
			envelope.WriteError(w, http.StatusUnprocessableEntity,
				"validation_failed", "validation failed", runID, nil)
			return
		}
		var verifiedVideo *video.Blob
		if req.Embed != nil && req.Embed.Video != nil {
			if videoVerifier == nil {
				envelope.WriteError(w, http.StatusBadGateway,
					"video_service_unavailable", "could not verify video", runID, nil)
				return
			}
			submitted := video.Blob{
				CID:      syntax.CID(req.Embed.Video.Blob.Ref.Link),
				MIMEType: req.Embed.Video.Blob.MIMEType,
				Size:     req.Embed.Video.Blob.Size,
			}
			verified, err := videoVerifier.Verify(r.Context(), did, req.Embed.Video.JobID, submitted)
			if err != nil {
				var verificationError *video.VerificationError
				if errors.As(err, &verificationError) && verificationError.Kind == video.VerificationRejected {
					envelope.WriteError(w, http.StatusUnprocessableEntity,
						"video_verification_failed", "video could not be verified", runID, nil)
					return
				}
				envelope.WriteError(w, http.StatusBadGateway,
					"video_service_unavailable", "could not verify video", runID, nil)
				return
			}
			verifiedVideo = &verified
		}
		effectTargets := make([]syntax.DID, 0, 2)
		if req.Embed != nil && req.Embed.Quote != nil {
			target, err := validateQuoteShareTarget(r.Context(), store, *req.Embed.Quote)
			if err != nil {
				fe, isFE := err.(*FieldError)
				if isFE {
					envelope.WriteError(w, http.StatusUnprocessableEntity,
						fe.Code, "validation failed", runID, fe.Fields)
					return
				}
				if errors.Is(err, ErrPostNotFound) {
					envelope.WriteError(w, http.StatusNotFound,
						"post_not_found", "post not found", runID, nil)
					return
				}
				logger.Error("post: validate quote target failed",
					pdsLogErrorAttrs(runID, pdsOperationPostCreate, pdsStageRequestBuild, err)...)
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "could not resolve quote target", runID, nil)
				return
			}
			req.Embed.Quote.URI = target.URI
			req.Embed.Quote.CID = target.CID
			quoteDID, _ := subjectDIDFromATURI(target.URI)
			if !authorizeDirectedInteraction(w, r, store, did, quoteDID, relationships.OperationQuoteCreate) {
				return
			}
			effectTargets = append(effectTargets, quoteDID)
		}
		if req.Reply != nil {
			replyDID, err := subjectDIDFromATURI(req.Reply.Parent.URI)
			if err != nil {
				envelope.WriteError(w, http.StatusUnprocessableEntity,
					"validation_failed", "validation failed", runID, map[string]string{"reply.parent.uri": "must identify a DID-authored post"})
				return
			}
			if !authorizeDirectedInteraction(w, r, store, did, replyDID, relationships.OperationReplyCreate) {
				return
			}
			rootDID, err := subjectDIDFromATURI(req.Reply.Root.URI)
			if err != nil {
				envelope.WriteError(w, http.StatusUnprocessableEntity,
					"validation_failed", "validation failed", runID, map[string]string{"reply.root.uri": "must identify a DID-authored post"})
				return
			}
			effectTargets = append(effectTargets, replyDID, rootDID)
		}
		mentioned, err := mentionedDIDs(req)
		if err != nil {
			envelope.WriteError(w, http.StatusUnprocessableEntity,
				"validation_failed", "validation failed", runID, map[string]string{"facets": "mention DID is invalid"})
			return
		}
		for _, subject := range mentioned {
			if !authorizeDirectedInteraction(w, r, store, did, subject, relationships.OperationMentionCreate) {
				return
			}
			effectTargets = append(effectTargets, subject)
		}
		logger.Debug("post create: validated request",
			pdsLogAttrs(runID, pdsOperationPostCreate, pdsStageRequestBuild)...)

		body, err := lexiconRecordBody(req, verifiedVideo)
		if err != nil {
			logger.Error("post create: generated record construction failed",
				pdsLogErrorAttrs(runID, pdsOperationPostCreate, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not prepare post", runID, nil)
			return
		}
		logger.Debug("post create: prepared PDS record",
			pdsLogAttrs(runID, pdsOperationPostCreate, pdsStageRequestBuild)...)

		executor, err := newPDSEffectExecutor(r.Context(), newEffects, did, sessionID)
		if err != nil {
			logger.Error("post: durable effect executor unavailable",
				pdsLogErrorAttrs(runID, pdsOperationPostCreate, pdsStageSessionResume, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, err)
			return
		}
		expectedOwners, err := executor.ResolveExpectedOwners(r.Context(), ownerGeneration, effectTargets)
		if err != nil {
			logger.Warn("post: durable effect scope rejected",
				pdsLogErrorAttrs(runID, pdsOperationPostCreate, pdsStageRequestBuild, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_write_failed", "could not write post", runID, err)
			return
		}
		rkey, err := newImmediateRecordKey()
		if err != nil {
			logger.Error("post: allocate record key failed",
				pdsLogErrorAttrs(runID, pdsOperationPostCreate, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not prepare post", runID, nil)
			return
		}
		operationID, mutationKey := immediateEffectIdentity(runID, "post.create")
		result, err := executor.PutRecord(r.Context(), pdseffects.PutRecordRequest{
			OperationID: operationID, MutationKey: mutationKey,
			Owner: did, OwnerGeneration: ownerGeneration, ExpectedOwners: expectedOwners,
			Collection: syntax.NSID(craftskyPostNSID), Rkey: rkey, Record: body,
		})
		if err != nil {
			logger.Warn("post: durable PDS put failed",
				pdsLogErrorAttrs(runID, pdsOperationPostCreate, pdsStagePDSRequest, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_write_failed", "could not write post", runID, err)
			return
		}
		logger.Debug("post create: PDS record created",
			pdsLogSuccessAttrs(runID, pdsOperationPostCreate, pdsStagePDSRequest)...)

		row, err := syntheticPostRow(r, store, did, result.URI, result.CID, req, verifiedVideo)
		if err != nil {
			logger.Error("post: hydrate author failed",
				pdsLogErrorAttrs(runID, pdsOperationPostCreate, pdsStagePDSRequest, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post created but hydrate failed", runID, nil)
			return
		}
		handle, err := resolver.ResolveHandle(r.Context(), did)
		if err != nil {
			logger.Warn("post: ResolveHandle failed",
				pdsLogErrorAttrs(runID, pdsOperationPostCreate, pdsStagePDSRequest, err)...)
			envelope.WriteError(w, http.StatusBadGateway,
				"identity_unavailable", "could not resolve handle", runID, nil)
			return
		}
		resp := buildPostResponse(row, handle, store)
		if err := attachQuoteView(r.Context(), store, resolver, resp); err != nil {
			logger.Error("post create: QuoteViewRows failed",
				pdsLogErrorAttrs(runID, pdsOperationPostCreate, pdsStagePDSRequest, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post created but quote lookup failed", runID, nil)
			return
		}
		logger.Debug("post create: response ready",
			pdsLogSuccessAttrs(runID, pdsOperationPostCreate, pdsStagePDSRequest)...)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	})
}

func validateQuoteShareTarget(ctx context.Context, store shareTargetReader, ref StrongRef) (*ShareTargetRef, error) {
	aturi, err := syntax.ParseATURI(ref.URI)
	if err != nil {
		return nil, &FieldError{Code: "validation_failed", Fields: map[string]string{"embed.quote.uri": "must be a valid AT-URI"}}
	}
	if aturi.Collection().String() != craftskyPostNSID || aturi.RecordKey().String() == "" {
		return nil, &FieldError{Code: "validation_failed", Fields: map[string]string{"target": "must reference a Craftsky post"}}
	}
	did, err := syntax.ParseDID(aturi.Authority().String())
	if err != nil {
		return nil, &FieldError{Code: "validation_failed", Fields: map[string]string{"target": "must reference a DID-authored Craftsky post"}}
	}
	target, err := store.ResolveShareTarget(ctx, did.String(), aturi.RecordKey().String())
	if err != nil {
		return nil, err
	}
	if target.IsReply {
		return nil, &FieldError{Code: "validation_failed", Fields: map[string]string{"target": "reply posts cannot be quoted"}}
	}
	return target, nil
}

func subjectDIDFromATURI(uri string) (syntax.DID, error) {
	aturi, err := syntax.ParseATURI(uri)
	if err != nil {
		return "", err
	}
	return syntax.ParseDID(aturi.Authority().String())
}

func mentionedDIDs(req PostCreateRequest) ([]syntax.DID, error) {
	raw, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	seen := map[syntax.DID]struct{}{}
	var walk func(any) error
	walk = func(node any) error {
		switch typed := node.(type) {
		case []any:
			for _, item := range typed {
				if err := walk(item); err != nil {
					return err
				}
			}
		case map[string]any:
			if typed["$type"] == "app.bsky.richtext.facet#mention" {
				did, err := syntax.ParseDID(fmt.Sprint(typed["did"]))
				if err != nil {
					return err
				}
				seen[did] = struct{}{}
			}
			for _, item := range typed {
				if err := walk(item); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(value); err != nil {
		return nil, err
	}
	out := make([]syntax.DID, 0, len(seen))
	for did := range seen {
		out = append(out, did)
	}
	slices.Sort(out)
	return out, nil
}

// lexiconRecordBody translates the wire request into the lexicon-shaped
// record body that goes to the PDS. Facets are pass-through raw JSON so
// the PDS sees exactly what the client sent (including any "$type"
// discriminators on union variants).
func lexiconRecordBody(req PostCreateRequest, verifiedVideo *video.Blob) (map[string]any, error) {
	body := map[string]any{
		"$type":     craftskyPostNSID,
		"text":      req.Text,
		"createdAt": time.Now().UTC().Format(time.RFC3339),
	}
	if len(req.Facets) > 0 {
		body["facets"] = req.Facets
	}
	if len(req.Langs) > 0 {
		body["langs"] = req.Langs
	}
	if req.Reply != nil {
		body["reply"] = map[string]any{
			"root":   map[string]any{"uri": req.Reply.Root.URI, "cid": req.Reply.Root.CID},
			"parent": map[string]any{"uri": req.Reply.Parent.URI, "cid": req.Reply.Parent.CID},
		}
	}
	if req.Embed != nil && req.Embed.Quote != nil {
		body["embed"] = map[string]any{
			"$type": craftskyPostNSID + "#quoteEmbed",
			"record": map[string]any{
				"uri": req.Embed.Quote.URI,
				"cid": req.Embed.Quote.CID,
			},
		}
	}
	if req.Embed != nil && req.Embed.External != nil {
		embed, err := externalEmbedRecord(*req.Embed.External)
		if err != nil {
			return nil, err
		}
		body["embed"] = embed
	}
	if req.Embed != nil && req.Embed.Video != nil {
		if verifiedVideo == nil {
			return nil, errors.New("verified video is required")
		}
		body["embed"] = videoEmbedRecord(*req.Embed.Video, *verifiedVideo)
	}
	if len(req.Images) > 0 {
		images := make([]map[string]any, 0, len(req.Images))
		for _, img := range req.Images {
			one := map[string]any{
				"image": img.Image,
			}
			if strings.TrimSpace(img.Alt) != "" {
				one["alt"] = strings.TrimSpace(img.Alt)
			}
			if img.AspectRatio != nil {
				one["aspectRatio"] = map[string]any{
					"width":  img.AspectRatio.Width,
					"height": img.AspectRatio.Height,
				}
			}
			images = append(images, one)
		}
		body["images"] = images
	}
	if req.Project != nil {
		body["project"] = req.Project
	}
	return body, nil
}

func videoEmbedRecord(request VideoEmbedRequest, verified video.Blob) map[string]any {
	embed := map[string]any{
		"$type": "app.bsky.embed.video",
		"video": map[string]any{
			"$type":    "blob",
			"ref":      map[string]any{"$link": string(verified.CID)},
			"mimeType": verified.MIMEType,
			"size":     verified.Size,
		},
	}
	if request.Alt != "" {
		embed["alt"] = request.Alt
	}
	if request.AspectRatio != nil {
		embed["aspectRatio"] = map[string]any{
			"width":  request.AspectRatio.Width,
			"height": request.AspectRatio.Height,
		}
	}
	return embed
}

func externalEmbedRecord(external ExternalEmbedRequest) (map[string]any, error) {
	return postrecord.ExternalEmbed(
		external.URI, external.Title, external.Description, external.Thumb,
	)
}

// syntheticPostRow assembles the PostRow that BuildPostResponse needs
// from the request body, the PDS-assigned (uri, cid), and a single
// author lookup against the store. We don't wait for the firehose to
// land the row.
//
// Tags are extracted from req.Facets via a non-strict decode into the
// indigo richtext-facet typed slice. Errors on that decode produce
// empty tags (the PDS will still validate the lexicon shape) — we
// don't fail the whole request just because tag extraction couldn't
// parse facets the PDS may yet accept.
func syntheticPostRow(
	r *http.Request,
	store postAuthorReader,
	did syntax.DID,
	uri syntax.ATURI,
	cid syntax.CID,
	req PostCreateRequest,
	verifiedVideo *video.Blob,
) (*PostRow, error) {
	now := time.Now().UTC()
	row := &PostRow{
		URI:       string(uri),
		DID:       did.String(),
		Rkey:      path.Base(string(uri)),
		CID:       string(cid),
		Text:      req.Text,
		Tags:      postutil.MergeTags(extractRequestTags(req.Text, req.Facets), requestProjectTags(req.Project)),
		Langs:     append([]string(nil), req.Langs...),
		CreatedAt: now,
		IndexedAt: now,
		Project:   req.Project,
	}
	if len(req.Facets) > 0 {
		row.Facets = req.Facets
	}
	if req.Reply != nil {
		row.ReplyRootURI = strPtr(req.Reply.Root.URI)
		row.ReplyRootCID = strPtr(req.Reply.Root.CID)
		row.ReplyParentURI = strPtr(req.Reply.Parent.URI)
		row.ReplyParentCID = strPtr(req.Reply.Parent.CID)
	}
	if req.Embed != nil && req.Embed.Quote != nil {
		row.QuoteURI = strPtr(req.Embed.Quote.URI)
		row.QuoteCID = strPtr(req.Embed.Quote.CID)
	}
	if req.Embed != nil && req.Embed.External != nil {
		embed, err := externalEmbedRecord(*req.Embed.External)
		if err != nil {
			return nil, err
		}
		rawEmbed, err := json.Marshal(embed)
		if err != nil {
			return nil, err
		}
		row.RawEmbed = rawEmbed
	}
	if req.Embed != nil && req.Embed.Video != nil && verifiedVideo != nil {
		rawEmbed, err := json.Marshal(videoEmbedRecord(*req.Embed.Video, *verifiedVideo))
		if err != nil {
			return nil, err
		}
		row.RawEmbed = rawEmbed
	}
	if len(req.Images) > 0 {
		imagesJSON, err := syntheticPostImagesJSON(req.Images)
		if err != nil {
			return nil, err
		}
		row.Images = imagesJSON
	}

	author, err := store.ReadAuthor(r.Context(), did.String())
	if err != nil {
		return nil, err
	}
	if author != nil {
		row.AuthorDisplayName = author.DisplayName
		row.AuthorAvatarCID = author.AvatarCID
		row.AuthorAvatarMime = author.AvatarMime
	}
	return row, nil
}

func requestProjectTags(project *Project) []string {
	if project == nil {
		return nil
	}
	return postutil.ExtractProjectTags(
		project.Common.Tags,
		requestPatternFacetedTexts(project.Common.Pattern),
		requestMaterialFacetedTexts(project.Common.Materials),
	)
}

func requestPatternFacetedTexts(pattern *ProjectPattern) []postutil.FacetedText {
	if pattern == nil {
		return nil
	}
	return []postutil.FacetedText{
		{Text: stringPtrValue(pattern.Name), Facets: postutil.DecodeFacets(pattern.NameFacets)},
		{Text: stringPtrValue(pattern.Designer), Facets: postutil.DecodeFacets(pattern.DesignerFacets)},
		{Text: stringPtrValue(pattern.Publisher), Facets: postutil.DecodeFacets(pattern.PublisherFacets)},
	}
}

func requestMaterialFacetedTexts(materials []ProjectMaterial) []postutil.FacetedText {
	out := make([]postutil.FacetedText, 0, len(materials))
	for _, material := range materials {
		out = append(out, postutil.FacetedText{
			Text:   material.Text,
			Facets: postutil.DecodeFacets(material.Facets),
		})
	}
	return out
}

func syntheticPostImagesJSON(images []PostImage) (json.RawMessage, error) {
	out := make([]storedPostImage, 0, len(images))
	for _, img := range images {
		ref, _ := img.Image["ref"].(map[string]any)
		cid, _ := ref["$link"].(string)
		mime, _ := img.Image["mimeType"].(string)
		out = append(out, storedPostImage{
			CID:         cid,
			MIME:        mime,
			Size:        int64FromJSONNumber(img.Image["size"]),
			Alt:         strings.TrimSpace(img.Alt),
			AspectRatio: img.AspectRatio,
		})
	}
	return json.Marshal(out)
}

func int64FromJSONNumber(value any) int64 {
	switch n := value.(type) {
	case float64:
		return int64(n)
	case int:
		return int64(n)
	case int64:
		return n
	case json.Number:
		v, _ := n.Int64()
		return v
	default:
		return 0
	}
}

// extractRequestTags decodes the raw facets JSON into the indigo typed
// slice (best effort) and returns the same tag set the indexer would
// produce when this record arrives via the firehose. Returns an empty
// (non-nil) slice on decode failure so the response always carries a
// valid tags array.
func extractRequestTags(text string, raw json.RawMessage) []string {
	if len(raw) == 0 {
		return []string{}
	}
	typed := postutil.DecodeFacets(raw)
	if len(typed) == 0 {
		return []string{}
	}
	return postutil.ExtractTagsForText(text, typed)
}

func strPtr(s string) *string { return &s }
