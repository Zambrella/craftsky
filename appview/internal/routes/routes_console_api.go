package routes

import (
	"net/http"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/safetyincident"
)

var consoleReadRoutes = []struct {
	path, resource, param string
	permission            safetyincident.Permission
}{
	{"/v1/admin/accounts/{did}/eligibility", "eligibility", "", safetyincident.PermissionEligibilityReview},
	{"/v1/admin/operations", "operations", "", safetyincident.PermissionOperatorManage},
	{"/v1/admin/moderation/cases/{caseReference}/preview", "preview", "", safetyincident.PermissionModerationRead},
	{"/v1/admin/moderation/cases/{caseReference}/provenance", "case-provenance", "", safetyincident.PermissionModerationRead},
	{"/v1/admin/accounts/{did}/moderation", "standing", "", safetyincident.PermissionModerationRead},
	{"/v1/admin/safety/incidents/{incidentReference}/subjects", "incident-subjects", "", safetyincident.PermissionIncidentConfirm},
	{"/v1/admin/safety/incidents/{incidentReference}/reports", "incident-reports", "", safetyincident.PermissionAuthorityReport},
	{"/v1/admin/safety/incidents/{incidentReference}/information-requests", "information-requests", "", safetyincident.PermissionAuthorityReport},
	{"/v1/admin/safety/incidents", "incidents", "", safetyincident.PermissionSafetyRead},
	{"/v1/admin/safety/external-intakes", "external-intakes", "", safetyincident.PermissionIntakeManage},
	{"/v1/admin/safety/external-intakes/{intakeId}", "external-intakes", "intakeId", safetyincident.PermissionIntakeManage},
	{"/v1/admin/safety/evidence", "evidence", "", safetyincident.PermissionEvidenceAccess},
	{"/v1/admin/safety/evidence/{evidenceId}", "evidence", "evidenceId", safetyincident.PermissionEvidenceAccess},
	{"/v1/admin/safety/holds", "holds", "", safetyincident.PermissionHoldManage},
	{"/v1/admin/safety/holds/{holdId}", "holds", "holdId", safetyincident.PermissionHoldManage},
	{"/v1/admin/safety/workflows/{workflowId}/events", "workflow-events", "", safetyincident.PermissionSafetyRead},
	{"/v1/admin/safety/incidents/{incidentReference}/events", "incident-events", "", safetyincident.PermissionSafetyRead},
	{"/v1/admin/safety/external-intakes/{intakeId}/correspondence", "intake-correspondence", "", safetyincident.PermissionIntakeManage},
	{"/v1/admin/accounts/{did}/moderation/history", "account-history", "", safetyincident.PermissionModerationRead},
	{"/v1/admin/safety/workflows/{workflowId}", "workflows", "workflowId", safetyincident.PermissionSafetyRead},
	{"/v1/admin/safety/workflows", "workflows", "", safetyincident.PermissionSafetyRead},
	{"/v1/admin/safety/scans", "scans", "", safetyincident.PermissionSafetyRead},
	{"/v1/admin/safety/scans/{scanId}", "scans", "scanId", safetyincident.PermissionSafetyRead},
}

func registerConsoleAPIRoutes(mux Registrar, middleware v1Middleware, deps *Dependencies) {
	if !deps.Config.ModerationAdminEnabled {
		return
	}
	for _, route := range consoleReadRoutes {
		handler := api.ConsoleListHandler(deps.DB, route.resource)
		if route.resource == "eligibility" {
			handler = api.ConsoleAccountEligibilityHandler(deps.DB)
		} else if route.resource == "preview" {
			handler = api.ConsoleCasePreviewHandler(deps.DB)
		} else if route.resource == "standing" {
			handler = api.ConsoleAccountStandingHandler(deps.DB)
		} else if route.param != "" {
			handler = api.ConsoleDetailHandler(deps.DB, route.resource, route.param)
		}
		mux.Handle("GET "+route.path, middleware.wrap(mustConfiguredPolicy(deps.Config.Env, deps.Config, "GET", route.path), handler))
	}
	for _, route := range consoleMutationRoutes {
		var handler http.Handler
		switch route.operation {
		case "confirm":
			handler = api.ConsoleDetectionConfirmationHandler(deps.DB)
		case "retry":
			handler = api.ConsoleScanRetryHandler(deps.DB)
		case "case":
			handler = api.ConsoleCaseCreationHandler(deps.DB)
		case "release":
			handler = api.ConsoleHoldReleaseHandler(deps.DB)
		}
		mux.Handle("POST "+route.path, middleware.wrap(mustConfiguredPolicy(deps.Config.Env, deps.Config, "POST", route.path), handler))
	}

}

func consoleAPIPolicies() []RoutePolicy {
	policies := []RoutePolicy{}
	for _, route := range consoleMutationRoutes {
		policies = append(policies, RoutePolicy{Method: "POST", PathPattern: route.path, RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessModerator, OperatorPermission: route.permission})
	}
	for _, route := range consoleReadRoutes {
		policies = append(policies, RoutePolicy{Method: "GET", PathPattern: route.path, RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessModerator, OperatorPermission: route.permission})
	}
	return policies
}

var consoleMutationRoutes = []struct {
	path, operation string
	permission      safetyincident.Permission
}{
	{"/v1/admin/moderation/cases", "case", safetyincident.PermissionModerationManage},
	{"/v1/admin/safety/incidents/{incidentReference}/confirmations", "confirm", safetyincident.PermissionIncidentConfirm},
	{"/v1/admin/safety/scans/{scanId}/retries", "retry", safetyincident.PermissionScanRetry},
	{"/v1/admin/safety/holds/{holdId}/releases", "release", safetyincident.PermissionHoldManage},
}
