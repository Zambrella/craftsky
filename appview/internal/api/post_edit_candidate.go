package api

import (
	"encoding/json"
	"errors"
	"strings"
)

// Creation's supported-craft inventory cannot invalidate an untouched open-enum
// value in an existing record. New unsupported values and duplicates still fail.
func validatePostEditCandidate(candidate PostCreateRequest, prior json.RawMessage, limits MediaLimits) error {
	err := validatePostCreateWithLimits(candidate, limits, false)
	var fields *FieldError
	if err == nil || candidate.Project == nil || !errors.As(err, &fields) {
		return err
	}
	original, decodeErr := postEditCandidate(prior)
	if decodeErr != nil || original.Project == nil {
		return err
	}
	common, previous := candidate.Project.Common, original.Project.Common
	if common.CraftType == previous.CraftType && strings.TrimSpace(common.CraftType) != "" {
		delete(fields.Fields, "project.common.craftType")
	}
	retained := map[string]bool{}
	for _, craft := range previous.AdditionalCrafts {
		retained[craft.CraftType] = true
	}
	seen := map[string]bool{common.CraftType: true}
	valid := true
	for _, craft := range common.AdditionalCrafts {
		if strings.TrimSpace(craft.CraftType) == "" || seen[craft.CraftType] || (!IsSupportedProjectCraftType(craft.CraftType) && !retained[craft.CraftType]) {
			valid = false
		}
		seen[craft.CraftType] = true
	}
	if valid {
		delete(fields.Fields, "project.common.additionalCrafts")
	}
	if len(fields.Fields) == 0 {
		return nil
	}
	return err
}

// postEditCandidate maps a public record's embed representation into the
// existing validation model. Public video records contain a blob, not a job.
func postEditCandidate(raw json.RawMessage) (PostCreateRequest, error) {
	var candidate PostCreateRequest
	// Decode embed separately: create requests use embed.video.blob, whereas
	// the Lexicon record uses embed.video as the blob itself.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return candidate, err
	}
	embed := fields["embed"]
	delete(fields, "embed")
	withoutEmbed, err := json.Marshal(fields)
	if err != nil {
		return candidate, err
	}
	if err := json.Unmarshal(withoutEmbed, &candidate); err != nil {
		return candidate, err
	}
	if len(embed) == 0 || string(embed) == "null" {
		return candidate, nil
	}
	var value struct {
		Type        string                `json:"$type"`
		Video       VideoBlob             `json:"video"`
		Alt         string                `json:"alt"`
		AspectRatio *PostImageAspectRatio `json:"aspectRatio"`
		External    *ExternalEmbedRequest `json:"external"`
		Record      *StrongRef            `json:"record"`
	}
	if err := json.Unmarshal(embed, &value); err != nil {
		return candidate, err
	}
	switch value.Type {
	case "app.bsky.embed.video":
		candidate.Embed = &EmbedRequest{Video: &VideoEmbedRequest{Blob: value.Video, Alt: value.Alt, AspectRatio: value.AspectRatio}}
	case "app.bsky.embed.external":
		candidate.Embed = &EmbedRequest{External: value.External}
	case "social.craftsky.feed.post#quoteEmbed":
		candidate.Embed = &EmbedRequest{Quote: value.Record}
	}
	return candidate, nil
}
