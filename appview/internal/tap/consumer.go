// Package tap consumes atproto events from a Tap sidecar over WebSocket.
//
// WSConsumer is the production consumer. It dials Tap's /channel WS endpoint,
// dispatches each record envelope to an indexer, and acks on success.
// Identity envelopes are acked and dropped at debug level. Reconnection uses
// exponential backoff capped at WSConsumerConfig.ReconnectMax. A poison-pill
// guard drops an event after MaxRetries consecutive Handle failures.
package tap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Event is one decoded record-event from Tap's /channel WebSocket.
// Identity events are consumed internally by the consumer and are not
// surfaced to indexers.
//
// DID, Collection, and Rkey are validated by the consumer before this
// struct is handed to an indexer; CID is a passthrough (per indigo's
// syntax.CID docstring, the typed wrapper is informal and the real
// validator is the ipfs/go-cid package).
type Event struct {
	URI        syntax.ATURI     // at://did/collection/rkey
	CID        syntax.CID       // content identifier of the record
	DID        syntax.DID       // repo owner
	Collection syntax.NSID      // e.g. "app.bsky.feed.post"
	Rkey       syntax.RecordKey // record key
	Action     string           // "create" | "update" | "delete"
	Record     json.RawMessage  // opaque JSON; nil or empty on Action == "delete"
	Live       bool             // false during backfill, true for steady-state
	ID         uint64           // Tap's per-event "id" field from the envelope
	Rev        string           // repo rev at time of event
}

// Consumer is the interface the appview uses to consume events from Tap.
type Consumer interface {
	// Run blocks until ctx is cancelled, continuously connecting to Tap
	// and dispatching events to the configured indexer. It always returns
	// a non-nil error; on graceful shutdown the error is ctx.Err().
	Run(ctx context.Context) error

	// State returns a snapshot of the consumer's current connection state.
	// Safe to call concurrently with Run.
	State() ConnState
}

// ConnState describes the consumer's current connection state; used by the
// /healthz handler and the `cli tap status` command.
type ConnState struct {
	Connected        bool
	LastEventAt      time.Time
	LastError        string
	ReconnectAttempt int
}

// NotImplemented is a stub consumer used until WSConsumer lands.
// Run returns an error immediately; State reports disconnected.
type NotImplemented struct{}

var _ Consumer = NotImplemented{}

func (NotImplemented) Run(ctx context.Context) error {
	return errors.New("tap: consumer not yet implemented")
}

func (NotImplemented) State() ConnState {
	return ConnState{LastError: "not implemented"}
}

// WSConsumerConfig wires a WSConsumer. All fields are required.
type WSConsumerConfig struct {
	URL          string // ws://tap:2480/channel
	Indexer      HandlerIndexer
	AckTimeout   time.Duration // per-event Handle deadline
	ReconnectMax time.Duration // cap for exponential reconnect backoff
	MaxRetries   int           // poison-pill threshold per event id
	Logger       *slog.Logger  // optional; nil → slog.Default()
}

// HandlerIndexer is the narrow interface the consumer needs. Defined here
// (not imported from internal/index) to avoid an import cycle.
type HandlerIndexer interface {
	Handle(ctx context.Context, ev Event) error
}

// WSConsumer connects to Tap's /channel WebSocket and dispatches events
// to an indexer, sending acks on success.
type WSConsumer struct {
	cfg    WSConsumerConfig
	logger *slog.Logger

	mu         sync.Mutex
	state      ConnState
	retryCount map[uint64]int // event id → how many times it's failed
}

var _ Consumer = (*WSConsumer)(nil)

// NewWSConsumer returns a consumer that connects to the given Tap WS URL.
func NewWSConsumer(cfg WSConsumerConfig) *WSConsumer {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &WSConsumer{
		cfg:        cfg,
		logger:     logger,
		retryCount: map[uint64]int{},
	}
}

func (c *WSConsumer) State() ConnState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

