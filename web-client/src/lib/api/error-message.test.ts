import { describe, it, expect } from 'vitest';
import {
	APIError,
	APITransport,
	MutationOutcomeUnknownError
} from './transport';
import { apiErrorMessage } from './error-message';
import {
	safeEvent,
	failureContext,
	expectedFailure
} from '../features/diagnostics/privacy';
import { FieldValidationError } from '../field-validation';
const id = '12345678-1234-4123-8123-123456789012';
const secret = 'PRIVATE_CANARY';

describe('error parity and diagnostic evidence', () => {
	it.each([
		[
			0,
			'network_unavailable',
			"You're offline. Check your connection and try again."
		],
		[401, 'unauthorized', 'Please sign in again.'],
		[403, 'forbidden', "You don't have permission to do that."],
		[404, 'not_found', 'That content is no longer available.'],
		[
			503,
			'internal_error',
			'CraftSky is having trouble right now. Please try again.'
		],
		[200, 'invalid_response', 'Something went wrong. Please try again.'],
		[400, 'bad_request', "That didn't work. Please try again."]
	] as const)(
		'maps %s/%s to client-owned Flutter copy',
		(status, code, message) => {
			expect(apiErrorMessage(new APIError(status, code, secret))).toBe(message);
		}
	);
	it('retains preclassified adapter failures rather than calling them offline', async () => {
		const failure = new APIError(
			200,
			'invalid_response',
			secret,
			id,
			undefined,
			undefined,
			{ cause: new SyntaxError(secret) }
		);
		failure.responseKind = 'nonJson';
		const transport = new APITransport(async () => {
			throw failure;
		});
		await expect(transport.get('/private')).rejects.toBe(failure);
		expect(failure.httpMethod).toBe('GET');
		expect(expectedFailure(failure)).toBe(false);
		expect(failureContext(failure, 'workspace')).toMatchObject({
			httpStatus: 200,
			appViewRequestId: id,
			responseKind: 'nonJson',
			errorCode: 'invalid_response'
		});
	});
	it('retains response evidence when a page shape is malformed', async () => {
		const transport = new APITransport(async () =>
			Response.json(
				{ items: secret, cursor: 123 },
				{ headers: { 'X-Request-ID': id } }
			)
		);
		const failure = await transport.page('/private').catch((error) => error);
		expect(failureContext(failure, 'workspace')).toEqual({
			operation: 'workspace',
			httpStatus: 200,
			httpMethod: 'GET',
			responseKind: 'json',
			errorCode: 'invalid_response',
			appViewRequestId: id
		});
		expect(expectedFailure(failure)).toBe(false);
		expect(apiErrorMessage(failure)).not.toContain(secret);
	});
	it('preserves causes for offline reads and uncertain writes without changing retry semantics', async () => {
		const cause = new TypeError(secret);
		const transport = new APITransport(async () => {
			throw cause;
		});
		await expect(transport.get('/private')).rejects.toMatchObject({
			cause,
			httpMethod: 'GET',
			code: 'network_unavailable'
		});
		await expect(
			transport.request('POST', '/private', { operationKey: id })
		).rejects.toMatchObject({ cause, operationKey: id, httpMethod: 'POST' });
		expect(expectedFailure(new MutationOutcomeUnknownError(id))).toBe(true);
		expect(expectedFailure(new FieldValidationError('private', secret))).toBe(
			true
		);
	});
	it('retains selected diagnostics through final serialization without private values', async () => {
		const transport = new APITransport(
			async () =>
				new Response(
					JSON.stringify({
						error: 'internal_error',
						message: secret,
						fields: { draft: secret }
					}),
					{ status: 503, headers: { 'X-Request-ID': id } }
				)
		);
		const error = await transport
			.get('/private?search=secret')
			.catch((error) => error);
		const event = safeEvent(
			{
				exception: { values: [{ type: 'APIError', value: secret }] },
				contexts: { diagnostic: failureContext(error, 'settings.load') }
			},
			{ originalException: error }
		);
		expect(event?.contexts?.diagnostic).toEqual({
			operation: 'settings.load',
			httpMethod: 'GET',
			httpStatus: 503,
			errorCode: 'internal_error',
			appViewRequestId: id,
			responseKind: 'appViewError'
		});
		expect(JSON.stringify(event)).not.toContain(secret);
	});
	it('rejects invalid request IDs and preserves parse causes', async () => {
		const transport = new APITransport(
			async () =>
				new Response(secret, {
					status: 502,
					headers: { 'X-Request-ID': secret }
				})
		);
		await expect(transport.get('/private')).rejects.toMatchObject({
			requestId: undefined,
			cause: expect.any(SyntaxError),
			responseKind: 'nonJson',
			httpMethod: 'GET'
		});
	});
});
