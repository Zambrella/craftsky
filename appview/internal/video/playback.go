package video

import (
	"errors"
	"net/url"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

const (
	defaultPlaylistTemplate  = "https://video.bsky.app/watch/{did}/{cid}/playlist.m3u8"
	defaultThumbnailTemplate = "https://video.bsky.app/watch/{did}/{cid}/thumbnail.jpg"
)

var ErrPlaybackConfig = errors.New("invalid video playback configuration")

type PlaybackConfig struct {
	PlaylistTemplate  string
	ThumbnailTemplate string
}

type PlaybackURLBuilder struct {
	playlistTemplate  string
	thumbnailTemplate string
}

func NewPlaybackURLBuilder(config PlaybackConfig) (*PlaybackURLBuilder, error) {
	if config.PlaylistTemplate == "" {
		config.PlaylistTemplate = defaultPlaylistTemplate
	}
	if config.ThumbnailTemplate == "" {
		config.ThumbnailTemplate = defaultThumbnailTemplate
	}
	if !validPlaybackTemplate(config.PlaylistTemplate) || !validPlaybackTemplate(config.ThumbnailTemplate) {
		return nil, ErrPlaybackConfig
	}
	return &PlaybackURLBuilder{playlistTemplate: config.PlaylistTemplate, thumbnailTemplate: config.ThumbnailTemplate}, nil
}

func (builder *PlaybackURLBuilder) URLs(did syntax.DID, cid syntax.CID) (string, string) {
	if builder == nil || did == "" || cid == "" {
		return "", ""
	}
	replacer := strings.NewReplacer("{did}", url.PathEscape(did.String()), "{cid}", url.PathEscape(cid.String()))
	return replacer.Replace(builder.playlistTemplate), replacer.Replace(builder.thumbnailTemplate)
}

func validPlaybackTemplate(template string) bool {
	if strings.Count(template, "{did}") != 1 || strings.Count(template, "{cid}") != 1 {
		return false
	}
	parsed, err := url.Parse(template)
	return err == nil && parsed.Scheme == "https" && parsed.Hostname() != "" && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == ""
}
