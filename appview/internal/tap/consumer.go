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

	"social.craftsky/appview/internal/observability"
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
	URL             string // ws://tap:2480/channel
	Indexer         HandlerIndexer
	AckTimeout      time.Duration // per-event Handle deadline
	ReconnectMax    time.Duration // cap for exponential reconnect backoff
	MaxRetries      int           // poison-pill threshold per event id
	Logger          *slog.Logger  // optional; nil → slog.Default()
	Observer        *observability.Observer
	IdentityHandler IdentityDeletionHandler
}

type IdentityDeletionHandler interface {
	HandleIdentityDeleted(context.Context, syntax.DID) error
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
	if c.cfg.Observer != nil {
		c.cfg.Observer.SetTapConnected(connected)
	}
	if connected {
		c.state.LastError = ""
		c.state.ReconnectAttempt = 0
	}
}

func (c *WSConsumer) recordError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Connected = false
	c.state.LastError = "connection_error"
	c.state.ReconnectAttempt++
	if c.cfg.Observer != nil {
		c.cfg.Observer.SetTapConnected(false)
		c.cfg.Observer.ObserveTapReconnect()
	}
}

func (c *WSConsumer) recordEvent() {
	now := time.Now().UTC()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.LastEventAt = now
	if c.cfg.Observer != nil {
		c.cfg.Observer.ObserveTapLastEventAt(now)
	}
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
			slog.String("result", "error"),
			slog.String("error_category", "connection"),
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
func (c *WSConsumer) runOnce(ctx context.Context) (err error) {
	var consumeSpan *observability.Span
	if c.cfg.Observer != nil {
		ctx, consumeSpan = c.cfg.Observer.StartSpan(ctx, observability.SpanContext{
			Operation: "tap.consume",
			Component: "tap",
			Attributes: observability.EventContext{
				"component": "tap",
			},
		})
		defer func() {
			result := "error"
			if err == nil || ctx.Err() != nil {
				result = "success"
			}
			consumeSpan.SetAttributes(observability.EventContext{
				"component": "tap",
				"result":    result,
			})
			consumeSpan.Finish(result)
		}()
	}
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
	c.logger.Info("tap consumer connected",
		slog.String("component", "tap"),
		slog.String("operation", "tap.connect"),
		slog.String("result", "success"))

	for {
		var env envelope
		readCtx, readSpan := c.startTapSpan(ctx, "tap.receive", false)
		if err := wsjson.Read(readCtx, conn, &env); err != nil {
			c.finishTapSpan(readSpan, "error", observability.EventContext{
				"component": "tap",
				"operation": "tap.receive",
				"result":    "error",
			})
			return fmt.Errorf("read: %w", err)
		}
		c.finishTapSpan(readSpan, "success", observability.EventContext{
			"component": "tap",
			"operation": "tap.receive",
			"result":    "success",
		})
		c.recordEvent()
		if c.cfg.Observer != nil {
			c.cfg.Observer.ObserveTapEventReceived(env.Type)
		}

		switch env.Type {
		case "record":
			if env.Record == nil {
				c.logger.Warn("record envelope missing record field", slog.Uint64("id", env.ID))
				if c.cfg.Observer != nil {
					c.cfg.Observer.ObserveIndexerSkipped("", "malformed")
				}
				continue
			}
			decodeCtx, decodeSpan := c.startTapSpan(ctx, "tap.decode", false)
			ev, err := decodeRecordEvent(env)
			if err != nil {
				c.finishTapSpan(decodeSpan, "error", observability.EventContext{
					"component": "tap",
					"operation": "tap.decode",
					"result":    "error",
				})
				// Malformed identifiers in the envelope itself: this won't
				// improve on retry, so ack and drop rather than letting
				// Tap redeliver indefinitely.
				c.logger.Error("dropping event with invalid identifier",
					slog.Uint64("id", env.ID),
					slog.String("result", "dropped"),
					slog.String("error_category", "malformed_identifier"),
				)
				if c.cfg.Observer != nil {
					c.cfg.Observer.ObserveIndexerSkipped(env.Record.Collection, "malformed")
				}
				if ackErr := c.sendAck(ctx, conn, env.ID); ackErr != nil {
					return fmt.Errorf("ack: %w", ackErr)
				}
				continue
			}
			_ = decodeCtx
			c.finishTapSpan(decodeSpan, "success", observability.EventContext{
				"component": "tap",
				"operation": "tap.decode",
				"nsid":      observability.SafeNSIDLabel(ev.Collection.String()),
				"result":    "success",
			})
			c.logger.Debug("tap record event received",
				slog.Uint64("id", ev.ID),
				slog.String("action", ev.Action),
				slog.String("nsid", observability.SafeNSIDLabel(ev.Collection.String())),
				slog.Int("recordBytes", len(ev.Record)),
			)
			if err := c.handleWithTimeout(ctx, ev); err != nil {
				c.logger.Error("indexer handle failed",
					slog.Uint64("id", ev.ID),
					slog.String("nsid", observability.SafeNSIDLabel(ev.Collection.String())),
					slog.String("result", "error"),
					slog.String("error_category", "indexer"),
				)
				if c.shouldDrop(ev.ID) {
					c.logger.Error("dropping poison-pill event after retries",
						slog.Uint64("id", ev.ID),
						slog.String("nsid", observability.SafeNSIDLabel(ev.Collection.String())),
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
			var identity struct {
				DID    string `json:"did"`
				Status string `json:"status"`
			}
			if err := json.Unmarshal(env.Identity, &identity); err != nil {
				continue
			}
			if identity.Status == "deleted" && c.cfg.IdentityHandler != nil {
				did, err := syntax.ParseDID(identity.DID)
				if err != nil {
					continue
				}
				if err := c.cfg.IdentityHandler.HandleIdentityDeleted(ctx, did); err != nil {
					continue
				}
			}
			c.logger.Debug("tap identity event received", slog.Uint64("id", env.ID))
			if c.cfg.Observer != nil {
				c.cfg.Observer.ObserveIndexerSkipped("", "identity")
			}
			if err := c.sendAck(ctx, conn, env.ID); err != nil {
				return fmt.Errorf("ack: %w", err)
			}
		default:
			c.logger.Warn("unknown tap envelope type", slog.String("type", env.Type), slog.Uint64("id", env.ID))
			// Ack anyway to avoid blocking.
			if c.cfg.Observer != nil {
				c.cfg.Observer.ObserveIndexerSkipped("", "unsupported")
			}
			if err := c.sendAck(ctx, conn, env.ID); err != nil {
				return fmt.Errorf("ack: %w", err)
			}
		}
	}
}

func (c *WSConsumer) handleWithTimeout(ctx context.Context, ev Event) (err error) {
	started := time.Now()
	handleCtx, cancel := context.WithTimeout(ctx, c.cfg.AckTimeout)
	defer cancel()
	var indexerSpan *observability.Span
	if c.cfg.Observer != nil {
		handleCtx, indexerSpan = c.cfg.Observer.StartTapSpan(handleCtx, "tap.indexer.handle", false)
		indexerSpan.SetAttributes(observability.EventContext{
			"component": "tap_indexer",
			"operation": "tap.indexer.handle",
			"nsid":      observability.SafeNSIDLabel(ev.Collection.String()),
		})
	}
	defer func() {
		panicCaptured := false
		if recovered := recover(); recovered != nil {
			c.mu.Lock()
			c.retryCount[ev.ID]++
			c.mu.Unlock()
			if c.cfg.Observer != nil {
				panicCaptured = true
				c.cfg.Observer.CapturePanic(handleCtx, observability.EventContext{
					"component": "tap_indexer",
					"nsid":      observability.SafeNSIDLabel(ev.Collection.String()),
					"result":    "error",
				}, recovered)
			}
			err = fmt.Errorf("indexer panic: %T", recovered)
		}
		if c.cfg.Observer != nil {
			c.cfg.Observer.ObserveIndexerHandled(ev.Collection.String(), err, time.Since(started))
			if err != nil && !panicCaptured {
				c.cfg.Observer.CaptureError(handleCtx, observability.EventContext{
					"component":      "tap_indexer",
					"nsid":           observability.SafeNSIDLabel(ev.Collection.String()),
					"result":         "error",
					"error_category": "unexpected",
				}, err)
			}
		}
		if indexerSpan != nil {
			result := "success"
			if err != nil {
				result = "error"
			}
			indexerSpan.SetAttributes(observability.EventContext{
				"component": "tap_indexer",
				"operation": "tap.indexer.handle",
				"nsid":      observability.SafeNSIDLabel(ev.Collection.String()),
				"result":    result,
			})
			indexerSpan.Finish(result)
		}
	}()
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
	ackCtx, ackSpan := c.startTapSpan(ctx, "tap.ack", false)
	err := wsjson.Write(ackCtx, conn, ackFrame{Type: "ack", ID: id})
	if c.cfg.Observer != nil {
		c.cfg.Observer.ObserveTapEventAcknowledged(err)
	}
	result := "success"
	if err != nil {
		result = "error"
	}
	c.finishTapSpan(ackSpan, result, observability.EventContext{
		"component": "tap",
		"operation": "tap.ack",
		"result":    result,
	})
	return err
}

func (c *WSConsumer) startTapSpan(ctx context.Context, operation string, force bool) (context.Context, *observability.Span) {
	if c.cfg.Observer == nil {
		return ctx, &observability.Span{}
	}
	return c.cfg.Observer.StartTapSpan(ctx, operation, force)
}

func (c *WSConsumer) finishTapSpan(span *observability.Span, result string, attrs observability.EventContext) {
	if span == nil || !span.Enabled() {
		return
	}
	span.SetAttributes(attrs)
	span.Finish(result)
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
