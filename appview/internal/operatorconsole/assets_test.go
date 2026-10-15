package operatorconsole_test

import (
	"net/http/httptest"
	"social.craftsky/appview/internal/operatorconsole"
	"strings"
	"testing"
	"testing/fstest"
)

func TestConsoleAssetsAreScopedAndNeverServeHTMLForMissingScripts(t *testing.T) {
	h := operatorconsole.Assets(fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<main>CraftSky moderation</main>")}, "assets/app.js": &fstest.MapFile{Data: []byte("console.log('shell')")}})
	for _, test := range []struct {
		path   string
		status int
		body   string
	}{{"/admin/", 200, "CraftSky moderation"}, {"/admin/assets/app.js", 200, "shell"}, {"/admin/assets/missing.js", 404, ""}, {"/v1/posts", 404, ""}} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", test.path, nil))
		if w.Code != test.status || !strings.Contains(w.Body.String(), test.body) {
			t.Fatalf("%s: %d", test.path, w.Code)
		}
		if test.status == 200 && w.Header().Get("Content-Security-Policy") == "" {
			t.Fatal("missing CSP")
		}
	}
}
