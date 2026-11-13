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
		[404, 'not_found', 'That content is no longer available.'],
		[
			503,
			'internal_error',
			'CraftSky is having trouble right now. Please try again.'
		],
		[200, 'invalid_response', 'Something went wrong. Please try again.'],
		[400, 'bad_request', "That didn't work. Please try again."],
		[
			503,
			'web_checkout_unavailable',
			'Web subscriptions are not available right now. Please try again later.'
		],
		[
			409,
			'subscription_reconciliation_required',
			'Refresh your subscription status before opening checkout.'
		],
		[
			409,
			'subscription_already_exists',
			'This account already has a subscription or a payment awaiting resolution. Refresh your subscription status.'
		],
		[
			409,
			'web_checkout_pending',
			'This account already has a checkout in progress. Continue with that plan or wait for the checkout to expire before choosing another.'
		],
		[
			503,
			'web_management_unavailable',
			'Subscription management is not available here yet. Use the management link in your Stripe receipt.'
		]
	] as const)(
		'maps %s/%s to client-owned Flutter copy',
		(status, code, message) => {
			expect(apiErrorMessage(new APIError(status, code, secret))).toBe(message);
		}
	);
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
