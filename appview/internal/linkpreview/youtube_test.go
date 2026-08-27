package linkpreview

import "testing"

func TestParseYouTubeURL(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ": "dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ?t=90":           "dQw4w9WgXcQ",
		"https://m.youtube.com/shorts/dQw4w9WgXcQ":    "dQw4w9WgXcQ",
		"https://youtube.com/live/dQw4w9WgXcQ":        "dQw4w9WgXcQ",
	}
	for raw, wantVideoID := range tests {
		raw, wantVideoID := raw, wantVideoID
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			got, ok := parseYouTubeURL(raw)
			if !ok || got.videoID != wantVideoID {
				t.Fatalf("parseYouTubeURL(%q) = %#v, %v; want %q, true", raw, got, ok, wantVideoID)
			}
		})
	}
}

func TestParseYouTubeURLRejectsUnsupportedForms(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		"https://youtube.com.example.org/watch?v=dQw4w9WgXcQ",
		"https://youtube.com/embed/dQw4w9WgXcQ",
		"https://youtube.com/watch?v=too-short",
		"javascript:alert(1)",
	} {
		if _, ok := parseYouTubeURL(raw); ok {
			t.Errorf("parseYouTubeURL(%q) accepted an unsupported URL", raw)
		}
	}
}
