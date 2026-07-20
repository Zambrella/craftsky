package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/relationships"
)

type relationshipMutationSpy struct {
	action  string
	owner   syntax.DID
	subject syntax.DID
	sid     string
}

func (s *relationshipMutationSpy) Mute(_ context.Context, owner, subject syntax.DID) (relationships.State, error) {
	s.action, s.owner, s.subject = "mute", owner, subject
	return relationships.State{Muted: true}, nil
}

func (s *relationshipMutationSpy) Unmute(_ context.Context, owner, subject syntax.DID) (relationships.State, error) {
	s.action, s.owner, s.subject = "unmute", owner, subject
	return relationships.State{}, nil
}

func (s *relationshipMutationSpy) Block(_ context.Context, owner, subject syntax.DID, sid string) (relationships.BlockMutationResult, error) {
	s.action, s.owner, s.subject, s.sid = "block", owner, subject, sid
	return relationships.BlockMutationResult{
		State: relationships.State{Blocking: true},
		URI:   "at://did:plc:alice/app.bsky.graph.block/block-1",
		CID:   "bafyblock1",
		Rkey:  "block-1",
	}, nil
}

func (s *relationshipMutationSpy) Unblock(_ context.Context, owner, subject syntax.DID, sid string) (relationships.BlockMutationResult, error) {
	s.action, s.owner, s.subject, s.sid = "unblock", owner, subject, sid
	return relationships.BlockMutationResult{State: relationships.State{}}, nil
}

type apiMembershipFake struct {
	current map[syntax.DID]bool
}

func (f apiMembershipFake) IsCurrentMember(_ context.Context, did syntax.DID) (bool, error) {
	return f.current[did], nil
}

func TestRelationshipHandlersResolveAndRejectTargetsBeforeMutation(t *testing.T) {
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")

	type handlerFactory func(*relationshipMutationSpy, relationships.MembershipLookup, api.HandleResolver) http.Handler
	operations := []struct {
		name       string
		method     string
		wantAction string
		factory    handlerFactory
	}{
		{name: "mute", method: http.MethodPost, wantAction: "mute", factory: func(s *relationshipMutationSpy, m relationships.MembershipLookup, r api.HandleResolver) http.Handler {
			return api.MuteProfileHandler(s, m, r, nilLogger())
		}},
		{name: "unmute", method: http.MethodDelete, wantAction: "unmute", factory: func(s *relationshipMutationSpy, m relationships.MembershipLookup, r api.HandleResolver) http.Handler {
			return api.UnmuteProfileHandler(s, m, r, nilLogger())
		}},
		{name: "block", method: http.MethodPost, wantAction: "block", factory: func(s *relationshipMutationSpy, m relationships.MembershipLookup, r api.HandleResolver) http.Handler {
			return api.BlockProfileHandler(s, m, r, nilLogger())
		}},
		{name: "unblock", method: http.MethodDelete, wantAction: "unblock", factory: func(s *relationshipMutationSpy, m relationships.MembershipLookup, r api.HandleResolver) http.Handler {
			return api.UnblockProfileHandler(s, m, r, nilLogger())
		}},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			tests := []struct {
				name       string
				raw        string
				resolver   fakeResolver
				membership apiMembershipFake
				wantStatus int
				wantCode   string
				wantCalled bool
			}{
				{name: "valid handle", raw: "bob.example", resolver: fakeResolver{didFor: bob}, membership: apiMembershipFake{current: map[syntax.DID]bool{bob: true}}, wantStatus: http.StatusOK, wantCalled: true},
				{name: "valid DID", raw: bob.String(), resolver: fakeResolver{}, membership: apiMembershipFake{current: map[syntax.DID]bool{bob: true}}, wantStatus: http.StatusOK, wantCalled: true},
				{name: "invalid identifier", raw: "NOT VALID", resolver: fakeResolver{}, membership: apiMembershipFake{}, wantStatus: http.StatusBadRequest, wantCode: "invalid_identifier"},
				{name: "self", raw: alice.String(), resolver: fakeResolver{}, membership: apiMembershipFake{current: map[syntax.DID]bool{alice: true}}, wantStatus: http.StatusBadRequest, wantCode: "self_relationship_not_allowed"},
				{name: "resolvable nonmember", raw: "outside.example", resolver: fakeResolver{didFor: "did:plc:outside"}, membership: apiMembershipFake{}, wantStatus: http.StatusNotFound, wantCode: "profile_not_found"},
				{name: "unknown DID", raw: "did:plc:unknown", resolver: fakeResolver{}, membership: apiMembershipFake{}, wantStatus: http.StatusNotFound, wantCode: "profile_not_found"},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					spy := &relationshipMutationSpy{}
					h := operation.factory(spy, tt.membership, tt.resolver)
					req := httptest.NewRequest(operation.method, "/v1/profiles/target/relationship", nil)
					req.SetPathValue("handleOrDid", tt.raw)
					ctx := middleware.WithDID(req.Context(), alice)
					ctx = middleware.WithOAuthSessionID(ctx, "session-alice")
					req = req.WithContext(ctx)
					rr := httptest.NewRecorder()

					h.ServeHTTP(rr, req)

					if rr.Code != tt.wantStatus {
						t.Fatalf("status = %d, want %d; body=%s", rr.Code, tt.wantStatus, rr.Body.String())
					}
					if tt.wantCalled {
						if spy.action != operation.wantAction || spy.owner != alice || spy.subject != bob {
							t.Fatalf("mutation = %q %s -> %s, want %q %s -> %s", spy.action, spy.owner, spy.subject, operation.wantAction, alice, bob)
						}
						if operation.wantAction == "block" || operation.wantAction == "unblock" {
							if spy.sid != "session-alice" {
								t.Fatalf("session id = %q, want session-alice", spy.sid)
							}
						}
						return
					}
					if spy.action != "" {
						t.Fatalf("rejected target called mutation %q", spy.action)
					}
					var body map[string]any
					if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
						t.Fatalf("decode error response: %v", err)
					}
					if body["error"] != tt.wantCode {
						t.Fatalf("error = %v, want %s", body["error"], tt.wantCode)
					}
				})
			}
		})
	}
}
