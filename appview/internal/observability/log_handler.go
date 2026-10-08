package observability

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
)

// NewDiagnosticHandler applies policy before direct/inherited slog records
// reach local output. Developer text is bounded/scrubbed; unknown objects are excluded.
func NewDiagnosticHandler(local slog.Handler) slog.Handler {
	return newDiagnosticHandler(local, time.Now)
}
func newDiagnosticHandler(local slog.Handler, clock func() time.Time) slog.Handler {
	if clock == nil {
		clock = time.Now
	}
	if _, ok := local.(*diagnosticHandler); ok {
		return local
	}
	return &diagnosticHandler{local: local, export: &diagnosticExport{}, retries: &retryLogLimiter{now: clock, scopes: map[string]retryLogState{}}}
}

var logVersionPattern = regexp.MustCompile(`^(dev|(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))$`)

type diagnosticExport struct {
	mu       sync.RWMutex
	sink     LogSink
	metadata EventContext
}

func (e *diagnosticExport) bind(sink LogSink, metadata EventContext) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.sink = sink
	e.metadata = SanitizeEventContext(metadata)
}
func (e *diagnosticExport) current() LogSink { e.mu.RLock(); defer e.mu.RUnlock(); return e.sink }

func (e *diagnosticExport) processMetadata() EventContext {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.metadata
}

type diagnosticHandler struct {
	retries *retryLogLimiter
	export  *diagnosticExport
	local   slog.Handler
	attrs   []slog.Attr
	group   string
}

func (h *diagnosticHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.local.Enabled(ctx, level)
}
func (h *diagnosticHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	copy := *h
	copy.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &copy
}
func (h *diagnosticHandler) WithGroup(name string) slog.Handler {
	copy := *h
	copy.group = name
	return &copy
}
func (h *diagnosticHandler) Handle(ctx context.Context, raw slog.Record) error {
	message := boundDiagnosticText(raw.Message, 512)
	record := slog.NewRecord(raw.Time, raw.Level, message, 0)
	attrs := append([]slog.Attr{}, h.attrs...)
	raw.Attrs(func(attr slog.Attr) bool { attrs = append(attrs, attr); return true })
	for key, value := range h.export.processMetadata() {
		attrs = append(attrs, slog.Any(key, value))
	}
	for key, value := range withDiagnosticCorrelation(ctx, nil) {
		attrs = append(attrs, slog.Any(key, value))
	}
	safe := make([]slog.Attr, 0, len(attrs))
	omitted := false
	for _, attr := range attrs {
		if selected, ok := selectSlogAttr(ctx, attr); ok {
			safe = append(safe, selected)
		}
	}
	// Core operation/correlation/cause fields win over optional excerpts.
	sort.SliceStable(safe, func(i, j int) bool { return logFieldPriority(safe[i].Key) < logFieldPriority(safe[j].Key) })
	size := 512 + len(message)
	for _, attr := range safe {
		encoded, err := json.Marshal(attr.Value.Any())
		if err != nil {
			continue
		}
		if size+len(encoded)+len(attr.Key)+8 > 15000 || record.NumAttrs() >= 31 {
			omitted = true
			continue
		}
		size += len(encoded) + len(attr.Key) + 8
		record.AddAttrs(attr)
	}
	if omitted {
		record.AddAttrs(slog.String("omitted", "[OMITTED: log field budget exceeded]"))
	}
	localOnly, _ := ctx.Value(localDiagnosticFallbackKey{}).(bool)
	emit, suppressed := true, 0
	if !localOnly {
		emit, suppressed = h.retries.selectRecord(record)
	}
	if !emit {
		return nil
	}
	if suppressed > 0 {
		record.AddAttrs(slog.Int("suppressed_count", suppressed))
	}
	err := handleProtectedLocal(ctx, h.local, record)
	if sink := h.export.current(); !localOnly && sink != nil && shouldExportDiagnosticLog(record) {
		fields := EventContext{}
		record.Attrs(func(attr slog.Attr) bool { fields[attr.Key] = attr.Value.Any(); return true })
		if !emitProtectedLog(ctx, sink, record.Level, record.Message, fields) {
			fallback := slog.NewRecord(time.Now(), slog.LevelWarn, "telemetry export failed", 0)
			record.Attrs(func(attr slog.Attr) bool { fallback.AddAttrs(attr); return true })
			// Already selected and bounded; bypass export and retry admission.
			_ = handleProtectedLocal(ctx, h.local, fallback)
		}
	}
	return err
}
func logFieldPriority(key string) int {
	switch key {
	case "run_id", "request_id", "operation", "component":
		return 0
	case "causes", "panic":
		return 1
	case "diagnostic":
		return 3
	default:
		return 2
	}
}
func selectSlogAttr(ctx context.Context, attr slog.Attr) (slog.Attr, bool) {
	// Do not resolve arbitrary LogValuer/Stringer implementations.
	value := attr.Value.Any()
	switch attr.Key {
	case "app_version":
		if version, ok := value.(string); ok && logVersionPattern.MatchString(version) {
			return slog.String(attr.Key, version), true
		}
	case "addr":
		if address, ok := value.(string); ok {
			host, port, err := net.SplitHostPort(address)
			n, portErr := strconv.ParseUint(port, 10, 16)
			if err == nil && portErr == nil && n > 0 && (net.ParseIP(host) != nil || host == "localhost" || host == "") {
				return slog.String(attr.Key, address), true
			}
		}
	case "id":
		if id, ok := value.(uint64); ok && id > 0 {
			return slog.Uint64(attr.Key, id), true
		}
	case "action":
		if action, ok := value.(string); ok && (action == "create" || action == "update" || action == "delete") {
			return slog.String(attr.Key, action), true
		}
	case "recordBytes":
		if size, ok := value.(int64); ok && size >= 0 {
			return slog.Int64(attr.Key, size), true
		}
	case "error", "err":
		if err, ok := value.(error); ok {
			return slog.Any("causes", DescribeError(err, EventContext{})), true
		}
	case "causes":
		if causes, ok := value.([]DiagnosticCause); ok {
			if len(causes) > 8 {
				causes = causes[:8]
			}
			selected := append([]DiagnosticCause{}, causes...)
			for i := range selected {
				selected[i].Type = safeDiagnosticType(selected[i].Type)
				selected[i].Message = boundDiagnosticText(selected[i].Message, 512)
			}
			return slog.Any(attr.Key, selected), true
		}
	case "panic":
		if panicRecord, ok := value.(panicLogRecord); ok {
			for i := range panicRecord.Exceptions {
				protectExceptionFrames(&panicRecord.Exceptions[i])
			}
			return slog.Any(attr.Key, panicRecord.Exceptions), true
		}
	case "diagnostic":
		if selected, ok := value.(workflowLogValue); ok {
			value = selected.workflow
		}
		if workflow, ok := value.(WorkflowContext); ok {
			if selected := workflowForRequest(ctx, workflow); selected != nil {
				return slog.Any(attr.Key, boundSelectedFields(selected.diagnosticFields())), true
			}
		}
	case "incoming_path":
		if request, ok := ctx.Value(requestDiagnosticContextKey{}).(RequestDiagnosticContext); ok {
			return slog.String(attr.Key, request.Path), true
		}
	case "method":
		if method, ok := value.(string); ok {
			return slog.String(attr.Key, safeHTTPMethod(method)), true
		}
	case "status":
		if status, ok := value.(int64); ok {
			return slog.Int64(attr.Key, status), true
		}
	case "bytes", "content_length", "attempt", "count":
		if n, ok := value.(int64); ok {
			return slog.Int64(attr.Key, n), true
		}
	case "duration":
		if d, ok := value.(time.Duration); ok {
			return slog.Duration(attr.Key, d), true
		}
	case "stage":
		if stage, ok := value.(string); ok {
			return slog.String("failure_stage", safeMetricStage(stage)), true
		}
	}
	if _, allowed := allowedEventContextKeys[attr.Key]; allowed {
		switch value.(type) {
		case string, bool, int, int64, float64:
			selected := SanitizeEventContext(EventContext{attr.Key: value})
			if value, ok := selected[attr.Key]; ok {
				return slog.Any(attr.Key, value), true
			}
		}
	}
	return slog.Attr{}, false
}

