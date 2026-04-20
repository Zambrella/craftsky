package testpipeline

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler serves GET /test/feed. See doc.go — disposable.
type Handler struct {
	pool *pgxpool.Pool
}

// NewHandler returns an HTTP handler backed by pool.
func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{pool: pool} }

const (
	defaultLimit = 50
	maxLimit     = 200
)

type feedPost struct {
	URI       string    `json:"uri"`
	CID       string    `json:"cid"`
	DID       string    `json:"did"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
	IndexedAt time.Time `json:"indexedAt"`
}

type feedResponse struct {
	Posts []feedPost `json:"posts"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	limit := defaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		if n > maxLimit {
			n = maxLimit
		}
		limit = n
	}

	rows, err := h.pool.Query(r.Context(),
		`SELECT uri, cid, did, text, created_at, indexed_at
		   FROM test_posts
		  ORDER BY created_at DESC
		  LIMIT $1`,
		limit,
	)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Non-nil zero-length slice so it serialises as [] not null.
	posts := make([]feedPost, 0)
	for rows.Next() {
		var p feedPost
		if err := rows.Scan(&p.URI, &p.CID, &p.DID, &p.Text, &p.CreatedAt, &p.IndexedAt); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(feedResponse{Posts: posts})
}
