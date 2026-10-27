// Package profilelinks implements the shared Bluesky profile-link record.
package profilelinks

import (
	_ "embed"
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/bluesky-social/indigo/atproto/atdata"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/lex/util"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"github.com/rivo/uniseg"
	"social.craftsky/appview/internal/recordjson"
)

const Collection syntax.NSID = "app.bsky.actor.link"
const MaxLinks = 10
const EditorTitleGraphemes = 30

type StrongRef struct {
	URI syntax.ATURI `json:"uri"`
	CID syntax.CID   `json:"cid"`
}

// Record mirrors the published app.bsky.actor.link schema; unfamiliar fields
// are preserved by the write path's authoritative JSON merge.
type Record struct {
	Type      string          `json:"$type"`
	URL       string          `json:"url"`
	Title     string          `json:"title,omitempty"`
	Icon      *util.LexBlob   `json:"icon,omitempty"`
	Labels    json.RawMessage `json:"labels,omitempty"`
	CreatedAt string          `json:"createdAt"`
}

type View struct {
	URI   syntax.ATURI `json:"uri"`
	CID   syntax.CID   `json:"cid"`
	URL   string       `json:"url"`
	Title string       `json:"title,omitempty"`
	Icon  string       `json:"icon,omitempty"`
	// Original blob and timestamp let the owner edit without reconstructing blobs
	// from CDN URLs. They are public record data, only included in the owner's view.
	IconBlob  *util.LexBlob `json:"iconBlob,omitempty"`
	CreatedAt string        `json:"createdAt,omitempty"`
}

//go:embed policy.json
var policyJSON []byte

type SupportProvider struct {
	Name       string `json:"name"`
	Domain     string `json:"domain"`
	PathPrefix string `json:"pathPrefix"`
}

var Policy = func() struct {
	Providers  []SupportProvider `json:"providers"`
	Shorteners []string          `json:"shorteners"`
} {
	var p struct {
		Providers  []SupportProvider `json:"providers"`
		Shorteners []string          `json:"shorteners"`
	}
	if err := json.Unmarshal(policyJSON, &p); err != nil {
		panic(err)
	}
	return p
}()

func hostMatches(host, domain string) bool {
	return host == domain || strings.HasSuffix(host, "."+domain)
}

// Normalize applies the editor's HTTPS-only and URL-shortener policy. This is
// intentionally stricter than the lexicon's generic URI syntax.
func Normalize(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || strings.ContainsAny(raw, "\r\n\t ") {
		return "", errors.New("profile link must be an HTTPS URL")
	}
	u.Host = strings.ToLower(u.Host)
	for _, domain := range Policy.Shorteners {
		if hostMatches(strings.ToLower(u.Hostname()), domain) {
			return "", errors.New("URL shorteners are not supported")
		}
	}
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String(), nil
}

func Support(raw string) *SupportProvider {
	u, err := url.Parse(raw)
	if err != nil {
		return nil
	}
	host := strings.ToLower(u.Hostname())
	path := strings.TrimRight(strings.ToLower(u.Path), "/")
	for _, p := range Policy.Providers {
		if !hostMatches(host, p.Domain) {
			continue
		}
		if p.PathPrefix != "" {
			if strings.HasPrefix(path, strings.TrimRight(p.PathPrefix, "/")+"/") {
				return &p
			}
		} else if (host != p.Domain && host != "www."+p.Domain) || path != "" {
			return &p
		}
	}
	return nil
}

func Title(raw string, limit int) string {
	g := uniseg.NewGraphemes(strings.TrimSpace(raw))
	var out strings.Builder
	for i := 0; i < limit && g.Next(); i++ {
		out.WriteString(g.Str())
	}
	return strings.TrimSpace(out.String())
}

func OwnRef(owner syntax.DID, uri syntax.ATURI) bool {
	return uri.Authority().String() == owner.String() && uri.Collection() == Collection && uri.RecordKey() != ""
}

// RecordCID uses the protocol's DAG-CBOR representation, including typed blob
// and CID links, rather than hashing JSON bytes.
func RecordCID(raw json.RawMessage) (syntax.CID, error) {
	value, err := recordjson.Parse(raw)
	if err != nil {
		return "", err
	}
	encoded, err := atdata.MarshalCBOR(value)
	if err != nil {
		return "", err
	}
	valueCID, err := (cid.Prefix{Version: 1, Codec: cid.DagCBOR, MhType: multihash.SHA2_256, MhLength: 32}).Sum(encoded)
	return syntax.CID(valueCID.String()), err
}
