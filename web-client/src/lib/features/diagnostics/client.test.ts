import { beforeEach, afterEach, describe, it, expect, vi } from 'vitest';
const state = vi.hoisted(() => ({
	options: { enabled: true },
	close: vi.fn(async () => true),
	init: vi.fn(),
	clear: vi.fn()
}));
vi.mock('$app/env/public', () => ({
	PUBLIC_SENTRY_DSN: 'https://public@o1.ingest.sentry.io/1'
}));
vi.mock('@sentry/browser', () => ({
	init: state.init,
	globalHandlersIntegration: () => ({ name: 'GlobalHandlers' }),
	linkedErrorsIntegration: () => ({ name: 'LinkedErrors' }),
	dedupeIntegration: () => ({ name: 'Dedupe' }),
	makeFetchTransport: () => ({
		send: async () => ({ statusCode: 200 }),
		flush: async () => true
	}),
	getCurrentScope: () => ({ setClient: state.clear })
}));
beforeEach(() => {
	vi.resetModules();
	vi.clearAllMocks();
	state.options.enabled = true;
	state.init.mockReturnValue({
		getOptions: () => state.options,
		close: state.close
	});
	const values = new Map<string, string>();
	vi.stubGlobal('localStorage', {
		getItem: (key: string) => values.get(key) || null,
		setItem: (key: string, value: string) => values.set(key, value)
	});
	vi.stubGlobal('window', { dispatchEvent: vi.fn() });
});
afterEach(() => vi.unstubAllGlobals());
describe('optional diagnostic lifecycle', () => {
	it('does not initialize the SDK until explicit consent and closes it on revocation', async () => {
		const client = await import('./client.ts');
		await client.applyConsent();
		expect(state.init).not.toHaveBeenCalled();
		client.setConsent(true);
		await client.applyConsent();
		expect(state.init).toHaveBeenCalledTimes(1);
		client.setConsent(false);
		await client.applyConsent();
		expect(state.options.enabled).toBe(false);
		expect(state.close).toHaveBeenCalledTimes(1);
		expect(state.clear).toHaveBeenCalledWith(undefined);
	});
	it('does not reinitialize for repeated updates or reuse a closed client after re-enabling', async () => {
		const client = await import('./client.ts');
		client.setConsent(true);
		await client.applyConsent();
		await client.applyConsent();
		expect(state.init).toHaveBeenCalledTimes(1);
		client.setConsent(false);
		await client.applyConsent();
		client.setConsent(true);
		await client.applyConsent();
		expect(state.init).toHaveBeenCalledTimes(2);
	});
});
