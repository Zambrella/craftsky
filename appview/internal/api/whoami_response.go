package api

import "github.com/bluesky-social/indigo/atproto/syntax"

// WhoAmIResponse is the 200 body for GET /v1/whoami.
//
// syntax.DID and syntax.Handle JSON-marshal via their TextMarshaler
// implementations, so the wire shape is unchanged from a string-typed
// version: each field becomes a plain JSON string.
type WhoAmIResponse struct {
	DID                  syntax.DID    `json:"did"`
	Handle               syntax.Handle `json:"handle"`
	PDSURL               string        `json:"pdsUrl,omitempty"`
	HostingProvider      string        `json:"hostingProvider,omitempty"`
	AccountManagementURL string        `json:"accountManagementUrl,omitempty"`
	PasswordChangeURL    string        `json:"passwordChangeUrl,omitempty"`
}
