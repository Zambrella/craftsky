package routes

import (
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/eligibility"
	"social.craftsky/appview/internal/moderation"
	"social.craftsky/appview/internal/safetyincident"
	"social.craftsky/appview/internal/safetyintake"
)

type moderationRouteBundle struct {
	mux         Registrar
	middleware  v1Middleware
	store       *moderation.Store
	commands    api.ModerationCommander
	sourceDID   syntax.DID
	config      Config
	imageHealth api.ImageSafetyHealthReader
	safetyWork  []api.SafetyWorkReader
	incidents   *safetyincident.Store
	intake      *safetyintake.Store
	evidence    *safetyincident.EvidenceService
	holds       *safetyincident.HoldService
	workflows   *safetyincident.WorkflowService
	csea        *safetyincident.CSEAWorkflow
	eligibility *eligibility.Store
	now         func() time.Time
}

func registerModerationRoutes(bundle moderationRouteBundle) {
	bundle.mux.Handle("GET /v1/moderation/standing", bundle.middleware.wrap(mustPolicy("GET", "/v1/moderation/standing"), api.ModerationStandingHandler(bundle.store)))
	bundle.mux.Handle("GET /v1/moderation/history", bundle.middleware.wrap(mustPolicy("GET", "/v1/moderation/history"), api.ModerationHistoryHandler(bundle.store)))
	bundle.mux.Handle("GET /v1/moderation/history/{caseReference}", bundle.middleware.wrap(mustPolicy("GET", "/v1/moderation/history/{caseReference}"), api.ModerationHistoryEntryHandler(bundle.store)))
	if !bundle.config.ModerationAdminEnabled {
		return
	}
	bundle.mux.Handle("GET /v1/admin/safety/status", bundle.middleware.wrap(
		mustConfiguredPolicy(bundle.config.Env, bundle.config, "GET", "/v1/admin/safety/status"),
		api.SafetyAdminStatusHandler(bundle.imageHealth, bundle.now, bundle.config.ImageSafetyAlertAge, bundle.safetyWork...),
	))
	bundle.mux.Handle("GET /v1/admin/safety/incidents/{incidentReference}", bundle.middleware.wrap(
		mustConfiguredPolicy(bundle.config.Env, bundle.config, "GET", "/v1/admin/safety/incidents/{incidentReference}"),
		api.RequireSafetyPermission(safetyincident.PermissionIncidentReadSafe)(api.SafetyIncidentDetailHandler(bundle.incidents)),
	))
	bundle.mux.Handle("POST /v1/admin/safety/external-intakes", bundle.middleware.wrap(
		mustConfiguredPolicy(bundle.config.Env, bundle.config, "POST", "/v1/admin/safety/external-intakes"),
		api.SafetyExternalIntakeHandler(bundle.intake),
	))
	bundle.mux.Handle("POST /v1/admin/safety/external-intakes/{intakeId}/correspondence", bundle.middleware.wrap(
		mustConfiguredPolicy(bundle.config.Env, bundle.config, "POST", "/v1/admin/safety/external-intakes/{intakeId}/correspondence"),
		api.SafetyExternalCorrespondenceHandler(bundle.intake),
	))
	bundle.mux.Handle("POST /v1/admin/safety/external-intakes/{intakeId}/appeal-link", bundle.middleware.wrap(
		mustConfiguredPolicy(bundle.config.Env, bundle.config, "POST", "/v1/admin/safety/external-intakes/{intakeId}/appeal-link"),
		api.SafetyExternalAppealLinkHandler(bundle.intake),
	))
	bundle.mux.Handle("POST /v1/admin/eligibility/reviews", bundle.middleware.wrap(
		mustConfiguredPolicy(bundle.config.Env, bundle.config, "POST", "/v1/admin/eligibility/reviews"),
		api.AgeEligibilityReviewHandler(bundle.eligibility, bundle.now),
	))
	safetyCommands := api.SafetyCommandServices{
		Evidence: bundle.evidence, Holds: bundle.holds, Workflows: bundle.workflows, CSEA: bundle.csea, Now: bundle.now,
	}
	for _, command := range []struct{ pattern, operation string }{
		{"/v1/admin/safety/incidents/{incidentReference}/evidence", "preserveEvidence"},
		{"/v1/admin/safety/evidence/{evidenceId}/access", "accessEvidence"},
		{"/v1/admin/safety/incidents/{incidentReference}/holds", "createHold"},
		{"/v1/admin/safety/workflows/intimate-images", "openIntimateImage"},
		{"/v1/admin/safety/workflows/intimate-images/{workflowId}/outcome", "completeIntimateImage"},
		{"/v1/admin/safety/workflows/credible-threats", "openCredibleThreat"},
		{"/v1/admin/safety/workflows/authority-requests", "openAuthorityRequest"},
		{"/v1/admin/safety/workflows/authority-requests/{workflowId}/verification", "verifyAuthority"},
		{"/v1/admin/safety/workflows/authority-requests/{workflowId}/disclosures", "discloseAuthority"},
		{"/v1/admin/safety/workflows/authority-requests/{workflowId}/closure", "closeAuthority"},
		{"/v1/admin/safety/incidents/{incidentReference}/csea-classification", "classifyCSEA"},
		{"/v1/admin/safety/incidents/{incidentReference}/authority-reports", "submitCSEAReport"},
		{"/v1/admin/safety/incidents/{incidentReference}/information-requests", "recordCSEAInformationRequest"},
		{"/v1/admin/safety/information-requests/{requestId}/response", "respondCSEAInformationRequest"},
		{"/v1/admin/safety/incidents/{incidentReference}/resolution", "resolveCSEA"},
	} {
		bundle.mux.Handle("POST "+command.pattern, bundle.middleware.wrap(
			mustConfiguredPolicy(bundle.config.Env, bundle.config, "POST", command.pattern),
			api.SafetyCommandHandler(safetyCommands, command.operation),
		))
	}
	bundle.mux.Handle("GET /v1/admin/moderation/cases", bundle.middleware.wrap(mustConfiguredPolicy(bundle.config.Env, bundle.config, "GET", "/v1/admin/moderation/cases"), api.ModerationAdminQueueHandler(bundle.store)))
	bundle.mux.Handle("GET /v1/admin/moderation/cases/{caseReference}", bundle.middleware.wrap(mustConfiguredPolicy(bundle.config.Env, bundle.config, "GET", "/v1/admin/moderation/cases/{caseReference}"), api.ModerationAdminDetailHandler(bundle.store)))
	commands := []struct{ path, kind string }{{"decisions", "decision"}, {"appeal-confirmations", "appealConfirmation"}, {"appeal-resolutions", "appealResolution"}, {"effect-changes", "effectChange"}, {"restorations", "restoration"}}
	for _, command := range commands {
		pattern := "/v1/admin/moderation/cases/{caseReference}/" + command.path
		bundle.mux.Handle("POST "+pattern, bundle.middleware.wrap(mustConfiguredPolicy(bundle.config.Env, bundle.config, "POST", pattern), api.ModerationAdminCommandHandler(bundle.commands, bundle.sourceDID, command.kind)))
	}
}
