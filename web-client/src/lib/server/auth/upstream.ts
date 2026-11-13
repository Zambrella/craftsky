import { logServerFailure, type ServerOperation } from '../diagnostics';
import { env } from 'node:process';
import { appViewRequestId } from '#lib/api/request-id.ts';
export class UpstreamError extends Error {
	readonly name = 'UpstreamError';
	requestId?: string;
	constructor(
		public status: number,
		public code: string,
		public retryAfter?: string,
		options?: ErrorOptions
	) {
		super('CraftSky request failed', options);
	}
}
export function appviewOrigin() {
	const value = new URL(
		env.APPVIEW_ORIGIN || 'https://appview.craftsky.social'
	);
	if (
		value.username ||
		value.password ||
		value.pathname !== '/' ||
		value.search ||
		value.hash ||
		(value.protocol !== 'https:' &&
			!(
				value.protocol === 'http:' &&
				['127.0.0.1', 'localhost'].includes(value.hostname)
			))
	)
		throw new UpstreamError(503, 'configuration_unavailable');
	return value.origin;
}
async function boundedText(
	body: ReadableStream<Uint8Array> | null,
	limit: number
): Promise<string> {
	if (!body) return '';
	const reader = body.getReader();
	const decoder = new TextDecoder('utf-8', { fatal: true });
	let length = 0,
		text = '';
	try {
		for (;;) {
			const next = await reader.read();
			if (next.done) break;
			length += next.value.length;
			if (length > limit) {
				await reader.cancel();
				throw new UpstreamError(413, 'body_too_large');
			}
			text += decoder.decode(next.value, { stream: true });
		}
		return text + decoder.decode();
	} finally {
		reader.releaseLock();
	}
}
export async function input(
	request: Request,
	allowed: string[],
	limit = 8192
): Promise<Record<string, unknown>> {
	if (
		request.headers.get('content-type')?.split(';')[0].trim().toLowerCase() !==
		'application/json'
	)
		throw new UpstreamError(400, 'invalid_request');
	try {
		const value: unknown = JSON.parse(await boundedText(request.body, limit));
		if (
			!value ||
			typeof value !== 'object' ||
			Array.isArray(value) ||
			Object.keys(value).some((key) => !allowed.includes(key))
		)
			throw new Error('Invalid request');
		return value as Record<string, unknown>;
	} catch (error) {
		if (error instanceof UpstreamError) throw error;
		throw new UpstreamError(400, 'invalid_request');
	}
}
export function exactQuery(url: URL, allowed: string[]): boolean {
	return [...url.searchParams.keys()].every(
		(key) => allowed.includes(key) && url.searchParams.getAll(key).length === 1
	);
}
export function authorizationURL(raw: unknown, local = false): string {
	if (typeof raw !== 'string' || raw.length > 8192)
		throw new UpstreamError(502, 'invalid_authorization_response');
	const target = new URL(raw);
	if (
		target.username ||
		target.password ||
		target.hash ||
		(target.protocol !== 'https:' &&
			!(
				local &&
				target.protocol === 'http:' &&
				['127.0.0.1', 'localhost'].includes(target.hostname)
			))
	)
		throw new UpstreamError(502, 'invalid_authorization_response');
	return target.href;
}
export async function upstream(
	path: string,
	deviceId: string,
	body?: unknown,
	token?: string,
	method = body === undefined ? 'GET' : 'POST'
) {
	try {
		const response = await fetch(appviewOrigin() + path, {
			method,
			headers: {
				accept: 'application/json',
				'content-type': 'application/json',
				'X-Craftsky-Device-Id': deviceId,
				...(token ? { authorization: 'Bearer ' + token } : {})
			},
			...(body !== undefined ? { body: JSON.stringify(body) } : {}),
			redirect: 'manual',
			signal: AbortSignal.timeout(20000)
		});
		if (response.status >= 300 && response.status < 400)
			throw new UpstreamError(502, 'upstream_redirect');
		if (!response.ok) {
			let code = 'request_failed';
			let responseCause: unknown;
			try {
				const value = JSON.parse(await boundedText(response.body, 32768)) as {
					error?: string;
				};
				if (
					typeof value.error === 'string' &&
					/^[a-z0-9_]{1,80}$/.test(value.error)
				)
					code = value.error;
			} catch (cause) {
				responseCause = cause;
			}
			const retry = response.headers.get('retry-after');
			const failure = new UpstreamError(
				response.status,
				code,
				retry && /^\d{1,5}$/.test(retry) ? retry : undefined,
				{ cause: responseCause }
			);
			failure.requestId = response.headers.get('X-Request-ID') || undefined;
			throw failure;
		}
		if (response.status === 204) return undefined;
		try {
			const body = await boundedText(response.body, 1024 * 1024);
			return body.trim() ? JSON.parse(body) : undefined;
		} catch (error) {
			const failure = new UpstreamError(
				502,
				'invalid_upstream_response',
				undefined,
				{ cause: error }
			);
			failure.requestId = response.headers.get('X-Request-ID') || undefined;
			throw failure;
		}
	} catch (error) {
		if (error instanceof UpstreamError) throw error;
		throw new UpstreamError(503, 'connection_unavailable', undefined, {
			cause: error
		});
	}
}
export function safeFailure(
	error: unknown,
	operation: ServerOperation = 'auth.request'
) {
	const known = error instanceof UpstreamError;
	const status =
		known && error.status >= 400 && error.status <= 599 ? error.status : 503;
	const requestId = known ? appViewRequestId(error.requestId) : undefined;
	logServerFailure(
		error,
		operation,
		'request',
		known
			? { status: error.status, code: error.code, requestId: error.requestId }
			: {}
	);
	return Response.json(
		{
			error: known ? error.code : 'unavailable',
			message: 'This request could not be completed. Please try again.',
			...(requestId ? { requestId } : {})
		},
		{
			status,
			headers: {
				'cache-control': 'no-store',
				...(requestId ? { 'X-Request-ID': requestId } : {}),
				...(known && error.retryAfter
					? { 'retry-after': error.retryAfter }
					: {})
			}
		}
	);
}
export function requireSameOrigin(request: Request, url: URL) {
	return (
		request.headers.get('origin') === url.origin &&
		request.headers.get('sec-fetch-site') !== 'cross-site'
	);
}
