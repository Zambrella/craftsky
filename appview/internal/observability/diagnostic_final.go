package observability

import (
	"encoding/json"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/attribute"
	"path/filepath"
	"regexp"
	"strings"
)

var diagnosticHTTPStatusPattern = regexp.MustCompile(`^[1-5][0-9]{2}$`)

var diagnosticTechnicalPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,160}$`)

// Final SDK hooks strip private enrichment while retaining SDK exceptions,
// stacks and standard deployment/runtime metadata.
func protectSDKEvent(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	event.Request = nil
	event.User = sentry.User{}
	event.ServerName = ""
	event.Attachments = nil
	// Exception stacks retain useful frames; thread payloads can include locals
	// and source outside that protected exception path.
	event.Threads = nil
	event.Breadcrumbs = nil
	event.Message = ""
	tags := map[string]string{}
	for key, value := range SanitizeEventContext(eventContextFromTags(event.Tags)) {
		if !correlationFieldNames[key] {
			tags[key] = fmt.Sprint(value)
		}
	}
	event.Tags = tags
	for key, fields := range event.Contexts {
		switch key {
		case "diagnostic":
			event.Contexts[key] = sentry.Context(selectWorkflowFields(fields))
		case "correlation":
			event.Contexts[key] = sentry.Context(SanitizeEventContext(fields))
		case "trace":
			delete(fields, "data")
			delete(fields, "description")
		case "app", "device", "os", "runtime", "browser":
			// Standard SDK metadata is retained; user/request identity is excluded above.
		default:
			delete(event.Contexts, key)
		}
	}
	var causes []sentry.Exception
	if hint != nil && hint.OriginalException != nil {
		causes = diagnosticExceptions(DescribeError(hint.OriginalException, eventContextFromTags(event.Tags)))
	}
	for i := range event.Exception {
		exception := &event.Exception[i]
		exception.Value = "operation failed"
		if recoveredType, ok := event.Tags["recovered_type"]; ok && recoveredType == exception.Type {
			exception.Value = "panic recovered"
		}
		if i < len(causes) && causes[i].Type == exception.Type {
			exception.Value = causes[i].Value
		}
		protectExceptionFrames(exception)
	}
	return event
}
func eventContextFromTags(tags map[string]string) EventContext {
	fields := EventContext{}
	for key, value := range tags {
		fields[key] = value
	}
	return fields
}
func selectWorkflowFields(fields EventContext) EventContext {
	selected := EventContext{}
	for _, key := range []string{"actor_did", "target_did", "record_uri", "handle", "cid", "nsid", "record_key", "attempted_did", "identifier_valid", "validation_reason", "public_excerpt", "operation_account_did", "workflow_ref", "event_id", "outcome", "acknowledgement", "reason"} {
		if value, ok := fields[key]; ok {
			selected[key] = value
		}
	}
	return boundSelectedFields(selected)
}

func safeDiagnosticType(value string) string {
	if len(value) > 2048 || strings.ContainsAny(value, "\n\r\x00") {
		return "[OMITTED]"
	}
	return boundDiagnosticText(value, 256)
}
func boundSelectedFields(fields EventContext) EventContext {
	out := EventContext{}
	for key, value := range fields {
		if len(out) >= 32 {
			out["omitted"] = "[OMITTED: excess fields]"
			break
		}
		switch v := value.(type) {
		case string:
			if len(v) > MaxDiagnosticTextBytes && key != "public_excerpt" {
				out[key] = "[OMITTED: oversized identifier]"
			} else {
				out[key] = sanitizeKnownDiagnosticText(v)
			}
		case bool, int, int64, float64:
			out[key] = v
		}
	}
	return out
}
func protectSDKLog(log *sentry.Log) *sentry.Log {
	log.Body = boundDiagnosticText(log.Body, 512)
	selected := map[string]attribute.Value{}
	for key, value := range log.Attributes {
		if _, ok := allowedEventContextKeys[key]; ok {
			if v, ok := SanitizeEventContext(EventContext{key: value.AsInterface()})[key]; ok {
				switch safe := v.(type) {
				case bool:
					selected[key] = attribute.BoolValue(safe)
				case int:
					selected[key] = attribute.IntValue(safe)
				case int64:
					selected[key] = attribute.Int64Value(safe)
				case float64:
					selected[key] = attribute.Float64Value(safe)
				default:
					selected[key] = attribute.StringValue(fmt.Sprint(safe))
				}
			}
			continue
		}
		switch key {
		case "diagnostic":
			var fields EventContext
			if json.Unmarshal([]byte(value.AsString()), &fields) == nil {
				if data, err := json.Marshal(selectWorkflowFields(fields)); err == nil {
					selected[key] = attribute.StringValue(string(data))
				}
			}
		case "causes":
			var causes []DiagnosticCause
			if json.Unmarshal([]byte(value.AsString()), &causes) != nil {
				continue
			}
			if len(causes) > 8 {
				causes = causes[:8]
			}
			for i := range causes {
				cause := &causes[i]
				cause.Type = safeDiagnosticType(cause.Type)
				cause.Message = "operation failed"
				switch {
				case diagnosticHTTPStatusPattern.MatchString(cause.Code):
					prefix := "Upstream"
					switch cause.Type {
					case "*http.ResponseError":
						prefix = "Storage"
					case "*atclient.APIError":
						prefix = "PDS"
					}
					cause.Message = prefix + " request failed (HTTP " + cause.Code + ")"
				case sqlStatePattern.MatchString(cause.Code):
					cause.Message = "database error (SQLSTATE " + cause.Code + ")"
				default:
					cause.Code = ""
				}
			}
			if data, err := json.Marshal(causes); err == nil {
				selected[key] = attribute.StringValue(string(data))
			}
		case "panic":
			var exceptions []sentry.Exception
			if json.Unmarshal([]byte(value.AsString()), &exceptions) == nil {
				safe := protectSDKEvent(&sentry.Event{Exception: exceptions}, nil)
				if data, err := json.Marshal(safe.Exception); err == nil && len(data) <= 10000 {
					selected[key] = attribute.StringValue(string(data))
				}
			}
		case "incoming_path", "method", "status", "bytes", "content_length", "count", "suppressed_count":
			selected[key] = attribute.StringValue(boundDiagnosticText(value.AsString(), 2048))
		}
	}
	log.Attributes = selected
	return log
}

var diagnosticOperationPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$`)

