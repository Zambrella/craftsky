package api

import (
	"context"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func authoritativeContentLanguages(
	ctx context.Context,
	viewerDID syntax.DID,
	preferenceReaders []LanguagePreferenceReader,
) ([]string, error) {
	if len(preferenceReaders) == 0 {
		return []string{}, nil
	}
	preferences, err := preferenceReaders[0].Get(ctx, viewerDID)
	if err != nil {
		return nil, err
	}
	return preferences.ContentLanguages, nil
}

func languageVisibilityPredicate(
	postAlias string,
	viewerParameter string,
	languagesParameter string,
) string {
	if postAlias != "p" {
		panic("unsupported post alias for language visibility")
	}
	return `
			  AND (
				` + postAlias + `.did = ` + viewerParameter + `
				OR cardinality(` + languagesParameter + `::text[]) = 0
				OR ` + postAlias + `.langs && ` + languagesParameter + `::text[]
			  )
	`
}
