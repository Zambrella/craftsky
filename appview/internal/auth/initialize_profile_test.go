// appview/internal/auth/initialize_profile_test.go
package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

type mockPDS struct {
	getCalls []getCall
	putCalls []putCall

	getRecord func(collection, rkey string, out any) (string, error)
	putRecord func(collection, rkey string, record any) error
}

type getCall struct{ Collection, Rkey string }
type putCall struct {
	Collection, Rkey string
	Record           any
}

func (m *mockPDS) GetRecord(_ context.Context, _ syntax.DID, collection, rkey string, out any) (string, error) {
	m.getCalls = append(m.getCalls, getCall{collection, rkey})
	return m.getRecord(collection, rkey, out)
}
func (m *mockPDS) PutRecord(_ context.Context, _ syntax.DID, collection, rkey string, record any) error {
	m.putCalls = append(m.putCalls, putCall{collection, rkey, record})
	return m.putRecord(collection, rkey, record)
}
func (m *mockPDS) CreateRecord(_ context.Context, _ syntax.DID, _ string, _ any) (syntax.ATURI, syntax.CID, error) {
	return "", "", nil
}
func (m *mockPDS) DeleteRecord(_ context.Context, _ syntax.DID, _, _ string) error {
	return nil
}
func (m *mockPDS) UploadBlob(_ context.Context, _ string, _ []byte) (*auth.UploadedBlob, error) {
	return nil, nil
}

const (
	bskyNSID = "app.bsky.actor.profile"
	cskyNSID = "social.craftsky.actor.profile"
)

func TestInitializeProfile_ReturningUserBothPresent(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) (string, error) {
			switch coll {
			case bskyNSID:
				*(out.(*map[string]any)) = map[string]any{"displayName": "Alice"}
				return "", nil
			case cskyNSID:
				*(out.(*map[string]any)) = map[string]any{
					"$type":  cskyNSID,
					"crafts": []any{"sewing"},
				}
				return "", nil
			}
			t.Fatalf("unexpected get collection %q", coll)
			return "", nil
		},
		putRecord: func(_, _ string, _ any) error {
			t.Fatalf("PutRecord should not be called for returning user")
			return nil
		},
	}
	if err := auth.InitializeProfile(context.Background(), m, syntax.DID("did:plc:a")); err != nil {
		t.Fatalf("InitializeProfile: %v", err)
	}
	if len(m.getCalls) != 2 {
		t.Errorf("getCalls = %d, want 2", len(m.getCalls))
	}
	if len(m.putCalls) != 0 {
		t.Errorf("putCalls = %d, want 0", len(m.putCalls))
	}
}

func TestInitializeProfile_NewUserWritesEmptyCraftsky(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) (string, error) {
			switch coll {
			case bskyNSID:
				*(out.(*map[string]any)) = map[string]any{"displayName": "Alice"}
				return "", nil
			case cskyNSID:
				return "", auth.ErrRecordNotFound
			}
			return "", nil
		},
		putRecord: func(coll, rkey string, record any) error {
			if coll != cskyNSID {
				t.Errorf("put collection = %q, want %q", coll, cskyNSID)
			}
			if rkey != "self" {
				t.Errorf("put rkey = %q, want self", rkey)
			}
			body, _ := record.(map[string]any)
			if body["$type"] != cskyNSID {
				t.Errorf("put $type = %v", body["$type"])
			}
			c, _ := body["crafts"].([]string)
			if len(c) != 0 {
				t.Errorf("put crafts = %v, want empty", c)
			}
			return nil
		},
	}
	if err := auth.InitializeProfile(context.Background(), m, syntax.DID("did:plc:b")); err != nil {
		t.Fatalf("InitializeProfile: %v", err)
	}
	if len(m.putCalls) != 1 {
		t.Errorf("putCalls = %d, want 1", len(m.putCalls))
	}
}

func TestInitializeProfile_NoBlueskyProfileIsOK(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, _ any) (string, error) {
			return "", auth.ErrRecordNotFound
		},
		putRecord: func(coll, _ string, _ any) error {
			if coll != cskyNSID {
				t.Errorf("put collection = %q, want %q", coll, cskyNSID)
			}
			return nil
		},
	}
	if err := auth.InitializeProfile(context.Background(), m, syntax.DID("did:plc:c")); err != nil {
		t.Fatalf("InitializeProfile: %v", err)
	}
}

func TestInitializeProfile_BlueskyReadErrorFails(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	m := &mockPDS{
		getRecord: func(coll, _ string, _ any) (string, error) {
			if coll == bskyNSID {
				return "", boom
			}
			return "", nil
		},
		putRecord: func(_, _ string, _ any) error { return nil },
	}
	err := auth.InitializeProfile(context.Background(), m, syntax.DID("did:plc:d"))
	if err == nil {
		t.Fatal("want error; got nil")
	}
	if !errors.Is(err, auth.ErrProfileInitFailed) {
		t.Errorf("want ErrProfileInitFailed; got %v", err)
	}
}

func TestInitializeProfile_CraftskyReadErrorFails(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) (string, error) {
			if coll == bskyNSID {
				*(out.(*map[string]any)) = map[string]any{}
				return "", nil
			}
			return "", errors.New("boom")
		},
		putRecord: func(_, _ string, _ any) error { return nil },
	}
	err := auth.InitializeProfile(context.Background(), m, syntax.DID("did:plc:e"))
	if !errors.Is(err, auth.ErrProfileInitFailed) {
		t.Errorf("want ErrProfileInitFailed; got %v", err)
	}
}

func TestInitializeProfile_MalformedCraftskyRecord(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) (string, error) {
			if coll == bskyNSID {
				*(out.(*map[string]any)) = map[string]any{}
				return "", nil
			}
			// crafts expected to be []string; return wrong type.
			*(out.(*map[string]any)) = map[string]any{
				"$type":  cskyNSID,
				"crafts": "not an array",
			}
			return "", nil
		},
		putRecord: func(_, _ string, _ any) error { return nil },
	}
	err := auth.InitializeProfile(context.Background(), m, syntax.DID("did:plc:f"))
	if !errors.Is(err, auth.ErrProfileDataInvalid) {
		t.Errorf("want ErrProfileDataInvalid; got %v", err)
	}
}

func TestInitializeProfile_PutRecordFailure(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) (string, error) {
			if coll == bskyNSID {
				*(out.(*map[string]any)) = map[string]any{}
				return "", nil
			}
			return "", auth.ErrRecordNotFound
		},
		putRecord: func(_, _ string, _ any) error { return errors.New("pds down") },
	}
	err := auth.InitializeProfile(context.Background(), m, syntax.DID("did:plc:g"))
	if !errors.Is(err, auth.ErrProfileInitFailed) {
		t.Errorf("want ErrProfileInitFailed; got %v", err)
	}
}
