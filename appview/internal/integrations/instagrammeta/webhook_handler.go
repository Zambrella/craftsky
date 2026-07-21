package instagrammeta

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"
)

type WebhookWorkSink interface {
	EnqueueWebhookWork(ctx context.Context, items []WorkItem, now time.Time) (int, error)
}

// WebhookInvalidRedemptionLimiter is an opaque, request-scoped limiter for the
// trusted source IP of one signed webhook delivery. Implementations must keep
// the source IP out of the returned value's observable representation.
type WebhookInvalidRedemptionLimiter interface {
	AllowInvalidRedemption(context.Context) (WebhookLimitDecision, error)
}

// GuardedWebhookWorkSink classifies newly inserted work while it still owns
// the delivery transaction. This lets duplicate and currently redeemable
// challenges bypass the invalid-redemption bucket without creating a race
// between classification and durable persistence.
type GuardedWebhookWorkSink interface {
	EnqueueWebhookWorkGuarded(context.Context, []WorkItem, time.Time, WebhookInvalidRedemptionLimiter) (int, error)
}

type WebhookLimitDecision struct {
	Allowed    bool
	RetryAfter time.Duration
}

// WebhookRequestLimiter keeps ingress policy outside the Meta adapter while
// fixing the security-sensitive call order inside the exact-byte handler.
type WebhookRequestLimiter interface {
	AllowSourceIP(context.Context, *http.Request) (WebhookLimitDecision, error)
	AllowGlobal(context.Context) (WebhookLimitDecision, error)
	InvalidRedemptionSourceIP(*http.Request) (WebhookInvalidRedemptionLimiter, error)
}

type WebhookHandlerConfig struct {
	AppSecret      []byte                `json:"-"`
	VerifyToken    string                `json:"-"`
	Reducer        *PayloadReducer       `json:"-"`
	Sink           WebhookWorkSink       `json:"-"`
	Limiter        WebhookRequestLimiter `json:"-"`
	BodyLimitBytes int64                 `json:"-"`
	MaxEvents      int                   `json:"-"`
	Now            func() time.Time      `json:"-"`
}

func (WebhookHandlerConfig) String() string {
	return "Instagram webhook handler config [REDACTED]"
}

func (WebhookHandlerConfig) GoString() string {
	return "Instagram webhook handler config [REDACTED]"
}

type WebhookHandler struct {
	appSecret   []byte
	verifyToken string
	reducer     *PayloadReducer
	sink        WebhookWorkSink
	limiter     WebhookRequestLimiter
	bodyLimit   int64
	maxEvents   int
	now         func() time.Time
}

func NewWebhookHandler(config WebhookHandlerConfig) (*WebhookHandler, error) {
	if len(config.AppSecret) == 0 || config.VerifyToken == "" || config.Reducer == nil || config.Sink == nil {
		return nil, errors.New("Instagram webhook handler configuration is incomplete")
	}
	if config.Limiter != nil {
		if _, ok := config.Sink.(GuardedWebhookWorkSink); !ok {
			return nil, errors.New("Instagram webhook guarded work sink is required with rate limiting")
		}
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.BodyLimitBytes == 0 {
		config.BodyLimitBytes = MaxWebhookBodyBytes
	}
	if config.BodyLimitBytes < 1 || config.BodyLimitBytes > MaxWebhookBodyBytes {
		return nil, errors.New("Instagram webhook body limit is invalid")
	}
	if config.MaxEvents == 0 {
		config.MaxEvents = MaxSupportedEvents
	}
	if config.MaxEvents < 1 || config.MaxEvents > MaxSupportedEvents {
		return nil, errors.New("Instagram webhook event limit is invalid")
	}
	return &WebhookHandler{
		appSecret:   append([]byte(nil), config.AppSecret...),
		verifyToken: config.VerifyToken,
		reducer:     config.Reducer,
		sink:        config.Sink,
		limiter:     config.Limiter,
		bodyLimit:   config.BodyLimitBytes,
		maxEvents:   config.MaxEvents,
		now:         config.Now,
	}, nil
}

func (*WebhookHandler) String() string {
	return "Instagram webhook handler [REDACTED]"
}

func (*WebhookHandler) GoString() string {
	return "Instagram webhook handler [REDACTED]"
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.serveVerification(w, r)
	case http.MethodPost:
		h.serveDelivery(w, r)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		writeGenericWebhookError(w, http.StatusMethodNotAllowed)
	}
}

func (h *WebhookHandler) serveVerification(w http.ResponseWriter, r *http.Request) {
	challenge, ok := VerifyCallbackQuery(r.URL.Query(), h.verifyToken)
	if !ok {
		writeGenericWebhookError(w, http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, challenge)
}

func (h *WebhookHandler) serveDelivery(w http.ResponseWriter, r *http.Request) {
	if h.limiter != nil {
		decision, err := h.limiter.AllowSourceIP(r.Context(), r)
		if err != nil {
			writeGenericWebhookError(w, http.StatusServiceUnavailable)
			return
		}
		if !decision.Allowed {
			writeWebhookRateLimited(w, decision.RetryAfter)
			return
		}
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, h.bodyLimit+1))
	if err != nil {
		writeGenericWebhookError(w, http.StatusBadRequest)
		return
	}
	if int64(len(body)) > h.bodyLimit {
		writeGenericWebhookError(w, http.StatusRequestEntityTooLarge)
		return
	}
	if err := VerifySignatureValues(h.appSecret, body, r.Header.Values("X-Hub-Signature-256")); err != nil {
		writeGenericWebhookError(w, http.StatusUnauthorized)
		return
	}
	if h.limiter != nil {
		decision, err := h.limiter.AllowGlobal(r.Context())
		if err != nil {
			writeGenericWebhookError(w, http.StatusServiceUnavailable)
			return
		}
		if !decision.Allowed {
			writeWebhookRateLimited(w, decision.RetryAfter)
			return
		}
	}
	items, err := h.reducer.Reduce(body)
	if err != nil {
		switch {
		case errors.Is(err, ErrPayloadTooLarge), errors.Is(err, ErrTooManyMessageEvents):
			writeGenericWebhookError(w, http.StatusRequestEntityTooLarge)
		case errors.Is(err, ErrInvalidPayload):
			writeGenericWebhookError(w, http.StatusBadRequest)
		default:
			writeGenericWebhookError(w, http.StatusInternalServerError)
		}
		return
	}
	if len(items) > h.maxEvents {
		writeGenericWebhookError(w, http.StatusRequestEntityTooLarge)
		return
	}
	now := h.now().UTC()
	if now.IsZero() {
		writeGenericWebhookError(w, http.StatusInternalServerError)
		return
	}
	if h.limiter != nil && len(items) > 0 {
		invalidLimiter, err := h.limiter.InvalidRedemptionSourceIP(r)
		if err != nil {
			writeGenericWebhookError(w, http.StatusServiceUnavailable)
			return
		}
		guardedSink := h.sink.(GuardedWebhookWorkSink)
		if _, err := guardedSink.EnqueueWebhookWorkGuarded(r.Context(), items, now, invalidLimiter); err != nil {
			writeGenericWebhookError(w, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
	if _, err := h.sink.EnqueueWebhookWork(r.Context(), items, now); err != nil {
		writeGenericWebhookError(w, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func writeGenericWebhookError(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, http.StatusText(status)+"\n")
}

func writeWebhookRateLimited(w http.ResponseWriter, retryAfter time.Duration) {
	seconds := int((retryAfter + time.Second - 1) / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	if seconds > 60 {
		seconds = 60
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	writeGenericWebhookError(w, http.StatusTooManyRequests)
}
