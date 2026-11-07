import { boundedBody, bodyLimit } from '#lib/server/auth/body.ts';
import { UpstreamError } from '#lib/server/auth/upstream.ts';
import type { RequestHandler } from './$types';
import { device, selected, remove } from '#lib/server/auth/session.ts';
import {
	appviewOrigin,
	requireSameOrigin,
	safeFailure
} from '#lib/server/auth/upstream.ts';
import { allowedRoute, safeProxyPath } from '#lib/server/auth/routes.ts';
const handler: RequestHandler = async ({ request, url, cookies, params }) => {
	const target = safeProxyPath(params.path);
	const decoded = '/' + params.path;
	if (
		!target ||
		!allowedRoute(request.method, decoded) ||
		url.search.length > 16384
	)
		return Response.json(
			{ error: 'not_found', message: 'Endpoint not found' },
			{ status: 404 }
		);
	if (
		!['GET', 'HEAD'].includes(request.method) &&
		!requireSameOrigin(request, url)
	)
		return Response.json(
			{ error: 'csrf_rejected', message: 'Refresh this page and try again.' },
			{ status: 403 }
		);
	if (request.headers.get('sec-fetch-site') === 'cross-site')
		return new Response(null, { status: 403 });
	const account = selected(
		cookies,
		url,
		request.headers.get('X-Craftsky-Account-Id') ?? undefined
	);
	if (!account)
		return Response.json(
			{ error: 'authentication_required', message: 'Sign in to continue.' },
			{ status: 401 }
		);
	const headers = new Headers({
		accept: 'application/json',
		authorization: 'Bearer ' + account.token,
		'X-Craftsky-Device-Id': device(cookies, url)
	});
	for (const key of ['content-type', 'if-match', 'idempotency-key']) {
		const value = request.headers.get(key);
		if (value) {
			if (value.length > 512) return new Response(null, { status: 400 });
			headers.set(key, value);
		}
	}
	try {
		const signal = AbortSignal.any([
			request.signal,
			AbortSignal.timeout(
				request.method === 'POST' &&
					/^\/v1\/blobs\/videos\/[0-9a-f-]{36}\/status$/.test(target)
					? 7 * 60 * 1000
					: 120000
			)
		]);
		const input = boundedBody(
			request,
			bodyLimit(request.method, decoded),
			signal
		);
		let response: Response;
		try {
			response = await fetch(appviewOrigin() + target + url.search, {
				method: request.method,
				headers,
				body: input.body,
				duplex: 'half',
				redirect: 'manual',
				signal
			} as RequestInit);
		} catch (error) {
			throw input.failure() ?? error;
		} finally {
			input.close();
		}

		if (response.status >= 300 && response.status < 400)
			return Response.json(
				{ error: 'upstream_redirect', message: 'Request unavailable.' },
				{ status: 502 }
			);
		if (response.status === 401) remove(cookies, url, account.id);
		const output = new Headers({ 'cache-control': 'no-store' });
		for (const key of [
			'content-type',
			'retry-after',
			'x-request-id',
			'etag',
			'content-disposition'
		]) {
			const value = response.headers.get(key);
			if (value) output.set(key, value);
		}
		return new Response(response.body, {
			status: response.status,
			headers: output
		});
	} catch (error) {
		return safeFailure(
			error instanceof DOMException && error.name === 'TimeoutError'
				? new UpstreamError(504, 'request_timeout', undefined, { cause: error })
				: error,
			'appview.proxy'
		);
	}
};
export {
	handler as GET,
	handler as POST,
	handler as PUT,
	handler as PATCH,
	handler as DELETE
};
