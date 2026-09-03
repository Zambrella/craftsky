package api

const (
	DefaultMaxPostImages       = 4
	DefaultMaxImageUploadBytes = 2_000_000

	// MaxImageUploadBytes preserves the historical default for tests and
	// callers that do not need deployment-specific limits.
	MaxImageUploadBytes int64 = DefaultMaxImageUploadBytes
)

// MediaLimits holds the AppView media policy used by upload and create-post
// handlers. Zero values are normalized to the approved API defaults.
type MediaLimits struct {
	MaxPostImages       int
	MaxImageUploadBytes int64
}

func DefaultMediaLimits() MediaLimits {
	return MediaLimits{
		MaxPostImages:       DefaultMaxPostImages,
		MaxImageUploadBytes: DefaultMaxImageUploadBytes,
	}
}

func normalizeMediaLimits(limits MediaLimits) MediaLimits {
	defaults := DefaultMediaLimits()
	if limits.MaxPostImages == 0 {
		limits.MaxPostImages = defaults.MaxPostImages
	}
	if limits.MaxImageUploadBytes == 0 {
		limits.MaxImageUploadBytes = defaults.MaxImageUploadBytes
	}
	return limits
}
