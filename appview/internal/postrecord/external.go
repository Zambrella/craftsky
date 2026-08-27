package postrecord

import (
	"encoding/json"

	appbsky "github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"

	"social.craftsky/appview/internal/lexicon/craftsky"
)

func ExternalEmbed(uri, title, description string, thumb map[string]any) (map[string]any, error) {
	var generatedThumb *lexutil.LexBlob
	if thumb != nil {
		body, err := json.Marshal(thumb)
		if err != nil {
			return nil, err
		}
		generatedThumb = new(lexutil.LexBlob)
		if err := json.Unmarshal(body, generatedThumb); err != nil {
			return nil, err
		}
	}
	embed := &craftsky.FeedPost_Embed{EmbedExternal: &appbsky.EmbedExternal{
		External: &appbsky.EmbedExternal_External{
			Uri: uri, Title: title, Description: description, Thumb: generatedThumb,
		},
	}}
	body, err := json.Marshal(embed)
	if err != nil {
		return nil, err
	}
	var wire map[string]any
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, err
	}
	return wire, nil
}
