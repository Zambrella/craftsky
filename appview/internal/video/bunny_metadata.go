package video

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

// Collections are an operator convenience, never an ownership or cleanup source.
// A transaction advisory lock serializes first uploads across AppView instances;
// looking up the exact DID on each allocation also survives restarts and manual
// collection removal without introducing a second ownership mapping.
func (b *Bunny) ownerCollection(ctx context.Context, owner syntax.DID) (string, error) {
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "craftsky.bunny.collection:"+b.library+":"+owner.String()); err != nil {
		return "", err
	}
	for page := 1; page <= 100; page++ {
		query := url.Values{"search": {owner.String()}, "page": {strconv.Itoa(page)}, "itemsPerPage": {"100"}, "includeThumbnails": {"false"}}
		var result struct {
			Total int `json:"totalItems"`
			Items []struct {
				GUID string `json:"guid"`
				Name string `json:"name"`
			} `json:"items"`
		}
		if err = b.libraryAPI(ctx, "GET", "/collections?"+query.Encode(), nil, &result); err != nil {
			return "", err
		}
		for _, item := range result.Items {
			// Bunny search is a case-insensitive substring search; enforce exactness.
			if item.Name == owner.String() {
				if _, err = uuid.Parse(item.GUID); err != nil {
					return "", ErrBunnyUnavailable
				}
				return item.GUID, nil
			}
		}
		if page*100 >= result.Total {
			var created struct {
				GUID string `json:"guid"`
			}
			if err = b.libraryAPI(ctx, "POST", "/collections", map[string]string{"name": owner.String()}, &created); err != nil {
				return "", err
			}
			if _, err = uuid.Parse(created.GUID); err != nil {
				return "", ErrBunnyUnavailable
			}
			return created.GUID, nil
		}
	}
	return "", ErrBunnyUnavailable
}

type bunnyAllocation struct {
	ID, Title, CollectionID string
}

func (b *Bunny) createOrganizedVideo(ctx context.Context, owner syntax.DID, job string, createdAt time.Time) (bunnyAllocation, error) {
	collection, err := b.ownerCollection(ctx, owner)
	if err != nil {
		return bunnyAllocation{}, err
	}
	var created struct {
		GUID string `json:"guid"`
	}
	title := createdAt.UTC().Format("2006-01-02 15:04 UTC") + " · " + job[:8]
	if err = b.api(ctx, "POST", "", map[string]string{"title": title, "collectionId": collection}, &created); err != nil {
		return bunnyAllocation{}, err
	}
	if _, err = uuid.Parse(created.GUID); err != nil {
		return bunnyAllocation{}, ErrBunnyUnavailable
	}
	return bunnyAllocation{ID: created.GUID, Title: title, CollectionID: collection}, nil
}

// Only technical references cross this boundary. Never copy post/draft text,
// original filenames, credentials or a single post URI into shared video metadata.
func (b *Bunny) setVideoMetadata(ctx context.Context, videoID, job, source, cid string) error {
	type tag struct {
		Property string `json:"property"`
		Value    string `json:"value"`
	}
	tags := []tag{{"craftsky:uploadJobId", job}, {"craftsky:source", source}}
	if cid != "" {
		tags = append(tags, tag{"craftsky:pdsBlobCid", cid})
	}
	return b.api(ctx, "POST", "/"+videoID, struct {
		Tags []tag `json:"metaTags"`
	}{tags}, nil)
}