func (c *WSConsumer) setConnected(connected bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Connected = connected
	if connected {
		c.state.LastError = ""
		c.state.ReconnectAttempt = 0
	}
}

func (c *WSConsumer) recordError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Connected = false
	c.state.LastError = err.Error()
	c.state.ReconnectAttempt++
}

func (c *WSConsumer) recordEvent() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.LastEventAt = time.Now().UTC()
}

// Run loops forever connecting, reading, and reconnecting on error.
// Returns only when ctx is cancelled.
func (c *WSConsumer) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := c.runOnce(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		c.recordError(err)
		backoff := c.backoff()
		c.logger.Warn("tap consumer disconnected",
			slog.Any("err", err),
			slog.Duration("backoff", backoff),
			slog.Int("attempt", c.State().ReconnectAttempt),
		)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *WSConsumer) backoff() time.Duration {
	attempt := c.State().ReconnectAttempt
	if attempt <= 0 {
		// recordError should have incremented attempt to >=1 before
		// backoff() is called, but guard anyway so callers can't panic.
		return time.Second
	}
	// 1s, 2s, 4s, 8s, 16s, 32s... capped at ReconnectMax.
	// A very large attempt would overflow the shift into negative; the
	// <= 0 check below catches that and clamps to ReconnectMax.
	d := time.Second << (attempt - 1)
	if d <= 0 || d > c.cfg.ReconnectMax {
		d = c.cfg.ReconnectMax
	}
	return d
}

// envelope is the outer shape of every frame Tap sends.
type envelope struct {
	ID       uint64          `json:"id"`
	Type     string          `json:"type"`
	Record   *recordPayload  `json:"record,omitempty"`
	Identity json.RawMessage `json:"identity,omitempty"`
}

type recordPayload struct {
	Live       bool            `json:"live"`
	Rev        string          `json:"rev"`
	DID        string          `json:"did"`
	Collection string          `json:"collection"`
	Rkey       string          `json:"rkey"`
	Action     string          `json:"action"`
	CID        string          `json:"cid"`
	Record     json.RawMessage `json:"record,omitempty"`
}

// ackFrame is sent back to Tap after a successful Handle.
//
// Shape confirmed by reading indigo/cmd/tap/types.go (types WsResponse,
// WsResponseAck) and server.go's /channel handler during Task 3.2.
// Tap's server sends outgoing events as raw bytes over TextMessage frames
// containing a MarshallableEvt JSON. The client acks with a WsResponse
// containing {"type": "ack", "id": <id>}.
type ackFrame struct {
	Type string `json:"type"` // always "ack"
	ID   uint64 `json:"id"`
}

