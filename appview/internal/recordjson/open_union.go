package recordjson

import (
	"encoding/json"
	"io"

	"github.com/bluesky-social/indigo/atproto/atdata"
)

// OpenUnion retains an unfamiliar variant instead of silently discarding it.
// Parse preserves integer precision and validates nested AT data, including blobs.
type OpenUnion struct{ Raw json.RawMessage }

func (u *OpenUnion) UnmarshalJSON(raw []byte) error {
	if _, err := Parse(raw); err != nil {
		return err
	}
	u.Raw = append(u.Raw[:0], raw...)
	return nil
}
func (u OpenUnion) MarshalJSON() ([]byte, error) { return u.Raw, nil }
func (u OpenUnion) MarshalCBOR(w io.Writer) error {
	value, err := Parse(u.Raw)
	if err != nil {
		return err
	}
	raw, err := atdata.MarshalCBOR(value)
	if err != nil {
		return err
	}
	_, err = w.Write(raw)
	return err
}
func (u *OpenUnion) UnmarshalCBOR(r io.Reader) error {
	raw, err := io.ReadAll(io.LimitReader(r, atdata.MAX_CBOR_RECORD_SIZE+1))
	if err != nil {
		return err
	}
	value, err := atdata.UnmarshalCBOR(raw)
	if err != nil {
		return err
	}
	u.Raw, err = json.Marshal(value)
	return err
}
