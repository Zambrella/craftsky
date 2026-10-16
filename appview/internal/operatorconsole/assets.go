package operatorconsole

import (
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
)

func Assets(files fs.FS, playbackURLs ...string) http.Handler {
	origins := []string{"https://video.bsky.app"}
	for _, raw := range playbackURLs {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
			continue
		}
		origin := "https://" + parsed.Host
		found := false
		for _, prior := range origins {
			if prior == origin {
				found = true
			}
		}
		if !found {
			origins = append(origins, origin)
		}
	}
	videoOrigins := strings.Join(origins, " ")
	policy := "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' blob: https://cdn.bsky.app " + videoOrigins + "; media-src 'self' blob: " + videoOrigins + "; connect-src 'self' " + videoOrigins + "; frame-ancestors 'none'; form-action 'self'; base-uri 'none'"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		noStore(w)
		w.Header().Set("Content-Security-Policy", policy)
		w.Header().Set("Referrer-Policy", "no-referrer")
		if !strings.HasPrefix(r.URL.Path, "/admin/") {
			http.NotFound(w, r)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/admin/")
		if name == "" {
			name = "index.html"
		}
		if !fs.ValidPath(name) {
			http.NotFound(w, r)
			return
		}
		data, err := fs.ReadFile(files, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		contentType := mime.TypeByExtension(path.Ext(name))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", contentType)
		w.Write(data)
	})
}
