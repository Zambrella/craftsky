import { afterEach, describe, it, expect } from 'vitest';
import * as sdk from '@sentry/browser';
import { serializeEnvelope, type Envelope, type Transport } from '@sentry/core';
import { APIError } from '#lib/api/client.ts';
import { diagnosticOptions } from './options.ts';
import { safeEvent, privateTransport } from './privacy.ts';
const canary = 'PROTECTED_PRIVATE_CANARY_123';
const makeError = () => {
	const cause = new RangeError(canary);
	cause.stack = `RangeError: ${canary}\n at failure (https://app.craftsky.social/_app/immutable/chunks/cause.ABC123.js?token=${canary}:8:2)`;
	const error = new TypeError(canary, { cause });
	error.stack = `TypeError: ${canary}\n at failure (https://app.craftsky.social/_app/immutable/chunks/view.XYZ123.js?proof=${canary}:12:4)`;
	return error;
};
afterEach(async () => {
	await sdk.close(1);
	sdk.getCurrentScope().clearAttachments();
	sdk.getCurrentScope().clearBreadcrumbs();
	sdk.getCurrentScope().setUser(null);
});
describe('diagnostic source selection and serialization', () => {
	it('retains release, debug IDs, handled state and AppView correlation in serialized SDK output', async () => {
		const bodies: string[] = [];
		const client = sdk.init(
			diagnosticOptions(
				sdk,
				'https://public@o1.ingest.sentry.io/1',
				() => true,
				async (_request, init) => {
					bodies.push(String(init?.body));
					return new Response('', { status: 200 });
				},
				{ release: 'craftsky-web@1.1.1+abc123', environment: 'preview' }
			)
		)!;
		const id = '12345678-1234-4123-8123-123456789012';
		client.addEventProcessor((event) => ({
			...event,
			debug_meta: {
				images: [
					{
						type: 'sourcemap',
						debug_id: id,
						code_file: `https://app.craftsky.social/_app/immutable/chunks/view.js?token=${canary}`
					},
					{
						type: 'sourcemap',
						debug_id: id,
						code_file: `https://private.example/${canary}`
					}
				]
			}
		}));
		const error = new APIError(
			503,
			'temporarily_unavailable',
			canary,
			id,
			{ draft: canary },
			undefined,
			{ cause: makeError() }
		);
		sdk.captureException(error, {
			contexts: { diagnostic: { operation: 'workspace', recipient: canary } }
		});
		await client.flush(1000);
		expect(bodies).toHaveLength(1);
		const wire = bodies[0];
		expect(wire).not.toContain(canary);
		expect(wire).toContain('craftsky-web@1.1.1+abc123');
		expect(wire).toContain('"environment":"preview"');
		expect(wire).toContain('"httpStatus":503');
		expect(wire).toContain('"errorCode":"temporarily_unavailable"');
		expect(wire).toContain('"appViewRequestId":"' + id + '"');
		expect(wire).toContain('"debug_id":"' + id + '"');
		expect(wire).toContain('"handled":true');
		expect(wire).toContain('APIError');
		expect(wire).toContain('RangeError');
	});
	it('preserves native cause types and code coordinates in actual SDK wire output while dropping private canaries', async () => {
		const bodies: string[] = [];
		const fetcher: typeof fetch = async (_request, init) => {
			const body = init?.body;
			bodies.push(
				typeof body === 'string'
					? body
					: new TextDecoder().decode(body as Uint8Array)
			);
			return new Response('', { status: 200 });
		};
		const client = sdk.init(
			diagnosticOptions(
				sdk,
				'https://public@o1.ingest.sentry.io/1',
				() => true,
				fetcher
			)
		)!;
		sdk.setUser({ id: canary, email: canary });
		sdk.setExtra('draft', canary);
		sdk.setTag('recipient', canary);
		sdk.addBreadcrumb({ message: canary, data: { token: canary } });
		sdk.getCurrentScope().addAttachment({ filename: canary, data: canary });
		client.addEventProcessor((event) => ({
			...event,
			request: {
				url: `https://app.craftsky.social/app/messages/${canary}?proof=${canary}`,
				headers: { cookie: canary },
				data: canary
			},
			contexts: { private: { value: canary } },
			transaction: canary
		}));
		sdk.captureException(makeError());
		await client.flush(1000);
		expect(bodies).toHaveLength(1);
		const wire = bodies[0];
		expect(wire).not.toContain(canary);
		expect(wire).not.toContain('https://public@');
		expect(wire).not.toContain('attachment');
		expect(wire).not.toContain('breadcrumbs');
		expect(wire).toContain('TypeError');
		expect(wire).toContain('RangeError');
		expect(wire).toContain('/_app/immutable/chunks/view.XYZ123.js');
		expect(wire).toContain('"lineno":12');
		expect(wire).toContain('"colno":4');
	});
	it('suppresses expected API/offline/cancellation failures before the transport', async () => {
		const sent: string[] = [];
		const client = sdk.init(
			diagnosticOptions(
				sdk,
				'https://public@o1.ingest.sentry.io/1',
				() => true,
				async (_request, init) => {
					sent.push(String(init?.body));
					return new Response('', { status: 200 });
				}
			)
		)!;
		sdk.captureException(new APIError(403, 'not_allowed', canary));
		sdk.captureException(new TypeError('Failed to fetch'));
		sdk.captureException(new DOMException(canary, 'AbortError'));
		await client.flush(1000);
		expect(sent).toEqual([]);
	});
	it('blocks both new events and queued envelopes after consent is revoked', async () => {
		let enabled = false;
		const event = {
			event_id: 'a'.repeat(32),
			exception: { values: [{ type: 'TypeError', value: canary }] }
		};
		expect(safeEvent(event, {}, enabled)).toBeNull();
		const sent: Envelope[] = [];
		const inner: Transport = {
			send: async (envelope) => {
				sent.push(envelope);
				return { statusCode: 200 };
			},
			flush: async () => true
		};
		const transport = privateTransport(inner, () => enabled);
		await transport.send([
			{ event_id: 'a'.repeat(32), sent_at: new Date().toISOString() },
			[[{ type: 'event' }, event]]
		]);
		expect(sent).toEqual([]);
		enabled = true;
		await transport.send([
			{ event_id: 'a'.repeat(32), sent_at: new Date().toISOString() },
			[[{ type: 'event' }, event]]
		]);
		expect(sent).toHaveLength(1);
		enabled = false;
		await transport.send([
			{ event_id: 'a'.repeat(32), sent_at: new Date().toISOString() },
			[[{ type: 'event' }, event]]
		]);
		expect(sent).toHaveLength(1);
	});
	it('drops unknown event fields, external paths and frame source/locals rather than scrubbing them', async () => {
		const event = {
			event_id: 'b'.repeat(32),
			exception: {
				values: [
					{
						type: canary,
						value: canary,
						stacktrace: {
							frames: [
								{
									filename: `file:///Users/${canary}/secret.js`,
									lineno: 42,
									context_line: canary,
									vars: { secret: canary }
								},
								{
									filename: `https://evil.example/_app/immutable/chunks/${canary}.js`,
									lineno: 9
								}
							]
						}
					}
				]
			},
			user: { email: canary },
			extra: { payload: canary }
		};
		const result = safeEvent(event)!;
		expect(JSON.stringify(result)).not.toContain(canary);
		expect(result.exception?.values?.[0].type).toBe('Error');
		expect(result.exception?.values?.[0].stacktrace?.frames?.[0]).toEqual({
			filename: '[redacted]',
			lineno: 42
		});
		let wire: string | Uint8Array = '';
		const inner: Transport = {
			send: async (envelope) => {
				wire = serializeEnvelope(envelope);
				return { statusCode: 200 };
			},
			flush: async () => true
		};
		await privateTransport(inner, () => true).send([
			{
				event_id: 'b'.repeat(32),
				sent_at: new Date().toISOString(),
				dsn: canary,
				trace: { private: canary }
			},
			[
				[{ type: 'event' }, event],
				[
					{ type: 'attachment', filename: canary, length: canary.length },
					canary
				]
			]
		] as unknown as Envelope);
		expect(
			typeof wire === 'string' ? wire : new TextDecoder().decode(wire)
		).not.toContain(canary);
	});
});
