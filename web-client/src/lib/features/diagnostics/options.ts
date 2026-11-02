import type * as Sentry from '@sentry/browser';
import { privateTransport, safeEvent } from './privacy.ts';

export function diagnosticOptions(
	sdk: typeof Sentry,
	dsn: string,
	allowed: () => boolean,
	nativeFetch?: typeof fetch
): Sentry.BrowserOptions {
	return {
		dsn,
		defaultIntegrations: false,
		integrations: [
			sdk.globalHandlersIntegration(),
			sdk.linkedErrorsIntegration({ limit: 5 }),
			sdk.dedupeIntegration()
		],
		// Current Sentry 11 uses granular dataCollection in place of sendDefaultPii.
		dataCollection: {
			userInfo: false,
			cookies: false,
			httpHeaders: false,
			httpBodies: [],
			urlQueryParams: false,
			stackFrameVariables: false,
			frameContextLines: 0,
			graphQL: { document: false, variables: false },
			genAI: { inputs: false, outputs: false },
			mcp: { inputs: false, outputs: false },
			databaseQueryData: false,
			queues: false
		},
		traceLifecycle: 'static',
		tracePropagationTargets: [],
		sendClientReports: false,
		maxBreadcrumbs: 0,
		debug: false,
		spotlight: false,
		profileSessionSampleRate: 0,
		replaysSessionSampleRate: 0,
		replaysOnErrorSampleRate: 0,
		beforeSend(event, hint) {
			hint.attachments = [];
			return safeEvent(event, hint, allowed());
		},
		beforeBreadcrumb: () => null,
		beforeSendLog: () => null,
		beforeSendMetric: () => null,
		beforeSendTransaction: () => null,
		transport: (options) =>
			privateTransport(sdk.makeFetchTransport(options, nativeFetch), allowed)
	};
}
