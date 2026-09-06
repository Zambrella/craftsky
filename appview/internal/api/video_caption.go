package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

type VideoCaptionPostReader interface {
	ReadOne(context.Context, string, string) (*PostRow, error)
}

type VideoCaptionBlobFetcher interface {
	Fetch(context.Context, syntax.DID, syntax.CID) ([]byte, error)
}

func VideoCaptionHandler(store VideoCaptionPostReader, fetcher VideoCaptionBlobFetcher, logger *slog.Logger, observers ...VideoOperationObserver) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		result, reason := "success", "none"
		defer func() {
			if len(observers) > 0 && observers[0] != nil {
				observers[0].ObserveVideoOperation(r.Context(), "caption", result, reason, time.Since(started))
			}
		}()
		runID := middleware.GetRunID(r.Context())
		did, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			result, reason = "rejected", "invalid_request"
			envelope.WriteError(w, http.StatusBadRequest, "invalid_post", "invalid post identifier", runID, nil)
			return
		}
		rkey, err := syntax.ParseRecordKey(r.PathValue("rkey"))
		if err != nil {
			result, reason = "rejected", "invalid_request"
			envelope.WriteError(w, http.StatusBadRequest, "invalid_post", "invalid post identifier", runID, nil)
			return
		}
		captionCID, err := syntax.ParseCID(r.PathValue("captionCid"))
		if err != nil {
			result, reason = "rejected", "invalid_request"
			envelope.WriteError(w, http.StatusBadRequest, "invalid_caption", "invalid caption identifier", runID, nil)
			return
		}
		if store == nil || fetcher == nil {
			result, reason = "unavailable", "upstream"
			envelope.WriteError(w, http.StatusServiceUnavailable, "video_captions_unavailable", "video captions unavailable", runID, nil)
			return
		}
		row, err := store.ReadOne(r.Context(), did.String(), rkey.String())
		if err != nil || !indexedVideoHasCaption(row, captionCID) {
			result, reason = "rejected", "not_found"
			envelope.WriteError(w, http.StatusNotFound, "caption_not_found", "caption not found", runID, nil)
			return
		}
		body, err := fetcher.Fetch(r.Context(), did, captionCID)
		if err != nil || len(body) == 0 || len(body) > 20_000 || !validWebVTT(body) {
			result, reason = "unavailable", "invalid_content"
			logger.Warn("video caption fetch failed", slog.String("result", "unavailable"))
			envelope.WriteError(w, http.StatusBadGateway, "video_caption_fetch_failed", "video caption unavailable", runID, nil)
			return
		}
		w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
		w.Header().Set("Cache-Control", "private, max-age=300")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
}

func indexedVideoHasCaption(row *PostRow, wanted syntax.CID) bool {
	if row == nil || len(row.RawEmbed) == 0 {
		return false
	}
	var embed struct {
		Type     string `json:"$type"`
		Captions []struct {
			File struct {
				Ref struct {
					Link string `json:"$link"`
				} `json:"ref"`
				MIMEType string `json:"mimeType"`
				Size     int64  `json:"size"`
			} `json:"file"`
		} `json:"captions"`
	}
	if json.Unmarshal(row.RawEmbed, &embed) != nil || embed.Type != "app.bsky.embed.video" {
		return false
	}
	for _, caption := range embed.Captions {
		if caption.File.Ref.Link == wanted.String() && caption.File.MIMEType == "text/vtt" && caption.File.Size > 0 && caption.File.Size <= 20_000 {
			return true
		}
	}
	return false
}

func validWebVTT(body []byte) bool {
	body = bytes.TrimPrefix(body, []byte{0xef, 0xbb, 0xbf})
	return bytes.HasPrefix(body, []byte("WEBVTT")) && (len(body) == 6 || body[6] == '\n' || body[6] == '\r' || body[6] == ' ' || body[6] == '\t')
}
