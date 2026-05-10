// appview/internal/api/interaction_response.go
package api

import "time"

// InteractionWriteResponse is the shared response for successful like and
// repost writes. The subject is the post strongRef the interaction points at.
type InteractionWriteResponse struct {
	URI       string            `json:"uri"`
	CID       string            `json:"cid"`
	Rkey      string            `json:"rkey"`
	Subject   ResponseStrongRef `json:"subject"`
	CreatedAt time.Time         `json:"createdAt"`
}
