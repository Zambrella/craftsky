package api

// WhoAmIResponse is the 200 body for GET /v1/whoami.
type WhoAmIResponse struct {
	DID    string `json:"did"`
	Handle string `json:"handle"`
}
