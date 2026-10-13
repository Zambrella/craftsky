// Package recordjson decodes unmodified repository JSON without rounding integers.
package recordjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/bluesky-social/indigo/atproto/atdata"
)

func Unmarshal(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("record contains multiple JSON values")
		}
		return err
	}
	return nil
}

// Parse returns generic AT data for schema validation. The original JSON should
// still be retained for publication and storage, including unfamiliar fields.
func Parse(raw []byte) (map[string]any, error) {
	if len(raw) > atdata.MAX_JSON_RECORD_SIZE {
		return nil, errors.New("record exceeds JSON data limit")
	}
	var value map[string]any
	if err := Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	if value == nil {
		return nil, errors.New("record must be an object")
	}
	normalized, err := parse(value, 0, false)
	if err != nil {
		return nil, err
	}
	if err := atdata.Validate(normalized.(map[string]any)); err != nil {
		return nil, err
	}
	parsed, err := parse(normalized, 0, true)
	if err != nil {
		return nil, err
	}
	object, ok := parsed.(map[string]any)
	if !ok {
		return nil, errors.New("record must be an ordinary object")
	}

	return object, nil
}

func parse(value any, depth int, compounds bool) (any, error) {
	if depth > 128 {
		return nil, errors.New("record nesting exceeds data limit")
	}
	switch value := value.(type) {
	case json.Number:
		integer, err := value.Int64()
		if err != nil {
			return nil, fmt.Errorf("record number must be a signed integer: %w", err)
		}
		return integer, nil
	case []any:
		for index, item := range value {
			parsed, err := parse(item, depth+1, compounds)
			if err != nil {
				return nil, err
			}
			value[index] = parsed
		}
	case map[string]any:
		// Use the protocol library for compound data representations, whose
		// metadata is bounded independently of arbitrary extension integers.
		var compound any
		switch {
		case value["$link"] != nil:
			compound = new(atdata.CIDLink)
		case value["$bytes"] != nil:
			compound = new(atdata.Bytes)
		case value["$type"] == "blob":
			compound = new(atdata.Blob)
		}
		if compounds && compound != nil {
			raw, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(raw, compound); err != nil {
				return nil, err
			}
			switch compound := compound.(type) {
			case *atdata.CIDLink:
				return *compound, nil
			case *atdata.Bytes:
				return *compound, nil
			case *atdata.Blob:
				return *compound, nil
			}
		}
		for key, item := range value {
			parsed, err := parse(item, depth+1, compounds)
			if err != nil {
				return nil, err
			}
			value[key] = parsed
		}
	}
	return value, nil
}
