package app

import (
	"net/http"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/linkpreview"
)

func newLinkPreviewDependencies() api.LinkPreviewService {
	transport := linkpreview.NewPinnedTransport(nil, nil)
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	return linkpreview.NewService(linkpreview.NewFetcher(nil, client))
}
