package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/scheduledposts"
)

type scheduledPostCreateRequest struct {
	OperationID string                      `json:"operationId"`
	ScheduledAt time.Time                   `json:"scheduledAt"`
	Payload     scheduledPostPayloadRequest `json:"payload"`
}

type scheduledPostUpdateRequest struct {
	ScheduledAt time.Time                   `json:"scheduledAt"`
	Payload     scheduledPostPayloadRequest `json:"payload"`
}

type scheduledPostPublicationRequest struct {
	Payload scheduledPostPayloadRequest `json:"payload"`
}

type scheduledPostPayloadRequest struct {
	Kind     scheduledposts.PostKind         `json:"kind"`
	Text     string                          `json:"text"`
	Facets   json.RawMessage                 `json:"facets,omitempty"`
	Langs    []string                        `json:"langs,omitempty"`
	Project  json.RawMessage                 `json:"project,omitempty"`
	Media    []scheduledposts.PayloadMedia   `json:"media,omitempty"`
	Reply    json.RawMessage                 `json:"reply,omitempty"`
	Embed    json.RawMessage                 `json:"embed,omitempty"`
	External *scheduledposts.PayloadExternal `json:"external,omitempty"`
}

func (payload scheduledPostPayloadRequest) canonical() scheduledposts.Payload {
	return scheduledposts.Payload{
		Kind: payload.Kind, Text: payload.Text, Facets: payload.Facets,
		Langs: payload.Langs, Project: payload.Project, Media: payload.Media,
		External: payload.External,
	}
}

func decodeScheduledPostCreate(body io.Reader) (scheduledPostCreateRequest, error) {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	var request scheduledPostCreateRequest
	if err := decoder.Decode(&request); err != nil {
		return scheduledPostCreateRequest{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return scheduledPostCreateRequest{}, errors.New("request contains multiple values")
		}
		return scheduledPostCreateRequest{}, err
	}
	return request, nil
}

func decodeScheduledPostUpdate(body io.Reader) (scheduledPostUpdateRequest, error) {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	var request scheduledPostUpdateRequest
	if err := decoder.Decode(&request); err != nil {
		return scheduledPostUpdateRequest{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return scheduledPostUpdateRequest{}, errors.New("request contains multiple values")
		}
		return scheduledPostUpdateRequest{}, err
	}
	return request, nil
}

func decodeScheduledPostPublication(body io.Reader) (scheduledPostPublicationRequest, error) {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	var request scheduledPostPublicationRequest
	if err := decoder.Decode(&request); err != nil {
		return scheduledPostPublicationRequest{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return scheduledPostPublicationRequest{}, errors.New("request contains multiple values")
		}
		return scheduledPostPublicationRequest{}, err
	}
	return request, nil
}

