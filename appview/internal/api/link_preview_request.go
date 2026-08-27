package api

import (
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"
)

const maxLinkPreviewURLBytes = 8_192

type LinkPreviewRequest struct {
	URL string `json:"url"`
}

func DecodeLinkPreviewRequest(body io.Reader) (LinkPreviewRequest, error) {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	var request LinkPreviewRequest
	if err := decoder.Decode(&request); err != nil {
		return LinkPreviewRequest{}, invalidLinkPreviewRequest(err.Error())
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return LinkPreviewRequest{}, invalidLinkPreviewRequest(err.Error())
	}
	if request.URL == "" || !utf8.ValidString(request.URL) || len(request.URL) > maxLinkPreviewURLBytes {
		return LinkPreviewRequest{}, invalidLinkPreviewRequest("url must contain at most 8192 UTF-8 bytes")
	}
	return request, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return errors.New("request must contain exactly one JSON object")
}

func invalidLinkPreviewRequest(message string) error {
	return &FieldError{Code: "invalid_request", Fields: map[string]string{"_": message}}
}
