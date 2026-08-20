// Package tap consumes atproto events from a Tap sidecar over WebSocket.
//
// WSConsumer is the production consumer. It dials Tap's /channel WS endpoint,
// durably ingests record, identity, and invalid envelopes, and acknowledges
// only after the ingestor reports a committed terminal outcome. Reconnection
// uses exponential backoff capped at WSConsumerConfig.ReconnectMax. Retryable
// failures remain unacknowledged until Tap redelivers and a durable outcome
// succeeds. Delivery counts are never a correctness boundary.
package tap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
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
	// and passing events to the configured durable ingestor. It always returns
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

// WSConsumerConfig wires a WSConsumer. Ingestor is the only event-processing
// boundary; Logger and Observer are optional.
type WSConsumerConfig struct {
	URL          string // ws://tap:2480/channel
	Ingestor     DurableIngestor
	AckTimeout   time.Duration // per-event durable ingestion deadline
	ReconnectMax time.Duration // cap for exponential reconnect backoff
	Logger       *slog.Logger  // optional; nil → slog.Default()
	Observer     *observability.Observer
}

type IdentityDeletionHandler interface {
	HandleIdentityDeleted(context.Context, syntax.DID) error
}

// WSConsumer connects to Tap's /channel WebSocket and sends ACKs only after a
// durable source, lifecycle, or quarantine outcome has committed.
type WSConsumer struct {
	cfg    WSConsumerConfig
	logger *slog.Logger

	mu    sync.Mutex
	state ConnState
}

var _ Consumer = (*WSConsumer)(nil)