func validateScheduledPostRequest(
	now time.Time,
	request scheduledPostCreateRequest,
	limits MediaLimits,
) (uuid.UUID, []uuid.UUID, []byte, error) {
	operationID, err := uuid.Parse(request.OperationID)
	if err != nil {
		return uuid.Nil, nil, nil, &FieldError{Code: "validation_failed", Fields: map[string]string{
			"operationId": "must be a UUID",
		}}
	}
	if err := scheduledposts.ValidateScheduledAt(now, request.ScheduledAt); err != nil {
		return uuid.Nil, nil, nil, err
	}
	if err := scheduledposts.ValidateScheduleEligibility(scheduledposts.PostShape{
		Kind:              request.Payload.Kind,
		HasReplyReference: hasScheduledJSONValue(request.Payload.Reply),
		HasQuoteEmbed:     hasScheduledJSONValue(request.Payload.Embed),
		HasExternal:       request.Payload.External != nil,
	}); err != nil {
		return uuid.Nil, nil, nil, err
	}

	postRequest := PostCreateRequest{
		Text:   request.Payload.Text,
		Facets: request.Payload.Facets,
		Langs:  request.Payload.Langs,
	}
	if len(request.Payload.Project) > 0 {
		if err := json.Unmarshal(request.Payload.Project, &postRequest.Project); err != nil {
			return uuid.Nil, nil, nil, &FieldError{Code: "validation_failed", Fields: map[string]string{
				"payload.project": "must be a valid project",
			}}
		}
	}
	if request.Payload.Kind == scheduledposts.PostKindProject && postRequest.Project == nil {
		return uuid.Nil, nil, nil, scheduledposts.ErrIneligibleScheduledPost
	}
	if request.Payload.Kind == scheduledposts.PostKindStandard && postRequest.Project != nil {
		return uuid.Nil, nil, nil, scheduledposts.ErrIneligibleScheduledPost
	}

	mediaIDs := make([]uuid.UUID, 0, len(request.Payload.Media))
	for index, media := range request.Payload.Media {
		mediaID, err := uuid.Parse(media.ID)
		if err != nil {
			return uuid.Nil, nil, nil, &FieldError{Code: "validation_failed", Fields: map[string]string{
				"payload.media": "contains an invalid media ID",
			}}
		}
		mediaIDs = append(mediaIDs, mediaID)
		postImage := PostImage{
			Image: map[string]any{
				"ref":      map[string]any{"$link": "bafk-scheduled-placeholder"},
				"mimeType": "image/jpeg",
				"size":     1,
			},
			Alt: media.Alt,
		}
		if media.Width != 0 || media.Height != 0 {
			postImage.AspectRatio = &PostImageAspectRatio{Width: media.Width, Height: media.Height}
		}
		postRequest.Images = append(postRequest.Images, postImage)
		_ = index
	}
	if external := request.Payload.External; external != nil {
		sourceURI, err := canonicalScheduledExternalSourceURI(external.SourceURI)
		if err != nil {
			return uuid.Nil, nil, nil, &FieldError{Code: "validation_failed", Fields: map[string]string{
				"payload.external.sourceUri": "must be a normalized fragmentless HTTP or HTTPS URL",
			}}
		}
		external.SourceURI = sourceURI
		postRequest.Embed = &EmbedRequest{External: &ExternalEmbedRequest{
			URI: external.URI, Title: external.Title, Description: external.Description,
		}}
		if external.ThumbMediaID != "" {
			mediaID, err := uuid.Parse(external.ThumbMediaID)
			if err != nil {
				return uuid.Nil, nil, nil, &FieldError{Code: "validation_failed", Fields: map[string]string{
					"payload.external.thumbMediaId": "must be a valid media ID",
				}}
			}
			mediaIDs = append(mediaIDs, mediaID)
			postRequest.Embed.External.Thumb = map[string]any{
				"$type":    "blob",
				"ref":      map[string]any{"$link": "bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"},
				"mimeType": "image/jpeg", "size": 1,
			}
		}
	}
	if err := ValidatePostCreateWithLimits(postRequest, limits); err != nil {
		return uuid.Nil, nil, nil, err
	}
	payloadBytes, err := scheduledposts.EncodePayload(request.Payload.canonical())
	if err != nil {
		return uuid.Nil, nil, nil, err
	}
	// Copying through a decoder above guarantees deterministic field ordering
	// when this canonical payload is encoded for persistence and hashing.
	return operationID, mediaIDs, bytes.Clone(payloadBytes), nil
}

func canonicalScheduledExternalSourceURI(source string) (string, error) {
	if source == "" || source != strings.TrimSpace(source) || len(source) > maxLinkPreviewURLBytes || strings.Contains(source, "#") {
		return "", errors.New("invalid source URI")
	}
	parsed, err := url.Parse(source)
	if err != nil || parsed.Opaque != "" || parsed.User != nil || parsed.Hostname() == "" {
		return "", errors.New("invalid source URI")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", errors.New("invalid source URI")
	}
	hostname := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if scheme == "http" && port == "80" || scheme == "https" && port == "443" {
		port = ""
	}
	host := hostname
	if port != "" {
		host = net.JoinHostPort(hostname, port)
	} else if strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}
	return (&url.URL{
		Scheme: scheme, Host: host, Path: parsed.Path, RawPath: parsed.RawPath,
		RawQuery: parsed.RawQuery, ForceQuery: parsed.ForceQuery,
	}).String(), nil
}

func hasScheduledJSONValue(value json.RawMessage) bool {
	trimmed := bytes.TrimSpace(value)
	return len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null"))
}
