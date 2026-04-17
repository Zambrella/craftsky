// Package api holds HTTP handler factories. Each handler factory takes
// only the specific dependencies it needs — never the full *app.Deps —
// so handlers can't silently grow dependencies over time.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthHandler returns a handler that pings the DB pool and reports
// 200 on success or 503 on failure. The ping is given a 2-second
// per-request timeout so a hung DB doesn't hang health checks.
//
// The response contract:
//   - 200 + application/json + {"status":"ok"}
//   - 503 + text/plain + "db unreachable"
//
// The underlying error is logged at Error via logger but not returned
// to the client.
func HealthHandler(pool *pgxpool.Pool, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			logger.Error("health: db ping failed", slog.String("err", err.Error()))
			// http.Error sets Content-Type: text/plain; charset=utf-8 itself.
			http.Error(w, "db unreachable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
}