func protectSDKTransaction(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	name := safeTransactionName(event.Transaction)
	if !strings.Contains(name, " ") && !diagnosticOperationPattern.MatchString(name) {
		name = "unknown"
	}
	trace := event.Contexts["trace"]
	event = protectSDKEvent(event, hint)
	event.Message = ""
	event.Transaction = name
	if trace != nil {
		safe := event.Contexts["trace"]
		if safe == nil {
			safe = sentry.Context{}
		}
		if op, ok := trace["op"].(string); ok && diagnosticOperationPattern.MatchString(op) {
			safe["op"] = op
		}
		if data, ok := trace["data"].(map[string]any); ok {
			safe["data"] = SanitizeEventContext(data)
		}
		event.Contexts["trace"] = safe
	}
	if len(event.Spans) > 64 {
		event.Spans = event.Spans[:64]
	}
	for _, span := range event.Spans {
		if !diagnosticOperationPattern.MatchString(span.Op) {
			span.Op = "unknown"
		}
		span.Name = ""
		span.Description = ""
		span.Tags = nil
		span.Origin = ""
		span.Data = SanitizeEventContext(span.Data)
	}
	return event
}

func protectExceptionFrames(exception *sentry.Exception) {
	exception.Type = safeDiagnosticType(exception.Type)
	if exception.Stacktrace == nil {
		return
	}
	for j := range exception.Stacktrace.Frames {
		frame := &exception.Stacktrace.Frames[j]
		frame.Vars = nil
		frame.PreContext = nil
		frame.PostContext = nil
		frame.ContextLine = ""
		frame.Filename = boundDiagnosticText(filepath.Base(strings.ReplaceAll(frame.Filename, "\\", "/")), 160)
		frame.AbsPath = ""
		frame.Function = boundDiagnosticText(frame.Function, 256)
	}
}
