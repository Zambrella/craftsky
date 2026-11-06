import { afterEach, describe, it, expect, vi } from 'vitest';
import { logServerFailure } from './diagnostics';
import { safeFailure, UpstreamError } from './auth/upstream';
import { publicAPI } from './api';
afterEach(() => vi.restoreAllMocks());
const id = '12345678-1234-4123-8123-123456789012';

describe('protected Worker diagnostics', () => {
	it('preserves cause types and deployed coordinates, excludes private prose and paths', () => {
		const sink = vi.spyOn(console, 'error').mockImplementation(() => {});
		const cause = new SyntaxError('PRIVATE_BODY');
		cause.stack =
			'SyntaxError: PRIVATE_BODY\n at PRIVATE_FUNCTION (worker.js:42:7)\n at f (/Users/PRIVATE_HOME/private.ts:1:2)';
		const error = new UpstreamError(
			502,
			'invalid_upstream_response',
			undefined,
			{ cause }
		);
		error.requestId = id;
		const response = safeFailure(error);
		expect(response.status).toBe(502);
		const wire = String(sink.mock.calls[0][0]);
		expect(wire).toContain('SyntaxError');
		expect(wire).toContain('"line":42');
		expect(wire).toContain(id);
		expect(wire).toContain('invalid_upstream_response');
		expect(wire).not.toContain('PRIVATE');
	});
	it('keeps expected failures at warning severity and survives sink failures', () => {
		const warning = vi.spyOn(console, 'warn').mockImplementation(() => {
			throw new Error('sink');
		});
		expect(safeFailure(new UpstreamError(401, 'unauthorized')).status).toBe(
			401
		);
		expect(warning).toHaveBeenCalledOnce();
	});
	it('logs the consumed public parse cause once without exposing response content', async () => {
		const sink = vi.spyOn(console, 'error').mockImplementation(() => {});
		await expect(
			publicAPI(
				async () =>
					new Response('PRIVATE_BODY', { headers: { 'X-Request-ID': id } }),
				'private?token=PRIVATE_TOKEN'
			)
		).rejects.toBeDefined();
		expect(sink).toHaveBeenCalledOnce();
		const wire = String(sink.mock.calls[0][0]);
		expect(wire).toContain('SyntaxError');
		expect(wire).toContain('decode');
		expect(wire).toContain(id);
		expect(wire).not.toContain('PRIVATE');
	});
	it('excludes malformed correlation and arbitrary error fields', () => {
		const sink = vi.spyOn(console, 'error').mockImplementation(() => {});
		logServerFailure(
			Object.assign(new Error('PRIVATE'), {
				cookie: 'PRIVATE',
				name: 'PRIVATE'
			}),
			'worker.unhandled',
			'request',
			{ code: 'PRIVATE prose', requestId: 'PRIVATE' }
		);
		expect(String(sink.mock.calls[0][0])).not.toContain('PRIVATE');
	});
});

it('the Worker hook owns unexpected errors without duplicating handled HTTP failures', async () => {
	const { handleError } = await import('../../hooks.server');
	const sink = vi.spyOn(console, 'error').mockImplementation(() => {});
	const unexpected = {
		kind: 'unknown',
		error: new Error('PRIVATE_CANARY'),
		event: {}
	} as Parameters<typeof handleError>[0];
	expect(await handleError(unexpected)).toEqual({
		message: 'Something went wrong. Please try again.'
	});
	expect(sink).toHaveBeenCalledOnce();
	await handleError({
		kind: 'app',
		error: { message: 'PRIVATE_CANARY' },
		event: {}
	} as Parameters<typeof handleError>[0]);
	expect(sink).toHaveBeenCalledOnce();
	expect(String(sink.mock.calls[0][0])).not.toContain('PRIVATE_CANARY');
});
