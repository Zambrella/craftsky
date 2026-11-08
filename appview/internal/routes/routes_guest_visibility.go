package routes

import (
	"net/http"
	"social.craftsky/appview/internal/api"
)

func registerGuestVisibilityRoutes(mux Registrar, middleware v1Middleware, settings api.GuestVisibilitySettings) {
	const path = "/v1/preferences/logged-out-visibility"
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		mux.Handle(method+" "+path, middleware.wrap(mustPolicy(method, path), api.LoggedOutVisibilityHandler(settings)))
	}
}