type panicLogRecord struct{ Exceptions []sentry.Exception }

func shouldExportDiagnosticLog(record slog.Record) bool {
	if record.Level >= slog.LevelWarn {
		return true
	}
	if record.Level != slog.LevelInfo {
		return false
	}
	switch record.Message {
	case "moderation operation completed", "Request received", "Request completed", "post create: response ready", "post delete: PDS record deleted", "profile put: writes succeeded", "RevenueCat reconciliation completed", "Subscription assignment completed", "follower growth capture completed", "account deletion completed", "manual scheduled publication completed":
		return true
	default:
		return false
	}
}
func emitProtectedLog(ctx context.Context, sink LogSink, level slog.Level, message string, fields EventContext) (success bool) {
	defer func() {
		if recover() != nil {
			success = false
		}
	}()
	sink.Emit(ctx, level, message, fields)
	return true
}

type localDiagnosticFallbackKey struct{}

func (o *Observer) localFailure(ctx context.Context, message string, fields EventContext, causes []DiagnosticCause) {
	defer func() { _ = recover() }()
	if o == nil || o.logger == nil {
		return
	}
	attrs := eventContextSlogAttrs(SanitizeEventContext(withDiagnosticCorrelation(ctx, fields)))
	if len(causes) > 0 {
		attrs = append(attrs, slog.Any("causes", causes))
	}
	o.logger.WarnContext(context.WithValue(ctx, localDiagnosticFallbackKey{}, true), message, attrs...)
}

func handleProtectedLocal(ctx context.Context, handler slog.Handler, record slog.Record) (err error) {
	defer func() {
		if recover() != nil {
			err = nil
		}
	}()
	return handler.Handle(ctx, record)
}
