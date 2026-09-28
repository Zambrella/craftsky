package buildinfo

import "testing"

func TestVersionNormalizesOnlyStrictSemanticVersions(t *testing.T) {
	original := version
	t.Cleanup(func() { version = original })

	for _, test := range []struct {
		name string
		raw  string
		want string
	}{
		{name: "release", raw: "1.0.3", want: "1.0.3"},
		{name: "development default", raw: "dev", want: "dev"},
		{name: "empty", raw: "", want: "dev"},
		{name: "leading v", raw: "v1.0.3", want: "dev"},
		{name: "missing patch", raw: "1.0", want: "dev"},
		{name: "leading zero", raw: "1.00.3", want: "dev"},
		{name: "pre-release", raw: "1.0.3-rc.1", want: "dev"},
		{name: "build metadata", raw: "1.0.3+build.1", want: "dev"},
		{name: "surrounding whitespace", raw: " 1.0.3 ", want: "dev"},
	} {
		t.Run(test.name, func(t *testing.T) {
			version = test.raw
			if got := Version(); got != test.want {
				t.Fatalf("Version() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSentryReleaseUsesOnlySemanticVersion(t *testing.T) {
	original := version
	t.Cleanup(func() { version = original })

	version = "1.0.3"
	if got := SentryRelease(); got != "craftsky-appview@1.0.3" {
		t.Fatalf("SentryRelease() = %q", got)
	}

	version = "dev"
	if got := SentryRelease(); got != "" {
		t.Fatalf("SentryRelease() = %q, want empty for development build", got)
	}
}
