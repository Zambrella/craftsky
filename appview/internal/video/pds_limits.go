package video

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

// PDSBlobLimits discovers public capabilities using the hardened PDS JSON client.
// Unknown limits retain Craftsky's ceiling. Cache keys are session-authoritative
// PDS origins, so an account migration cannot reuse its previous host's limit.
type PDSBlobLimits struct {
	client *http.Client
	mu     sync.Mutex
	cache  map[string]pdsLimitEntry
}
type pdsLimitEntry struct {
	limit   int64
	expires time.Time
}

func NewPDSBlobLimits(client *http.Client) *PDSBlobLimits {
	return &PDSBlobLimits{client: client, cache: make(map[string]pdsLimitEntry)}
}

func (p *PDSBlobLimits) Get(ctx context.Context, origin string) int64 {
	origin, err := serviceOrigin(origin)
	if err != nil {
		return maxVideoBlobBytes
	}
	p.mu.Lock()
	entry, ok := p.cache[origin]
	p.mu.Unlock()
	if ok && time.Now().Before(entry.expires) {
		return entry.limit
	}
	limit, known := p.fetch(ctx, origin)
	ttl := 30 * time.Second
	if known {
		ttl = 5 * time.Minute
	}
	p.mu.Lock()
	// Bound memory even when many different federated hosts are encountered.
	if len(p.cache) >= 512 {
		clear(p.cache)
	}
	p.cache[origin] = pdsLimitEntry{limit, time.Now().Add(ttl)}
	p.mu.Unlock()
	return limit
}

func (p *PDSBlobLimits) fetch(ctx context.Context, origin string) (int64, bool) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, origin+"/xrpc/com.atproto.server.describeServer", nil)
	if err != nil || p.client == nil {
		return maxVideoBlobBytes, false
	}
	client := *p.client
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		return maxVideoBlobBytes, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return maxVideoBlobBytes, false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxLimitsResponseBytes+1))
	if err != nil || len(body) > maxLimitsResponseBytes {
		return maxVideoBlobBytes, false
	}
	var output struct {
		BlobUploadLimit int64 `json:"blobUploadLimit"`
	}
	if json.Unmarshal(body, &output) != nil || output.BlobUploadLimit <= 0 {
		return maxVideoBlobBytes, false
	}
	return min(output.BlobUploadLimit, maxVideoBlobBytes), true
}
