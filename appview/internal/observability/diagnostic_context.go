package observability

import (
	"context"
	"log/slog"
	"regexp"
	"social.craftsky/appview/internal/ctxkeys"
	"strconv"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// WorkflowContext controls identity/activity selection independently of error
// explanations. Only the reviewed context types in this package implement it.
type WorkflowContext interface{ diagnosticFields() EventContext }

type PublicRecordContext struct {
	ActorDID  syntax.DID
	TargetDID syntax.DID
	URI       syntax.ATURI
	Handle    syntax.Handle
	CID       syntax.CID
	NSID      syntax.NSID
	RecordKey syntax.RecordKey
}

func (p PublicRecordContext) diagnosticFields() EventContext {
	values := map[string]string{
		"actor_did": p.ActorDID.String(), "target_did": p.TargetDID.String(),
		"record_uri": p.URI.String(), "handle": p.Handle.String(),
		"cid": p.CID.String(), "nsid": p.NSID.String(), "record_key": p.RecordKey.String(),
	}
	fields := EventContext{}
	for key, value := range values {
		if value != "" {
			fields[key] = value
		}
	}
	return fields
}

type DiagnosticInput struct {
	Error    error
	Context  EventContext
	Workflow WorkflowContext
}

// CaptureDiagnostic retains selected public workflow context without granting
// arbitrary error messages permission to reach the SDK.
func (o *Observer) CaptureDiagnostic(ctx context.Context, input DiagnosticInput) {
	if o == nil || input.Error == nil || !claimCapture(ctx) {
		return
	}
	if input.Workflow == nil {
		input.Workflow = RequestFailureWorkflow(ctx, input.Error)
	}
	// Keep the caller's context immutable; classification adds only to the copy.
	fields := EventContext{}
	for key, value := range input.Context {
		fields[key] = value
	}
	if o.sentryClient == nil {
		return
	}
	if ExpectedDiagnostic(input.Error, fields) || ctx.Err() == context.Canceled {
		return
	}
	classified := ClassifyError(input.Error, fields)
	fields["error_category"] = classified.Category
	fields["error_code"] = classified.Code
	fields["failure_stage"] = classified.Stage
	fields["result"] = classified.Result
	var workflow EventContext
	if selected := workflowForRequest(ctx, input.Workflow); selected != nil {
		workflow = selected.diagnosticFields()
	}
	o.captureExceptions(ctx, classified.Message, diagnosticExceptions(DescribeError(input.Error, fields)), fields, input.Error, workflow)
}

// AttemptedPublicDIDContext is only for a public DID validation boundary;
// it cannot claim that a failed parse yielded a trusted typed identifier.
type AttemptedPublicDIDContext struct{ Value string }

func (p AttemptedPublicDIDContext) diagnosticFields() EventContext {
	value := redactedValue
	if len(p.Value) <= 2048 && attemptedDIDPattern.MatchString(p.Value) {
		value = p.Value
	}
	return EventContext{"attempted_did": value, "identifier_valid": false, "validation_reason": "invalid DID"}
}

var attemptedDIDPattern = regexp.MustCompile(`^did:[a-z0-9]+:[A-Za-z0-9._:!-]+$`)

// PublishedRecordParseFailureContext may only be constructed from an already
// published record failing decode/indexing, never a publication request body.
type PublishedRecordParseFailureContext struct {
	Record PublicRecordContext
	Text   string
}

func (p PublishedRecordParseFailureContext) diagnosticFields() EventContext {
	fields := p.Record.diagnosticFields()
	if p.Text != "" {
		fields["public_excerpt"] = sanitizeKnownDiagnosticText(p.Text)
	}
	return fields
}

type requestObserverKey struct{}

// WithRequestObserver provides the HTTP operation's existing issue owner to
// cause-bearing handlers, without coupling handlers to SDK implementation.
func WithRequestObserver(ctx context.Context, observer *Observer) context.Context {
	ctx = context.WithValue(ctx, requestObserverKey{}, observer)
	return ctxkeys.WithDiagnosticBoundary(ctx, func(ctx context.Context, err error, fields map[string]any) []slog.Attr {
		observer.CaptureDiagnostic(ctx, DiagnosticInput{Error: err, Context: EventContext(fields)})
		return []slog.Attr{slog.Any("causes", DescribeError(err, EventContext(fields)))}
	})
}

func CaptureRequestDiagnostic(ctx context.Context, input DiagnosticInput) {
	if observer, ok := ctx.Value(requestObserverKey{}).(*Observer); ok {
		observer.CaptureDiagnostic(ctx, input)
	}
}

// TapEventFailureContext describes a publicly received record and its durable
// processing outcome, never the full envelope or dependency recipient state.
type TapEventFailureContext struct {
	Record           PublicRecordContext
	EventID          uint64
	Outcome          string
	Acknowledgement  string
	PublishedFailure *PublishedRecordParseFailureContext
	Reason           string
}

func (t TapEventFailureContext) diagnosticFields() EventContext {
	fields := t.Record.diagnosticFields()
	fields["event_id"] = strconv.FormatUint(t.EventID, 10)
	fields["outcome"] = t.Outcome
	fields["acknowledgement"] = t.Acknowledgement
	fields["reason"] = t.Reason
	if t.PublishedFailure != nil {
		for key, value := range t.PublishedFailure.diagnosticFields() {
			fields[key] = value
		}
	}
	return fields
}

// ReportRequestFailure is the cause-bearing owner for handlers without a
// feature logger. It leaves the public envelope and business result unchanged.
func ReportRequestFailure(ctx context.Context, err error, operation, stage string) {
	if err == nil {
		return
	}
	input := DiagnosticInput{Error: err, Context: EventContext{"component": "api", "operation": operation, "failure_stage": stage, "result": "error"}, Workflow: RequestFailureWorkflow(ctx, err)}
	logger := slog.Default()
	if observer, ok := ctx.Value(requestObserverKey{}).(*Observer); ok {
		logger = observer.logger
		observer.CaptureDiagnostic(ctx, input)
	}
	LogDiagnostic(ctx, logger, input)
}

// PrivateFailureContext is only for an operational failure, with the initiating
// account and a non-capability worker ID. Private targets never belong here.
func PrivateFailureContext(err error, actor syntax.DID, workflowID string) WorkflowContext {
	if err == nil {
		return nil
	}
	return privateFailureContext{actor: actor, workflowID: workflowID}
}

type privateFailureContext struct {
	actor      syntax.DID
	workflowID string
}

var workflowUUIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func (p privateFailureContext) diagnosticFields() EventContext {
	fields := EventContext{}
	if p.actor != "" {
		fields["operation_account_did"] = p.actor.String()
	}
	if workflowUUIDPattern.MatchString(p.workflowID) {
		fields["workflow_ref"] = p.workflowID
	} else if p.workflowID != "" {
		fields["workflow_ref"] = "[OMITTED: invalid workflow reference]"
	}
	return fields
}

// ObservePrivateFailure is a source-owned operational failure, not activity
// history. It never consumes a payload, recipient, lease token or private target.
func (o *Observer) ObservePrivateFailure(ctx context.Context, cause error, actor syntax.DID, workflowID, operation, stage, outcome string, attempt int) {
	if o == nil || cause == nil {
		return
	}
	input := DiagnosticInput{Error: cause, Context: EventContext{"component": "worker", "operation": operation, "failure_stage": stage, "result": outcome, "attempt": attempt}, Workflow: PrivateFailureContext(cause, actor, workflowID)}
	level := slog.LevelError
	if outcome == "retry" || outcome == "expected" {
		level = slog.LevelWarn
	}
	logDiagnosticAt(ctx, o.logger, input, level)
	o.CaptureDiagnostic(ctx, input)
}
