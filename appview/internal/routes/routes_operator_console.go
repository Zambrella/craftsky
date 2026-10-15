package routes

import (
	"net/http"
	"os"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/operatorconsole"
)

func registerOperatorConsole(mux Registrar, mw v1Middleware, deps *Dependencies) {
	if !deps.Config.ModerationAdminEnabled {
		return
	}
	unavailable := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		envelope.WriteError(w, 503, "operator_auth_unavailable", "Operator sign-in is unavailable.", ctxkeys.GetRunID(r.Context()), nil)
	})
	var bootstrap, login, logout, callback, metadata, jwks http.Handler = unavailable, unavailable, unavailable, unavailable, unavailable, unavailable
	if h := deps.OperatorConsole; h != nil {
		bootstrap, login, logout, callback, metadata, jwks = h.Bootstrap(), h.Login(), h.Logout(), h.Callback(), h.Metadata(), h.JWKS()
	}
	mux.Handle("GET /v1/admin/auth/csrf", mw.wrap(mustConfiguredPolicy(deps.Config.Env, deps.Config, "GET", "/v1/admin/auth/csrf"), bootstrap))
	mux.Handle("POST /v1/admin/auth/login", mw.wrap(mustConfiguredPolicy(deps.Config.Env, deps.Config, "POST", "/v1/admin/auth/login"), login))
	mux.Handle("POST /v1/admin/auth/logout", mw.wrap(mustConfiguredPolicy(deps.Config.Env, deps.Config, "POST", "/v1/admin/auth/logout"), logout))
	mux.Handle("GET /oauth/admin/callback", callback)
	mux.Handle("GET /oauth/admin/client-metadata.json", metadata)
	mux.Handle("GET /oauth/admin/jwks.json", jwks)
	mux.Handle("GET /admin", http.RedirectHandler("/admin/", http.StatusTemporaryRedirect))
	mux.Handle("GET /admin/", operatorconsole.Assets(os.DirFS("admin/dist")))
}
