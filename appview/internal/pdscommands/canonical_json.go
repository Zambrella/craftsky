package pdscommands

import (
	"bytes"
	"encoding/json"
	"errors"
)

func canonicalJSON(value any) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var canonical any
	if err := decoder.Decode(&canonical); err != nil {
		return nil, err
	}
	if decoder.More() {
		return nil, errors.New("canonical JSON contains trailing values")
	}
	return json.Marshal(canonical)
}
