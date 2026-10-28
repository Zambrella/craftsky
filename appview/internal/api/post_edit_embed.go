package api

import (
	"bytes"
	"encoding/json"
	"errors"
)

// mergePostEditEmbed permits supported media changes but never changes a
// quote's structural reference or an unfamiliar embed union.
func mergePostEditEmbed(base, changes json.RawMessage) (json.RawMessage, error) {
	var original, patch struct {
		Type string `json:"$type"`
	}
	if len(base) > 0 && json.Unmarshal(base, &original) != nil {
		return nil, errors.New("invalid current embed")
	}
	if original.Type != "" && original.Type != "app.bsky.embed.external" && original.Type != "app.bsky.embed.video" {
		return nil, errors.New("fixed or unsupported embed cannot be changed")
	}
	if bytes.Equal(bytes.TrimSpace(changes), []byte("null")) {
		return nil, nil
	}
	if err := json.Unmarshal(changes, &patch); err != nil {
		return nil, err
	}
	kind := original.Type
	if patch.Type != "" {
		kind = patch.Type
	}
	if kind != original.Type {
		base = nil
	}
	switch kind {
	case "app.bsky.embed.external":
		return mergeSupportedPostEditObject(base, changes, []string{"$type", "external"}, map[string]postEditObjectMerger{
			"external": func(base, changes json.RawMessage) (json.RawMessage, error) {
				return mergeSupportedPostEditObject(base, changes, []string{"uri", "title", "description", "thumb"}, nil)
			},
		})
	case "app.bsky.embed.video":
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(changes, &fields); err != nil {
			return nil, err
		}
		delete(fields, "jobId") // Completion evidence is editor-only, never public.
		publicChanges, err := json.Marshal(fields)
		if err != nil {
			return nil, err
		}
		return mergeSupportedPostEditObject(base, publicChanges, []string{"$type", "video", "alt", "aspectRatio"}, map[string]postEditObjectMerger{"aspectRatio": mergePostEditAspectRatio})
	default:
		return nil, errors.New("unsupported embed edit")
	}
}
