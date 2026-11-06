import { UserFacingError } from '#lib/user-facing-error.ts';
import * as publicEnv from '$app/env/public';
import { diagnosticOptions } from './options.ts';
import {
	expectedFailure,
	failureContext,
	type DiagnosticOperation
} from './privacy.ts';
import { logFailure } from './local.ts';
const key = 'craftsky.diagnostics-consent';
export const configured = () => !!publicEnv.PUBLIC_SENTRY_DSN;
export function consent(): boolean {
	try {
		return (
			typeof localStorage !== 'undefined' &&
			localStorage.getItem(key) === 'true'
		);
	} catch {
		return false;
	}
}
export function setConsent(value: boolean) {
	try {
		localStorage.setItem(key, String(value));
	} catch (error) {
		throw new UserFacingError(
			'This browser could not save your privacy preference.',
			{ cause: error }
		);
	}
	if (typeof window !== 'undefined')
		window.dispatchEvent(new Event('craftsky-diagnostics-change'));
}
type SDK = typeof import('@sentry/browser');
let sdk: SDK | undefined,
	client: ReturnType<SDK['init']>,
	revision = 0;
const allowed = () => configured() && consent();
export async function applyConsent() {
	const version = ++revision;
	if (!allowed()) {
		const previous = client;
		client = undefined;
		sdk?.getCurrentScope().setClient(undefined);
		if (previous) {
			previous.getOptions().enabled = false;
			void Promise.resolve(previous.close(1)).catch(() => {});
		}
		return;
	}
	if (client) return;
	try {
		const loaded = await import('@sentry/browser');
		if (version !== revision || !allowed()) return;
		sdk = loaded;
		client = loaded.init(
			diagnosticOptions(
				loaded,
				publicEnv.PUBLIC_SENTRY_DSN,
				allowed,
				undefined,
				{
					release: __CRAFTSKY_WEB_RELEASE__,
					environment: publicEnv.PUBLIC_SENTRY_ENVIRONMENT || 'development'
				}
			)
		);
	} catch {
		/* Optional diagnostics must never affect normal application behavior. */
	}
}

/** The boundary consuming a failure owns reporting; the original cause stays in memory. */
export async function reportFailure(
	error: unknown,
	operation: DiagnosticOperation
) {
	try {
		logFailure(error, operation);
		if (expectedFailure(error) || !allowed()) return;
		await applyConsent();
		if (!client || !sdk || !allowed()) return;
		sdk.captureException(error, {
			contexts: { diagnostic: failureContext(error, operation) }
		});
	} catch {
		/* Optional reporting never changes responses, retries or navigation. */
	}
}
