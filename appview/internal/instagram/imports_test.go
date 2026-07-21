package instagram

import (
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

func TestImportServiceOwnsIdentityTimeAndFixedBounds(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 19, 14, 0, 0, 0, time.UTC)
	id := uuid.MustParse("00000000-0000-0000-0000-000000000221")
	repository := &recordingImportRepository{}
	service, err := NewImportService(ImportServiceOptions{
		Repository:      repository,
		Now:             func() time.Time { return now },
		NewID:           func() uuid.UUID { return id },
		DefaultPageSize: 20,
		MaxPageSize:     50,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	owner := syntax.DID("did:plc:synthetic-alice")
	if _, err := service.CreateImport(context.Background(), owner, ImportSourceManual, true, []ImportEntry{{Username: "Synthetic.User", Direction: DirectionFollowing}}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if repository.created.ID != id || repository.created.OwnerDID != owner || !repository.created.Now.Equal(now) {
		t.Fatalf("create params = %+v", repository.created)
	}
	if _, _, err := service.ListImports(context.Background(), owner, 0, nil); err != nil {
		t.Fatalf("list default: %v", err)
	}
	if repository.listLimit != 20 {
		t.Fatalf("default limit = %d", repository.listLimit)
	}
	if _, _, err := service.ListImports(context.Background(), owner, 51, nil); err != nil {
		t.Fatalf("list clamped: %v", err)
	}
	if repository.listLimit != 50 {
		t.Fatalf("clamped limit = %d", repository.listLimit)
	}
	if _, _, err := service.ListImports(context.Background(), owner, -1, nil); err != ErrInvalidInstagramPageLimit {
		t.Fatalf("negative limit error = %v", err)
	}
}

type recordingImportRepository struct {
	created   CreateImportParams
	listLimit int
}

func (r *recordingImportRepository) CreateImport(_ context.Context, params CreateImportParams) (CreateImportResult, error) {
	r.created = params
	return CreateImportResult{}, nil
}

func (r *recordingImportRepository) ListImports(_ context.Context, _ syntax.DID, limit int, _ *ImportCursor, _ time.Time) ([]GraphImport, *ImportCursor, error) {
	r.listLimit = limit
	return nil, nil, nil
}

func (r *recordingImportRepository) GetImport(context.Context, syntax.DID, uuid.UUID, time.Time) (GraphImport, error) {
	return GraphImport{}, nil
}

func (r *recordingImportRepository) UpdateImport(context.Context, syntax.DID, uuid.UUID, UpdateImportParams) (GraphImport, error) {
	return GraphImport{}, nil
}

func (r *recordingImportRepository) DeleteImport(context.Context, syntax.DID, uuid.UUID, time.Time) error {
	return nil
}

var _ ImportRepository = (*recordingImportRepository)(nil)