// runOnce handles one WS connection lifecycle.
func (c *WSConsumer) runOnce(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, c.cfg.URL, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	// Hard-close on any return. We never have a "normal exit" from the
	// event loop — every path is an error (read/write failure or ctx
	// cancel), so there's no place to call Close(StatusNormalClosure, "")
	// first. CloseNow() skips the close-frame handshake, which is both
	// faster on cancel and avoids a blocked write when the peer is gone.
	defer conn.CloseNow()
	c.setConnected(true)
	c.logger.Info("tap consumer connected", slog.String("url", c.cfg.URL))

	for {
		var env envelope
		if err := wsjson.Read(ctx, conn, &env); err != nil {
			return fmt.Errorf("read: %w", err)
		}
		c.recordEvent()

		switch env.Type {
		case "record":
			if env.Record == nil {
				c.logger.Warn("record envelope missing record field", slog.Uint64("id", env.ID))
				continue
			}
			ev, err := decodeRecordEvent(env)
			if err != nil {
				// Malformed identifiers in the envelope itself: this won't
				// improve on retry, so ack and drop rather than letting
				// Tap redeliver indefinitely.
				c.logger.Error("dropping event with invalid identifier",
					slog.Uint64("id", env.ID),
					slog.Any("err", err),
				)
				if ackErr := c.sendAck(ctx, conn, env.ID); ackErr != nil {
					return fmt.Errorf("ack: %w", ackErr)
				}
				continue
			}
			if err := c.handleWithTimeout(ctx, ev); err != nil {
				c.logger.Error("indexer handle failed",
					slog.String("uri", ev.URI.String()),
					slog.Uint64("id", ev.ID),
					slog.Any("err", err),
				)
				if c.shouldDrop(ev.ID) {
					c.logger.Error("dropping poison-pill event after retries",
						slog.String("uri", ev.URI.String()),
						slog.Uint64("id", ev.ID),
						slog.String("record", string(ev.Record)),
					)
					if err := c.sendAck(ctx, conn, ev.ID); err != nil {
						return fmt.Errorf("ack: %w", err)
					}
					c.forgetRetry(ev.ID)
				}
				continue // do not ack on ordinary error
			}
			c.forgetRetry(ev.ID)
			if err := c.sendAck(ctx, conn, ev.ID); err != nil {
				return fmt.Errorf("ack: %w", err)
			}
		case "identity":
			// Drop identity events at debug.
			c.logger.Debug("tap identity event received", slog.Uint64("id", env.ID))
			if err := c.sendAck(ctx, conn, env.ID); err != nil {
				return fmt.Errorf("ack: %w", err)
			}
		default:
			c.logger.Warn("unknown tap envelope type", slog.String("type", env.Type), slog.Uint64("id", env.ID))
			// Ack anyway to avoid blocking.
			if err := c.sendAck(ctx, conn, env.ID); err != nil {
				return fmt.Errorf("ack: %w", err)
			}
		}
	}
}

func (c *WSConsumer) handleWithTimeout(ctx context.Context, ev Event) error {
	handleCtx, cancel := context.WithTimeout(ctx, c.cfg.AckTimeout)
	defer cancel()
	if err := c.cfg.Indexer.Handle(handleCtx, ev); err != nil {
		c.mu.Lock()
		c.retryCount[ev.ID]++
		c.mu.Unlock()
		return err
	}
	return nil
}

func (c *WSConsumer) shouldDrop(id uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Drop when we have failed MORE THAN MaxRetries times. With
	// MaxRetries=5: first 5 failures are ignored (Tap redelivers), the
	// 6th failure triggers drop+ack.
	return c.retryCount[id] > c.cfg.MaxRetries
}

func (c *WSConsumer) forgetRetry(id uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.retryCount, id)
}

func (c *WSConsumer) sendAck(ctx context.Context, conn *websocket.Conn, id uint64) error {
	return wsjson.Write(ctx, conn, ackFrame{Type: "ack", ID: id})
}

// decodeRecordEvent parses the wire envelope into a typed Event, validating
// DID/NSID/RecordKey at the boundary. CID is a passthrough cast — indigo's
// syntax.CID is documented as an informal helper, not a complete CID
// validator, and tests use short fixture strings that wouldn't pass
// syntax.ParseCID. The downstream guarantee we want is type safety, not
// CID well-formedness; that's the codec's job.
func decodeRecordEvent(env envelope) (Event, error) {
	rec := env.Record
	did, err := syntax.ParseDID(rec.DID)
	if err != nil {
		return Event{}, fmt.Errorf("did %q: %w", rec.DID, err)
	}
	nsid, err := syntax.ParseNSID(rec.Collection)
	if err != nil {
		return Event{}, fmt.Errorf("collection %q: %w", rec.Collection, err)
	}
	rkey, err := syntax.ParseRecordKey(rec.Rkey)
	if err != nil {
		return Event{}, fmt.Errorf("rkey %q: %w", rec.Rkey, err)
	}
	return Event{
		URI:        syntax.ATURI(fmt.Sprintf("at://%s/%s/%s", did, nsid, rkey)),
		CID:        syntax.CID(rec.CID),
		DID:        did,
		Collection: nsid,
		Rkey:       rkey,
		Action:     rec.Action,
		Record:     rec.Record,
		Live:       rec.Live,
		ID:         env.ID,
		Rev:        rec.Rev,
	}, nil
}