// NewWSConsumer returns a consumer that connects to the given Tap WS URL.
func NewWSConsumer(cfg WSConsumerConfig) *WSConsumer {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &WSConsumer{
		cfg:    cfg,
		logger: logger,
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

const (
	maxTapRecordBytes = 1 << 20
	maxTapRevisionLen = 128
	maxTapCIDLen      = 512
)

// ackFrame is sent back to Tap after a durable acknowledgable outcome.
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
	// coder/websocket defaults to only 32 KiB. AT Protocol records may be
	// larger, while AppView's source contract intentionally caps record JSON
	// at 1 MiB and leaves bounded room for the Tap envelope.
	conn.SetReadLimit(MaxFrameBytes)
	c.setConnected(true)
	c.logger.Info("tap consumer connected",
		slog.String("component", "tap"),
		slog.String("operation", "tap.connect"),
		slog.String("result", "success"))

	for {
		var env envelope
		readCtx, readSpan := c.startTapSpan(ctx, "tap.receive", false)
		_, rawFrame, readErr := conn.Read(readCtx)
		if readErr != nil {
			c.finishTapSpan(readSpan, "error", observability.EventContext{
				"component": "tap",
				"operation": "tap.receive",
				"result":    "error",
			})
			return fmt.Errorf("read: %w", readErr)
		}
		c.finishTapSpan(readSpan, "success", observability.EventContext{
			"component": "tap",
			"operation": "tap.receive",
			"result":    "success",
		})
		if err := json.Unmarshal(rawFrame, &env); err != nil {
			outcome, quarantineErr := c.quarantineInvalid(ctx, InvalidEvent{
				Type: "unknown", Reason: ReasonInvalidEnvelope, Envelope: append(json.RawMessage(nil), rawFrame...),
			})
			if quarantineErr != nil {
				return fmt.Errorf("quarantine malformed Tap envelope: %w", quarantineErr)
			}
			if !outcome.Acknowledgable() {
				return errors.New("quarantine malformed Tap envelope: outcome was not durable")
			}
			// Without a trustworthy Tap id an ACK cannot be constructed. Reconnect
			// so Tap retains and redelivers the event after operator correction.
			return fmt.Errorf("decode Tap envelope: %w", err)
		}
		if env.ID == 0 {
			if err := c.quarantineAndAck(ctx, conn, env, rawFrame, ReasonInvalidEnvelope); err != nil {
				return err
			}
			return errors.New("tap envelope without an acknowledgement id")
		}
		c.recordEvent()
		if c.cfg.Observer != nil {
			c.cfg.Observer.ObserveTapEventReceived(env.Type)
		}

		switch env.Type {
		case "record":
			if env.Record == nil {
				if err := c.quarantineAndAck(ctx, conn, env, rawFrame, ReasonMissingRecord); err != nil {
					return err
				}
				continue
			}
			decodeCtx, decodeSpan := c.startTapSpan(ctx, "tap.decode", false)
			ev, reason, err := decodeRecordEvent(env)
			if err != nil {
				c.finishTapSpan(decodeSpan, "error", observability.EventContext{
					"component": "tap",
					"operation": "tap.decode",
					"result":    "error",
				})
				c.logger.Error("quarantining event with invalid input",
					slog.Uint64("id", env.ID),
					slog.String("result", "invalid"),
					slog.String("error_category", string(reason)),
				)
				if c.cfg.Observer != nil {
					c.cfg.Observer.ObserveIndexerSkipped(env.Record.Collection, "malformed")
				}
				if quarantineErr := c.quarantineAndAck(ctx, conn, env, rawFrame, reason); quarantineErr != nil {
					return quarantineErr
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
			outcome, err := c.ingestRecordWithTimeout(ctx, ev)
			if err != nil || !outcome.Acknowledgable() {
				c.logger.Error("indexer handle failed",
					slog.Uint64("id", ev.ID),
					slog.String("nsid", observability.SafeNSIDLabel(ev.Collection.String())),
					slog.String("result", "error"),
					slog.String("error_category", "indexer"),
				)
				continue // do not ack on ordinary error
			}
			if err := c.sendAck(ctx, conn, ev.ID); err != nil {
				return fmt.Errorf("ack: %w", err)
			}
		case "identity":
			identity, reason, decodeErr := decodeIdentityEvent(env)
			if decodeErr != nil {
				if err := c.quarantineAndAck(ctx, conn, env, rawFrame, reason); err != nil {
					return err
				}
				continue
			}
			outcome, err := c.ingestIdentityWithTimeout(ctx, identity)
			if err != nil || !outcome.Acknowledgable() {
				continue
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
			if c.cfg.Observer != nil {
				c.cfg.Observer.ObserveIndexerSkipped("", "unsupported")
			}
			if err := c.quarantineAndAck(ctx, conn, env, rawFrame, ReasonUnsupportedEventType); err != nil {
				return err
			}
		}
	}
}

func (c *WSConsumer) ingestRecordWithTimeout(ctx context.Context, ev Event) (outcome Outcome, err error) {
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
			if c.cfg.Observer != nil {
				panicCaptured = true
				c.cfg.Observer.CapturePanic(handleCtx, observability.EventContext{
					"component": "tap_indexer",
					"nsid":      observability.SafeNSIDLabel(ev.Collection.String()),
					"result":    "error",
				}, recovered)
			}
			outcome = Retryable(ReasonProjectionFailure)
			err = fmt.Errorf("ingestion panic: %T", recovered)
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
	if c.cfg.Ingestor == nil {
		return Retryable(ReasonDurableIngestorRequired), errors.New("tap: durable ingestor is required")
	}
	return c.cfg.Ingestor.IngestRecord(handleCtx, ev)
}

func (c *WSConsumer) ingestIdentityWithTimeout(ctx context.Context, event IdentityEvent) (outcome Outcome, err error) {
	callCtx, cancel := context.WithTimeout(ctx, c.cfg.AckTimeout)
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			outcome = Retryable(ReasonProjectionFailure)
			err = fmt.Errorf("identity ingestion panic: %T", recovered)
		}
	}()
	if c.cfg.Ingestor == nil {
		return Retryable(ReasonDurableIngestorRequired), errors.New("tap: durable ingestor is required")
	}
	return c.cfg.Ingestor.IngestIdentity(callCtx, event)
}

func (c *WSConsumer) quarantineAndAck(ctx context.Context, conn *websocket.Conn, env envelope, rawFrame []byte, reason ReasonCode) error {
	outcome, ingestErr := c.quarantineInvalid(ctx, InvalidEvent{
		ID: env.ID, Type: env.Type, Reason: reason, Envelope: append(json.RawMessage(nil), rawFrame...),
	})
	if ingestErr != nil {
		return fmt.Errorf("persist Tap quarantine: %w", ingestErr)
	}
	if !outcome.Acknowledgable() {
		return errors.New("persist Tap quarantine: outcome was not durable")
	}
	if env.ID == 0 {
		return errors.New("persisted Tap quarantine but envelope has no acknowledgement id")
	}
	if err := c.sendAck(ctx, conn, env.ID); err != nil {
		return fmt.Errorf("ack: %w", err)
	}
	return nil
}

func (c *WSConsumer) quarantineInvalid(ctx context.Context, invalid InvalidEvent) (outcome Outcome, err error) {
	callCtx, cancel := context.WithTimeout(ctx, c.cfg.AckTimeout)
	defer cancel()
	if c.cfg.Ingestor == nil {
		return Retryable(ReasonDurableIngestorRequired), errors.New("tap: durable ingestor is required")
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			outcome = Retryable(ReasonStorageUnavailable)
			err = fmt.Errorf("quarantine ingestion panic: %T", recovered)
		}
	}()
	return c.cfg.Ingestor.Quarantine(callCtx, invalid)
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
func decodeRecordEvent(env envelope) (Event, ReasonCode, error) {
	rec := env.Record
	if rec.Rev == "" || rec.Rev != strings.TrimSpace(rec.Rev) || len(rec.Rev) > maxTapRevisionLen {
		return Event{}, ReasonInvalidEnvelope, errors.New("repository revision is missing or invalid")
	}
	did, err := syntax.ParseDID(rec.DID)
	if err != nil {
		return Event{}, ReasonInvalidDID, fmt.Errorf("did %q: %w", rec.DID, err)
	}
	nsid, err := syntax.ParseNSID(rec.Collection)
	if err != nil {
		return Event{}, ReasonInvalidCollection, fmt.Errorf("collection %q: %w", rec.Collection, err)
	}
	rkey, err := syntax.ParseRecordKey(rec.Rkey)
	if err != nil {
		return Event{}, ReasonInvalidRecordKey, fmt.Errorf("rkey %q: %w", rec.Rkey, err)
	}
	switch rec.Action {
	case "create", "update":
		if rec.CID == "" || rec.CID != strings.TrimSpace(rec.CID) || len(rec.CID) > maxTapCIDLen {
			return Event{}, ReasonMalformedRecord, errors.New("record CID is missing or invalid")
		}
		if len(rec.Record) == 0 || !json.Valid(rec.Record) {
			return Event{}, ReasonMalformedRecord, errors.New("record body is missing or malformed")
		}
		if len(rec.Record) > maxTapRecordBytes {
			return Event{}, ReasonRecordTooLarge, errors.New("record body exceeds the durable source limit")
		}
	case "delete":
	default:
		return Event{}, ReasonUnsupportedAction, fmt.Errorf("unsupported record action %q", rec.Action)
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
	}, ReasonNone, nil
}

func decodeIdentityEvent(env envelope) (IdentityEvent, ReasonCode, error) {
	var raw struct {
		DID      string `json:"did"`
		Handle   string `json:"handle"`
		IsActive bool   `json:"is_active"`
		Status   string `json:"status"`
	}
	if len(env.Identity) == 0 || json.Unmarshal(env.Identity, &raw) != nil {
		return IdentityEvent{}, ReasonInvalidIdentity, errors.New("identity body is missing or malformed")
	}
	did, err := syntax.ParseDID(raw.DID)
	if err != nil {
		return IdentityEvent{}, ReasonInvalidDID, fmt.Errorf("identity DID %q: %w", raw.DID, err)
	}
	switch raw.Status {
	case "active", "takendown", "suspended", "deactivated", "deleted":
	default:
		return IdentityEvent{}, ReasonUnsupportedIdentityStatus, fmt.Errorf("unsupported identity status %q", raw.Status)
	}
	return IdentityEvent{
		ID: env.ID, DID: did, Handle: raw.Handle, IsActive: raw.IsActive, Status: raw.Status,
	}, ReasonNone, nil
}
