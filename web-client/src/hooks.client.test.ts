import { describe, expect, it, vi } from 'vitest';
const spies = vi.hoisted(() => ({
	report: vi.fn(),
	consent: vi.fn(),
	local: vi.fn(() => () => {})
}));
vi.mock('#lib/features/diagnostics/client.ts', () => ({
	reportFailure: spies.report,
	applyConsent: spies.consent
}));
vi.mock('#lib/features/diagnostics/local.ts', () => ({
	installLocalDiagnostics: spies.local
}));
import { handleError, init } from './hooks.client';

describe('browser framework diagnostics', () => {
	it('initializes local protection before optional consent-based reporting', async () => {
		await init();
		expect(spies.local).toHaveBeenCalledOnce();
		expect(spies.consent).toHaveBeenCalledOnce();
		expect(spies.local.mock.invocationCallOrder[0]).toBeLessThan(
			spies.consent.mock.invocationCallOrder[0]
		);
	});
	it('captures original unexpected errors without turning expected framework/app errors into issues', () => {
		const error = new TypeError('private');
		for (const kind of ['app', 'framework', 'unknown'] as const)
			handleError({ error, kind } as Parameters<typeof handleError>[0]);
		expect(spies.report).toHaveBeenCalledExactlyOnceWith(error, 'framework');
	});
});
