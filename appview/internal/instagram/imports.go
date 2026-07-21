package instagram

import (
	"context"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

var (
	ErrInvalidInstagramImport    = errors.New("invalid Instagram import")
	ErrInvalidInstagramPageLimit = errors.New("invalid Instagram page limit")
)

type ImportRepository interface {
	CreateImport(context.Context, CreateImportParams) (CreateImportResult, error)
	ListImports(context.Context, syntax.DID, int, *ImportCursor, time.Time) ([]GraphImport, *ImportCursor, error)
	GetImport(context.Context, syntax.DID, uuid.UUID, time.Time) (GraphImport, error)
	UpdateImport(context.Context, syntax.DID, uuid.UUID, UpdateImportParams) (GraphImport, error)
	DeleteImport(context.Context, syntax.DID, uuid.UUID, time.Time) error
}

type StagedImportRepository interface {
	ImportRepository
	CreateImportForMatching(context.Context, CreateImportParams) (CreateImportResult, error)
	FinalizeImportMatching(context.Context, syntax.DID, uuid.UUID, time.Time) error
}

type ImportSuggestionMatcher interface {
	MatchImport(context.Context, syntax.DID, uuid.UUID) (int, error)
}

type ImportServiceOptions struct {
	Repository      ImportRepository
	Matcher         ImportSuggestionMatcher
	Now             func() time.Time
	NewID           func() uuid.UUID
	MaxEntries      int
	DefaultPageSize int
	MaxPageSize     int
}

type ImportService struct {
	repository      ImportRepository
	matcher         ImportSuggestionMatcher
	now             func() time.Time
	newID           func() uuid.UUID
	maxEntries      int
	defaultPageSize int
	maxPageSize     int
}

func NewImportService(options ImportServiceOptions) (*ImportService, error) {
	if options.Repository == nil {
		return nil, errors.New("Instagram import repository is required")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.NewID == nil {
		options.NewID = uuid.New
	}
	if options.MaxEntries == 0 {
		options.MaxEntries = MaxImportEntries
	}
	if options.DefaultPageSize == 0 {
		options.DefaultPageSize = 20
	}
	if options.MaxPageSize == 0 {
		options.MaxPageSize = 50
	}
	if options.MaxEntries < 1 || options.MaxEntries > MaxImportEntries ||
		options.DefaultPageSize < 1 || options.MaxPageSize < options.DefaultPageSize || options.MaxPageSize > 50 {
		return nil, errors.New("invalid Instagram import service limits")
	}
	return &ImportService{
		repository:      options.Repository,
		matcher:         options.Matcher,
		now:             options.Now,
		newID:           options.NewID,
		maxEntries:      options.MaxEntries,
		defaultPageSize: options.DefaultPageSize,
		maxPageSize:     options.MaxPageSize,
	}, nil
}

func (s *ImportService) CreateImport(ctx context.Context, owner syntax.DID, sourceType ImportSourceType, retainUnmatched bool, entries []ImportEntry) (CreateImportResult, error) {
	if s == nil || s.repository == nil || owner == "" || !sourceType.Valid() {
		return CreateImportResult{}, ErrInvalidInstagramImport
	}
	normalized, err := NormalizeImportEntries(entries)
	if err != nil {
		return CreateImportResult{}, err
	}
	if len(normalized) == 0 {
		return CreateImportResult{}, ErrInvalidInstagramImport
	}
	if len(normalized) > s.maxEntries {
		return CreateImportResult{}, ErrTooManyImportEntries
	}
	params := CreateImportParams{
		ID:              s.newID(),
		OwnerDID:        owner,
		SourceType:      sourceType,
		RetainUnmatched: retainUnmatched,
		Entries:         normalized,
		Now:             s.now().UTC(),
	}
	if s.matcher == nil {
		return s.repository.CreateImport(ctx, params)
	}
	staged, ok := s.repository.(StagedImportRepository)
	if !ok {
		return CreateImportResult{}, errors.New("Instagram import repository does not support matching")
	}
	created, err := staged.CreateImportForMatching(ctx, params)
	if err != nil {
		return CreateImportResult{}, err
	}
	count, err := s.matcher.MatchImport(ctx, owner, params.ID)
	if err != nil {
		_ = s.repository.DeleteImport(context.WithoutCancel(ctx), owner, params.ID, s.now().UTC())
		return CreateImportResult{}, err
	}
	if err := staged.FinalizeImportMatching(ctx, owner, params.ID, s.now().UTC()); err != nil {
		_ = s.repository.DeleteImport(context.WithoutCancel(ctx), owner, params.ID, s.now().UTC())
		return CreateImportResult{}, err
	}
	created.InitialSuggestionCount = count
	return created, nil
}

func (s *ImportService) ListImports(ctx context.Context, owner syntax.DID, limit int, cursor *ImportCursor) ([]GraphImport, *ImportCursor, error) {
	if s == nil || s.repository == nil || owner == "" || limit < 0 {
		return nil, nil, ErrInvalidInstagramPageLimit
	}
	if limit == 0 {
		limit = s.defaultPageSize
	}
	if limit > s.maxPageSize {
		limit = s.maxPageSize
	}
	return s.repository.ListImports(ctx, owner, limit, cursor, s.now().UTC())
}

func (s *ImportService) GetImport(ctx context.Context, owner syntax.DID, id uuid.UUID) (GraphImport, error) {
	if s == nil || s.repository == nil || owner == "" || id == uuid.Nil {
		return GraphImport{}, ErrInstagramResourceNotFound
	}
	return s.repository.GetImport(ctx, owner, id, s.now().UTC())
}

func (s *ImportService) UpdateImport(ctx context.Context, owner syntax.DID, id uuid.UUID, retainUnmatched, reactivate *bool) (GraphImport, error) {
	if s == nil || s.repository == nil || owner == "" || id == uuid.Nil {
		return GraphImport{}, ErrInstagramResourceNotFound
	}
	if retainUnmatched == nil && reactivate == nil {
		return GraphImport{}, ErrInvalidInstagramImport
	}
	return s.repository.UpdateImport(ctx, owner, id, UpdateImportParams{
		RetainUnmatched: retainUnmatched,
		Reactivate:      reactivate,
		Now:             s.now().UTC(),
	})
}

func (s *ImportService) DeleteImport(ctx context.Context, owner syntax.DID, id uuid.UUID) error {
	if s == nil || s.repository == nil {
		return errors.New("Instagram import service is unavailable")
	}
	if owner == "" || id == uuid.Nil {
		return nil
	}
	return s.repository.DeleteImport(ctx, owner, id, s.now().UTC())
}

var _ ImportRepository = (*ImportStore)(nil)
