import { afterEach, describe, expect, it, vi } from 'vitest';
import { installLocalDiagnostics } from './local.ts';

afterEach(() => vi.restoreAllMocks());
const event = (type: string, fields: Record<string, unknown>) =>
	Object.assign(new Event(type, { cancelable: true }), fields);
describe('protected local browser diagnostics without the SDK', () => {
	it('serializes a native error kind and allowed code coordinates without private console prose', () => {
		const output: string[] = [];
		vi.spyOn(console, 'error').mockImplementation((value) => {
			output.push(String(value));
		});
		const target = new EventTarget(),
			cleanup = installLocalDiagnostics(target as unknown as Window);
		const error = new TypeError(
			'PRIVATE_MESSAGE_CANARY token=PRIVATE_TOKEN_CANARY'
		);
		error.stack =
			'TypeError: PRIVATE_MESSAGE_CANARY\n at PRIVATE_FUNCTION_CANARY (https://app.craftsky.social/_app/immutable/chunks/example.js?token=PRIVATE_TOKEN_CANARY:18:7)\n at private (file:///Users/PRIVATE_HOME_CANARY/private.js:5:8)\n at private (https://private.example/PRIVATE_PATH_CANARY:9:2)';
		const failure = event('error', {
			error,
			message: 'PRIVATE_EVENT_CANARY',
			filename: 'https://private.example/PRIVATE_FALLBACK_CANARY',
			lineno: 2,
			colno: 3
		});
		expect(target.dispatchEvent(failure)).toBe(false);
		expect(JSON.parse(output[0])).toEqual({
			message: 'Unexpected browser error',
			exceptions: [
				{
					kind: 'TypeError',
					frames: [
						{
							filename: '/_app/immutable/chunks/example.js',
							lineno: 18,
							colno: 7,
							in_app: true
						}
					]
				}
			]
		});
		expect(output.join('')).not.toMatch(/PRIVATE_/);
		cleanup();
	});
	it('preserves typed causes and the original rejection for the SDK owner while selecting safe local fields', () => {
		const output: string[] = [];
		vi.spyOn(console, 'error').mockImplementation((value) => {
			output.push(String(value));
		});
		const target = new EventTarget(),
			cleanup = installLocalDiagnostics(target as unknown as Window);
		const cause = new ReferenceError('PRIVATE_CAUSE_CANARY');
		cause.stack =
			'ReferenceError: PRIVATE_CAUSE_CANARY\nprivate@https://app.craftsky.social/src/lib/example.ts?code=PRIVATE_CODE_CANARY:10:4';
		const original = new RangeError('PRIVATE_REJECTION_CANARY', { cause });
		original.stack =
			'RangeError: PRIVATE_REJECTION_CANARY\n at fn (https://app.craftsky.social/_app/immutable/entry/start.js:6:2)';
		Object.assign(original, {
			cookies: 'PRIVATE_COOKIE_CANARY',
			body: 'PRIVATE_BODY_CANARY'
		});
		let observed: unknown;
		target.addEventListener('unhandledrejection', (value) => {
			observed = (value as PromiseRejectionEvent).reason;
		});
		const rejection = event('unhandledrejection', { reason: original });
		target.dispatchEvent(rejection);
		expect(rejection.defaultPrevented).toBe(true);
		expect(observed).toBe(original);
		expect(rejection.cancelBubble).toBe(false);
		expect(JSON.parse(output[0]).exceptions).toEqual([
			{
				kind: 'RangeError',
				frames: [
					{
						filename: '/_app/immutable/entry/start.js',
						lineno: 6,
						colno: 2,
						in_app: true
					}
				]
			},
			{
				kind: 'ReferenceError',
				frames: [
					{
						filename: '/src/lib/example.ts',
						lineno: 10,
						colno: 4,
						in_app: true
					}
				]
			}
		]);
		expect(output.join('')).not.toMatch(/PRIVATE_/);
		cleanup();
	});
	it('contains console failures and removes both handlers without consuming future events', () => {
		const logger = vi.spyOn(console, 'error').mockImplementation(() => {
			throw new Error('PRIVATE_LOGGER_CANARY');
		});
		const target = new EventTarget(),
			cleanup = installLocalDiagnostics(target as unknown as Window);
		const failure = event('unhandledrejection', {
			reason: { private: 'PRIVATE_OBJECT_CANARY' }
		});
		expect(() => target.dispatchEvent(failure)).not.toThrow();
		expect(failure.defaultPrevented).toBe(true);
		expect(logger).toHaveBeenCalledOnce();
		expect(String(logger.mock.calls[0][0])).not.toMatch(/PRIVATE_/);
		cleanup();
		expect(
			target.dispatchEvent(
				event('unhandledrejection', { reason: 'PRIVATE_LATER_CANARY' })
			)
		).toBe(true);
		expect(
			target.dispatchEvent(
				event('error', { error: new Error('PRIVATE_LATER_CANARY') })
			)
		).toBe(true);
		expect(logger).toHaveBeenCalledOnce();
	});
});

it('logs expected offline failures at warning level with safe correlation', async () => {
	const { logFailure } = await import('./local');
	const { APIError } = await import('#lib/api/transport.ts');
	const warning = vi.spyOn(console, 'warn').mockImplementation(() => {});
	const issue = vi.spyOn(console, 'error').mockImplementation(() => {});
	logFailure(
		new APIError(0, 'network_unavailable', 'PRIVATE_CANARY'),
		'social.Profile.profiles.get'
	);
	expect(issue).not.toHaveBeenCalled();
	expect(warning).toHaveBeenCalledOnce();
	const value = String(warning.mock.calls[0][0]);
	expect(value).toContain('Expected browser failure');
	expect(value).toContain('network_unavailable');
	expect(value).not.toContain('PRIVATE_CANARY');
});
