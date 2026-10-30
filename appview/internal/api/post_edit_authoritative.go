package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
	lexiconschema "social.craftsky/appview/internal/lexicon/schema"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/sourcevalidation"
	"social.craftsky/appview/internal/tap"
)

var ErrPostEditRecordInvalid = errors.New("post: authoritative edit record invalid")

// PostEditRecordReader exposes only an owner-fenced published-post read.
type PostEditRecordReader func(context.Context, syntax.DID, syntax.RecordKey, int64, string) (json.RawMessage, syntax.CID, error)

// PostEditSeedLoader supplies a private editor snapshot, never public serving proof.
type PostEditSeedLoader func(context.Context, syntax.DID, syntax.RecordKey, int64, string) (*PostEditSnapshot, error)

// NewPostEditRecordReader narrows the existing lifecycle/session-fenced read
// capability to this owner's post collection. It creates no mutation attempt.
func NewPostEditRecordReader(factory pdseffects.ExecutorFactory) PostEditRecordReader {
	return func(ctx context.Context, owner syntax.DID, key syntax.RecordKey, generation int64, session string) (json.RawMessage, syntax.CID, error) {
		if factory == nil {
			return nil, "", errors.New("post edit record reader unavailable")
		}
		executor, err := factory(ctx, owner, session)
		if err != nil {
			return nil, "", err
		}
		var raw json.RawMessage
		cid, err := executor.ReadRecord(ctx, pdseffects.ReadRecordRequest{Owner: owner, OwnerGeneration: generation, ExpectedOwners: []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: generation}}, Collection: syntax.NSID(craftskyPostNSID), Rkey: key}, &raw)
		return raw, cid, err
	}
}

func NewPostEditSeedLoader(store *PostStore, read PostEditRecordReader) PostEditSeedLoader {
	return func(ctx context.Context, owner syntax.DID, key syntax.RecordKey, generation int64, session string) (*PostEditSnapshot, error) {
		if err := store.CheckPostEditAllowed(ctx, owner, key); err != nil {
			return nil, err
		}
		if read == nil {
			return nil, errors.New("post edit record reader unavailable")
		}
		raw, cid, err := read(ctx, owner, key, generation, session)
		if errors.Is(err, auth.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		if err != nil {
			return nil, err
		}
		if cid == "" {
			return nil, ErrPostEditRecordInvalid
		}
		if err := lexiconschema.ValidateCraftskyRecord(raw, craftskyPostNSID); err != nil {
			return nil, errors.Join(ErrPostEditRecordInvalid, err)
		}
		validation := sourcevalidation.Validate(tap.Event{Collection: craftskyPostNSID, Rkey: key, Action: "update", Record: raw})
		if validation.StructuralStatus != sourcevalidation.Valid || validation.SemanticStatus != sourcevalidation.Valid {
			return nil, errors.Join(ErrPostEditRecordInvalid, validation.Cause)
		}
		candidate, err := postEditCandidate(raw)
		if err != nil {
			return nil, errors.Join(ErrPostEditRecordInvalid, err)
		}
		// The presentation must not provide a second, raw extension round-trip
		// alongside the bounded editable projection.
		supported, err := postEditEditable(raw)
		if err != nil {
			return nil, errors.Join(ErrPostEditRecordInvalid, err)
		}
		if project, ok := supported["project"]; ok {
			candidate.Project = &Project{}
			if err := json.Unmarshal(project, candidate.Project); err != nil {
				return nil, errors.Join(ErrPostEditRecordInvalid, err)
			}
		}
		var fields struct {
			CreatedAt      time.Time       `json:"createdAt"`
			EditedAt       *time.Time      `json:"editedAt"`
			Embed          json.RawMessage `json:"embed"`
			Crafts         json.RawMessage `json:"crafts"`
			ExternalImport *struct {
				Source string `json:"source"`
			} `json:"externalImport"`
		}
		if err := json.Unmarshal(raw, &fields); err != nil {
			return nil, errors.Join(ErrPostEditRecordInvalid, err)
		}
		images, err := syntheticPostImagesJSON(candidate.Images)
		if err != nil {
			return nil, fmt.Errorf("post edit image presentation: %w", err)
		}
		author, err := store.ReadAuthor(ctx, owner.String())
		if err != nil {
			return nil, err
		}
		row := &PostRow{URI: "at://" + owner.String() + "/" + craftskyPostNSID + "/" + key.String(), DID: owner.String(), Rkey: key.String(), CID: cid.String(), Text: candidate.Text, Sponsored: candidate.Sponsored, Facets: candidate.Facets, Langs: candidate.Langs, Project: candidate.Project, CreatedAt: fields.CreatedAt, IndexedAt: time.Now().UTC(), Images: images, RawEmbed: fields.Embed, Crafts: fields.Crafts, EditedAt: fields.EditedAt}
		if author != nil {
			row.AuthorDisplayName = author.DisplayName
			row.AuthorAvatarCID = author.AvatarCID
			row.AuthorAvatarMime = author.AvatarMime
		}
		if candidate.Reply != nil {
			row.ReplyParentURI = &candidate.Reply.Parent.URI
			row.ReplyParentCID = &candidate.Reply.Parent.CID
			row.ReplyRootURI = &candidate.Reply.Root.URI
			row.ReplyRootCID = &candidate.Reply.Root.CID
		}
		if candidate.Embed != nil && candidate.Embed.Quote != nil {
			row.QuoteURI = &candidate.Embed.Quote.URI
			row.QuoteCID = &candidate.Embed.Quote.CID
		}
		if fields.ExternalImport != nil {
			row.ExternalImportSource = &fields.ExternalImport.Source
		}
		// Policy may change while the remote read is in flight. Recheck before
		// exposing the private seed; the PDS read itself holds its owner/session fence.
		if err := store.CheckPostEditAllowed(ctx, owner, key); err != nil {
			return nil, err
		}
		return &PostEditSnapshot{Post: row, Record: raw}, nil
	}
}
