import * as publicEnv from '$app/env/public';
import { diagnosticOptions } from './options.ts';
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
	} catch {
		throw new Error('This browser could not save your privacy preference.');
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
			diagnosticOptions(loaded, publicEnv.PUBLIC_SENTRY_DSN, allowed)
		);
	} catch {
		/* Optional diagnostics must never affect normal application behavior. */
	}
}
