package index

import (
	"testing"

	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
)

// UT-014: only the exact recognized source activates import semantics.
func TestRecognizedExternalImportSource(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		rec  craftskylex.FeedPost
		want string
	}{
		{name: "absent", rec: craftskylex.FeedPost{}},
		{
			name: "instagram",
			rec: craftskylex.FeedPost{
				ExternalImport: &craftskylex.FeedPost_ExternalImport{Source: "instagram"},
			},
			want: "instagram",
		},
		{
			name: "unknown future source",
			rec: craftskylex.FeedPost{
				ExternalImport: &craftskylex.FeedPost_ExternalImport{Source: "future-source"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := recognizedExternalImportSource(&tc.rec); got != tc.want {
				t.Fatalf("recognizedExternalImportSource() = %q, want %q", got, tc.want)
			}
		})
	}
}
