import { beforeEach, afterEach, describe, it, expect, vi } from 'vitest';
const state = vi.hoisted(() => ({
	options: { enabled: true },
	close: vi.fn(async () => true),
	init: vi.fn(),
	capture: vi.fn(),
	clear: vi.fn()
}));
vi.mock('$app/env/public', () => ({
	PUBLIC_SENTRY_DSN: 'https://public@o1.ingest.sentry.io/1',
	PUBLIC_SENTRY_ENVIRONMENT: 'test'
}));
vi.mock('@sentry/browser', () => ({
	init: state.init,
	captureException: state.capture,
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
	vi.spyOn(console, 'error').mockImplementation(() => {});
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
afterEach(() => {
	vi.unstubAllGlobals();
	vi.restoreAllMocks();
});
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
	it('reports the original consumed failure only with consent and never leaks reporting failures', async () => {
		const client = await import('./client.ts');
		const error = new TypeError('PRIVATE_CANARY');
		await client.reportFailure(error, 'workspace');
		expect(state.capture).not.toHaveBeenCalled();
		client.setConsent(true);
		await client.reportFailure(error, 'workspace');
		expect(state.capture).toHaveBeenCalledWith(error, {
			contexts: { diagnostic: { operation: 'workspace' } }
		});
		expect(state.init.mock.calls[0][0]).toMatchObject({
			environment: 'test',
			release: expect.stringMatching(/^craftsky-web@/)
		});
		state.capture.mockImplementationOnce(() => {
			throw new Error('SDK unavailable');
		});
		await expect(
			client.reportFailure(error, 'composer')
		).resolves.toBeUndefined();
		client.setConsent(false);
		await client.reportFailure(error, 'framework');
		expect(state.capture).toHaveBeenCalledTimes(2);
	});
	it('does not export expected cancellation or offline failures', async () => {
		const client = await import('./client.ts');
		client.setConsent(true);
		await client.reportFailure(new TypeError('Failed to fetch'), 'workspace');
		await client.reportFailure(
			new DOMException('private', 'AbortError'),
			'composer'
		);
		expect(state.capture).not.toHaveBeenCalled();
	});
});
