package api

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/safetyincident"
)

type SafetyCommandServices struct {
	Evidence  *safetyincident.EvidenceService
	Holds     *safetyincident.HoldService
	Workflows *safetyincident.WorkflowService
	CSEA      *safetyincident.CSEAWorkflow
	Now       func() time.Time
}

func SafetyCommandHandler(services SafetyCommandServices, operation string) http.Handler {
	if services.Now == nil {
		services.Now = time.Now
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := safetyActor(r)
		if !ok {
			adminError(w, r, http.StatusUnauthorized, "moderator_authentication_failed", "moderator authentication failed")
			return
		}
		now := services.Now().UTC()
		if !safetyCommandAvailable(services, operation) {
			writeSafetyCommandError(w, r, errors.New("safety command dependency unavailable"))
			return
		}
		var err error
		switch operation {
		case "preserveEvidence":
			incidentID, parseErr := uuid.Parse(r.PathValue("incidentReference"))
			var request struct {
				Bytes       []byte    `json:"bytes"`
				ContentType string    `json:"contentType"`
				Reason      string    `json:"reason"`
				ExpiresAt   time.Time `json:"expiresAt"`
			}
			if parseErr != nil || decodeSafetyCommand(r, &request) != nil {
				writeSafetyCommandError(w, r, safetyincident.ErrInvalidEvidence)
				return
			}
			result, commandErr := services.Evidence.Preserve(r.Context(), actor, safetyincident.PreserveCommand{
				IncidentID: incidentID, Bytes: request.Bytes, SHA256: sha256.Sum256(request.Bytes),
				ContentType: request.ContentType, Reason: request.Reason, ExpiresAt: request.ExpiresAt,
			})
			if commandErr == nil {
				writeSafetyCommandJSON(w, http.StatusCreated, map[string]any{"id": result.ID, "objectKey": result.ObjectKey})
				return
			}
			err = commandErr
		case "accessEvidence":
			evidenceID, parseErr := uuid.Parse(r.PathValue("evidenceId"))
			var request struct {
				Reason string `json:"reason"`
			}
			if parseErr != nil || decodeSafetyCommand(r, &request) != nil {
				writeSafetyCommandError(w, r, safetyincident.ErrInvalidEvidence)
				return
			}
			reader, commandErr := services.Evidence.Access(r.Context(), actor, safetyincident.AccessCommand{EvidenceID: evidenceID, Reason: request.Reason})
			if commandErr == nil {
				defer reader.Close()
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("Cache-Control", "no-store")
				_, _ = io.Copy(w, reader)
				return
			}
			err = commandErr
		case "createHold":
			incidentID, parseErr := uuid.Parse(r.PathValue("incidentReference"))
			var request struct {
				EvidenceIDs []uuid.UUID `json:"evidenceIds"`
				Basis       string      `json:"basis"`
				ApprovedBy  string      `json:"approvedBy"`
				ExpiresAt   time.Time   `json:"expiresAt"`
			}
			if parseErr != nil || decodeSafetyCommand(r, &request) != nil {
				writeSafetyCommandError(w, r, safetyincident.ErrInvalidHold)
				return
			}
			id, commandErr := services.Holds.Create(r.Context(), actor, safetyincident.HoldRequest{
				IncidentID: incidentID, EvidenceIDs: request.EvidenceIDs, Basis: request.Basis,
				ApprovedBy: request.ApprovedBy, CreatedBy: actor.ID, CreatedAt: now, ExpiresAt: request.ExpiresAt,
			})
			if commandErr == nil {
				writeSafetyCommandJSON(w, http.StatusCreated, map[string]any{"id": id})
				return
			}
			err = commandErr
		case "openIntimateImage":
			var request struct {
				IntakeID         uuid.UUID `json:"intakeId"`
				ReceivedAt       time.Time `json:"receivedAt"`
				StandingCode     string    `json:"standingCode"`
				DeclarationsCode string    `json:"declarationsCode"`
			}
			if decodeSafetyCommand(r, &request) != nil {
				writeSafetyCommandError(w, r, safetyincident.ErrInvalidWorkflowTransition)
				return
			}
			id, deadline, commandErr := services.Workflows.OpenIntimateImage(r.Context(), safetyincident.IntimateImageCommand{
				IntakeID: request.IntakeID, ReceivedAt: request.ReceivedAt, Actor: actor,
				StandingCode: request.StandingCode, DeclarationsCode: request.DeclarationsCode,
			})
			if commandErr == nil {
				writeSafetyCommandJSON(w, http.StatusCreated, map[string]any{"id": id, "deadlineAt": deadline.DueAt, "alertAt": deadline.AlertAt})
				return
			}
			err = commandErr
		case "completeIntimateImage":
			workflowID, parseErr := uuid.Parse(r.PathValue("workflowId"))
			var request struct {
				JudgmentCode                string `json:"judgmentCode"`
				SameImageSearchCode         string `json:"sameImageSearchCode"`
				SubstantiallySameSearchCode string `json:"substantiallySameSearchCode"`
				ActionCode                  string `json:"actionCode"`
				ExceptionCode               string `json:"exceptionCode"`
				OutcomeCode                 string `json:"outcomeCode"`
			}
			if parseErr != nil || decodeSafetyCommand(r, &request) != nil {
				writeSafetyCommandError(w, r, safetyincident.ErrInvalidWorkflowTransition)
				return
			}
			err = services.Workflows.ResolveIntimateImage(r.Context(), safetyincident.IntimateImageOutcome{
				WorkflowID: workflowID, Actor: actor, JudgmentCode: request.JudgmentCode,
				SameImageSearchCode: request.SameImageSearchCode, SubstantiallySameSearchCode: request.SubstantiallySameSearchCode,
				ActionCode: request.ActionCode, ExceptionCode: request.ExceptionCode, OutcomeCode: request.OutcomeCode,
			})
		case "openCredibleThreat":
			var request struct {
				ReceivedAt       time.Time `json:"receivedAt"`
				PreservationCode string    `json:"preservationCode"`
			}
			if decodeSafetyCommand(r, &request) != nil {
				writeSafetyCommandError(w, r, safetyincident.ErrInvalidWorkflowTransition)
				return
			}
			id, commandErr := services.Workflows.OpenCredibleThreat(r.Context(), safetyincident.CredibleThreat{ReceivedAt: request.ReceivedAt, Actor: actor, PreservationCode: request.PreservationCode})
			if commandErr == nil {
				writeSafetyCommandJSON(w, http.StatusCreated, map[string]any{"id": id})
				return
			}
			err = commandErr
		case "openAuthorityRequest":
			var request struct {
				ReceivedAt         time.Time `json:"receivedAt"`
				RequesterReference string    `json:"requesterReference"`
			}
			if decodeSafetyCommand(r, &request) != nil {
				writeSafetyCommandError(w, r, safetyincident.ErrInvalidWorkflowTransition)
				return
			}
			id, commandErr := services.Workflows.OpenAuthorityRequest(r.Context(), safetyincident.AuthorityRequest{ReceivedAt: request.ReceivedAt, Actor: actor, RequesterReference: request.RequesterReference})
			if commandErr == nil {
				writeSafetyCommandJSON(w, http.StatusCreated, map[string]any{"id": id})
				return
			}
			err = commandErr
		case "verifyAuthority":
			workflowID, parseErr := uuid.Parse(r.PathValue("workflowId"))
			var request struct {
				VerificationReference string `json:"verificationReference"`
				Genuine               bool   `json:"genuine"`
			}
			if parseErr != nil || decodeSafetyCommand(r, &request) != nil {
				writeSafetyCommandError(w, r, safetyincident.ErrInvalidWorkflowTransition)
				return
			}
			err = services.Workflows.VerifyAuthority(r.Context(), workflowID, actor, request.VerificationReference, request.Genuine)
		case "discloseAuthority":
			workflowID, parseErr := uuid.Parse(r.PathValue("workflowId"))
			var request struct {
				LegalBasis string   `json:"legalBasis"`
				Fields     []string `json:"fields"`
			}
			if parseErr != nil || decodeSafetyCommand(r, &request) != nil {
				writeSafetyCommandError(w, r, safetyincident.ErrInvalidWorkflowTransition)
				return
			}
			err = services.Workflows.ApproveAndDisclose(r.Context(), workflowID, actor, request.LegalBasis, request.Fields)
		case "closeAuthority":
			workflowID, parseErr := uuid.Parse(r.PathValue("workflowId"))
			var request struct {
				ReviewCode string `json:"reviewCode"`
			}
			if parseErr != nil || decodeSafetyCommand(r, &request) != nil {
				writeSafetyCommandError(w, r, safetyincident.ErrInvalidWorkflowTransition)
				return
			}
			err = services.Workflows.CloseAuthorityRequest(r.Context(), workflowID, actor, request.ReviewCode)
		case "classifyCSEA":
			incidentID, parseErr := uuid.Parse(r.PathValue("incidentReference"))
			var request struct {
				Priority          safetyincident.CSEAPriority `json:"priority"`
				Assignee          string                      `json:"assignee"`
				ReportingDeadline time.Time                   `json:"reportingDeadline"`
				SourceSystem      string                      `json:"sourceSystem"`
				ReplayID          string                      `json:"replayId"`
			}
			if parseErr != nil || decodeSafetyCommand(r, &request) != nil {
				writeSafetyCommandError(w, r, safetyincident.ErrInvalidCSEAWorkflow)
				return
			}
			err = services.CSEA.ClassifyAndAssign(r.Context(), safetyincident.ClassificationCommand{
				IncidentID: incidentID, Priority: request.Priority, Assignee: request.Assignee,
				ReportingDeadline: request.ReportingDeadline, Actor: actor, SourceSystem: request.SourceSystem,
				ReplayID: request.ReplayID, At: now,
			})
		case "submitCSEAReport":
			incidentID, parseErr := uuid.Parse(r.PathValue("incidentReference"))
			var request struct {
				Kind               safetyincident.ReportKind `json:"kind"`
				AuthorityReference string                    `json:"authorityReference"`
				DuplicateOfID      uuid.UUID                 `json:"duplicateOfId"`
				RetentionUntil     time.Time                 `json:"retentionUntil"`
				SourceSystem       string                    `json:"sourceSystem"`
				ReplayID           string                    `json:"replayId"`
			}
			if parseErr != nil || decodeSafetyCommand(r, &request) != nil {
				writeSafetyCommandError(w, r, safetyincident.ErrInvalidCSEAWorkflow)
				return
			}
			result, commandErr := services.CSEA.SubmitReport(r.Context(), safetyincident.ReportCommand{
				IncidentID: incidentID, Kind: request.Kind, AuthorityReference: request.AuthorityReference,
				DuplicateOfID: request.DuplicateOfID, RetentionUntil: request.RetentionUntil,
				Actor: actor, SourceSystem: request.SourceSystem, ReplayID: request.ReplayID, At: now,
			})
			if commandErr == nil {
				writeSafetyCommandJSON(w, http.StatusCreated, map[string]any{"id": result.ID, "deadlineMet": result.DeadlineMet})
				return
			}
			err = commandErr
		case "recordCSEAInformationRequest":
			incidentID, parseErr := uuid.Parse(r.PathValue("incidentReference"))
			var request struct {
				AuthorityReference string    `json:"authorityReference"`
				DueAt              time.Time `json:"dueAt"`
				SourceSystem       string    `json:"sourceSystem"`
				ReplayID           string    `json:"replayId"`
			}
			if parseErr != nil || decodeSafetyCommand(r, &request) != nil {
				writeSafetyCommandError(w, r, safetyincident.ErrInvalidCSEAWorkflow)
				return
			}
			result, commandErr := services.CSEA.RecordInformationRequest(r.Context(), safetyincident.InformationRequestCommand{
				IncidentID: incidentID, AuthorityReference: request.AuthorityReference, DueAt: request.DueAt,
				Actor: actor, SourceSystem: request.SourceSystem, ReplayID: request.ReplayID, At: now,
			})
			if commandErr == nil {
				writeSafetyCommandJSON(w, http.StatusCreated, map[string]any{"id": result.ID})
				return
			}
			err = commandErr
		case "respondCSEAInformationRequest":
			requestID, parseErr := uuid.Parse(r.PathValue("requestId"))
			var request struct {
				SourceSystem string `json:"sourceSystem"`
				ReplayID     string `json:"replayId"`
			}
			if parseErr != nil || decodeSafetyCommand(r, &request) != nil {
				writeSafetyCommandError(w, r, safetyincident.ErrInvalidCSEAWorkflow)
				return
			}
			err = services.CSEA.RespondToInformationRequest(r.Context(), safetyincident.InformationResponseCommand{
				RequestID: requestID, Actor: actor, SourceSystem: request.SourceSystem, ReplayID: request.ReplayID, At: now,
			})
		case "resolveCSEA":
			incidentID, parseErr := uuid.Parse(r.PathValue("incidentReference"))
			var request struct {
				Outcome      safetyincident.ResolutionOutcome `json:"outcome"`
				SourceSystem string                           `json:"sourceSystem"`
				ReplayID     string                           `json:"replayId"`
			}
			if parseErr != nil || decodeSafetyCommand(r, &request) != nil {
				writeSafetyCommandError(w, r, safetyincident.ErrInvalidCSEAWorkflow)
				return
			}
			err = services.CSEA.Resolve(r.Context(), safetyincident.ResolutionCommand{
				IncidentID: incidentID, Outcome: request.Outcome, Actor: actor,
				SourceSystem: request.SourceSystem, ReplayID: request.ReplayID, At: now,
			})
		default:
			writeSafetyCommandError(w, r, safetyincident.ErrInvalidWorkflowTransition)
			return
		}
		if err != nil {
			writeSafetyCommandError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func safetyCommandAvailable(services SafetyCommandServices, operation string) bool {
	switch operation {
	case "preserveEvidence", "accessEvidence":
		return services.Evidence != nil
	case "createHold":
		return services.Holds != nil
	case "openIntimateImage", "completeIntimateImage", "openCredibleThreat", "openAuthorityRequest",
		"verifyAuthority", "discloseAuthority", "closeAuthority":
		return services.Workflows != nil
	case "classifyCSEA", "submitCSEAReport", "recordCSEAInformationRequest", "respondCSEAInformationRequest", "resolveCSEA":
		return services.CSEA != nil
	default:
		return false
	}
}

func safetyActor(r *http.Request) (safetyincident.Actor, bool) {
	moderator, ok := ctxkeys.GetModerator(r.Context())
	if !ok {
		return safetyincident.Actor{}, false
	}
	permissions := make(map[safetyincident.Permission]bool, len(moderator.Permissions))
	for permission, allowed := range moderator.Permissions {
		permissions[safetyincident.Permission(permission)] = allowed
	}
	assignments := make(map[uuid.UUID]bool, len(moderator.AssignedIncidentReferences))
	for reference, assigned := range moderator.AssignedIncidentReferences {
		if id, err := uuid.Parse(reference); err == nil {
			assignments[id] = assigned
		}
	}
	if moderator.Permissions == nil {
		permissions = nil
	}
	return safetyincident.Actor{ID: moderator.ActorID, Role: safetyincident.Role(moderator.Role), Permissions: permissions, AssignedIncidents: assignments}, true
}

func decodeSafetyCommand(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}

func writeSafetyCommandJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeSafetyCommandError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := http.StatusServiceUnavailable, "safety_command_unavailable", "safety command unavailable"
	if errors.Is(err, safetyincident.ErrUnauthorized) {
		status, code, message = http.StatusForbidden, "safety_action_forbidden", "safety action forbidden"
	} else if errors.Is(err, safetyincident.ErrInvalidEvidence) || errors.Is(err, safetyincident.ErrInvalidHold) ||
		errors.Is(err, safetyincident.ErrInvalidWorkflowTransition) || errors.Is(err, safetyincident.ErrInvalidCSEAWorkflow) {
		status, code, message = http.StatusBadRequest, "invalid_request", "invalid request"
	} else if errors.Is(err, safetyincident.ErrAuthorityNotVerified) || errors.Is(err, safetyincident.ErrLawfulApprovalRequired) {
		status, code, message = http.StatusConflict, "invalid_workflow_state", "invalid workflow state"
	}
	adminError(w, r, status, code, message)
}
