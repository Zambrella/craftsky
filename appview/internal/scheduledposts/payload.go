package scheduledposts

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

type Payload struct {
	Kind    PostKind        `json:"kind"`
	Text    string          `json:"text"`
	Facets  json.RawMessage `json:"facets,omitempty"`
	Langs   []string        `json:"langs,omitempty"`
	Project json.RawMessage `json:"project,omitempty"`
	Media   []PayloadMedia  `json:"media,omitempty"`
}

type PayloadMedia struct {
	ID     string `json:"id"`
	Alt    string `json:"alt"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

func EncodePayload(payload Payload) ([]byte, error) {
	return json.Marshal(payload)
}

func DecodePayload(encoded []byte) (Payload, error) {
	var payload Payload
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return Payload{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return Payload{}, errors.New("scheduled payload contains multiple values")
		}
		return Payload{}, err
	}
	return payload, nil
}
