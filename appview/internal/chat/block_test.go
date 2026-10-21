package chat

import (
	"errors"
	"github.com/google/uuid"
	"io"
	"net/http"
	"social.craftsky/appview/internal/auth"
	"strings"
	"testing"
)

// IT-014 / RULE-001: either block direction prevents creation and sends,
// while an upstream-accessible direct history/summary remains readable.
func TestEitherDirectionBlockPreservesReadAndPreventsEffects(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(map[bool]string{false: "owner blocks", true: "peer blocks"}[reverse], func(t *testing.T) {
			service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
				body := `{"chatDisabled":false}`
				switch r.URL.Path {
				case "/xrpc/chat.bsky.actor.getStatus":
				case "/xrpc/chat.bsky.convo.getConvo":
					body = `{"convo":{"id":"direct-1","status":"accepted","members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:external-peer"}]}}`
				case "/xrpc/chat.bsky.convo.listConvos":
					body = `{"convos":[{"id":"direct-1","status":"accepted","unreadCount":1,"members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:external-peer"}]}]}`
				case "/xrpc/chat.bsky.convo.listConvoRequests":
					body = `{"requests":[]}`
				default:
					t.Fatalf("blocked relationship reached effect %s", r.URL.Path)
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			}))
			actor, subject := info.DID.String(), "did:plc:external-peer"
			if reverse {
				actor, subject = subject, actor
			}
			if _, err := store.pool.Exec(t.Context(), `INSERT INTO tap_source_records(uri,did,collection,rkey,source_event_id,source_fingerprint,revision,cid,action,record,record_bytes,live,ordering_status,projection_disposition,structural_validation_status,semantic_validation_status) VALUES('at://synthetic/block/test',$1,'app.bsky.graph.block','test',1,decode(repeat('00',32),'hex'),'rev','cid','create','{}'::json,2,false,'authoritative','eligible','valid','valid')`, actor); err != nil {
				t.Fatal(err)
			}
			if _, err := store.pool.Exec(t.Context(), `INSERT INTO pds_set_sources(source_uri,kind,actor_did,scope_key,subject_did,activity_at,eligible) VALUES('at://synthetic/block/test','block',$1,$2,$2,now(),true)`, actor, subject); err != nil {
				t.Fatal(err)
			}
			if _, err := store.pool.Exec(t.Context(), `INSERT INTO pds_set_aggregates(kind,actor_did,scope_key,subject_did,eligible_source_count,representative_source_uri,activated_at) VALUES('block',$1,$2,$2,1,'at://synthetic/block/test',now())`, actor, subject); err != nil {
				t.Fatal(err)
			}
			summary, err := service.Conversation(t.Context(), info, "direct-1")
			if err != nil || summary.CanSend {
				t.Fatalf("blocked summary=%+v err=%v", summary, err)
			}
			counts, err := service.UnreadCounts(t.Context(), info)
			if err != nil || counts.Accepted != 0 {
				t.Fatalf("local block contributed to badge: counts=%+v err=%v", counts, err)
			}
			available, err := service.Availability(t.Context(), info, "did:plc:external-peer")
			if err != nil || available.CanChat {
				t.Fatalf("blocked availability=%+v err=%v", available, err)
			}
			if _, err := service.StartConversation(t.Context(), info, "did:plc:external-peer"); !errors.Is(err, ErrBlocked) {
				t.Fatalf("start err=%v", err)
			}
			if _, err := service.Send(t.Context(), info, uuid.New(), "direct-1", "retain this draft"); !errors.Is(err, ErrBlocked) {
				t.Fatalf("send err=%v", err)
			}
		})
	}
}
