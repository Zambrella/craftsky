package linkpreview

import (
	"net/url"
	"regexp"
	"strings"
)

var youtubeVideoID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

type youtubeReference struct {
	videoID string
	target  *url.URL
}

func parseYouTubeURL(raw string) (youtubeReference, bool) {
	target, err := url.Parse(raw)
	if err != nil || target.Scheme != "http" && target.Scheme != "https" {
		return youtubeReference{}, false
	}

	var videoID string
	segments := nonEmptyPathSegments(target.Path)
	switch strings.ToLower(target.Hostname()) {
	case "youtu.be":
		if len(segments) > 0 {
			videoID = segments[0]
		}
	case "youtube.com", "www.youtube.com", "m.youtube.com", "music.youtube.com":
		switch {
		case len(segments) >= 2 && (segments[0] == "shorts" || segments[0] == "live"):
			videoID = segments[1]
		case len(segments) == 0 || segments[0] == "watch":
			videoID = target.Query().Get("v")
		}
	}
	if !youtubeVideoID.MatchString(videoID) {
		return youtubeReference{}, false
	}
	targetCopy := *target
	return youtubeReference{videoID: videoID, target: &targetCopy}, true
}

func (reference youtubeReference) canonicalURL() *url.URL {
	return &url.URL{
		Scheme:   "https",
		Host:     "www.youtube.com",
		Path:     "/watch",
		RawQuery: url.Values{"v": {reference.videoID}}.Encode(),
	}
}

func (reference youtubeReference) oEmbedURL() string {
	return (&url.URL{
		Scheme: "https",
		Host:   "www.youtube.com",
		Path:   "/oembed",
		RawQuery: url.Values{
			"format": {"json"},
			"url":    {reference.canonicalURL().String()},
		}.Encode(),
	}).String()
}

func nonEmptyPathSegments(path string) []string {
	raw := strings.Split(strings.Trim(path, "/"), "/")
	segments := make([]string, 0, len(raw))
	for _, segment := range raw {
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	return segments
}
