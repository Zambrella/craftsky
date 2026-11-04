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
		let body: Uint8Array | undefined;
		if (!['GET', 'HEAD'].includes(request.method) && request.body) {
			const reader = request.body.getReader();
			const chunks: Uint8Array[] = [];
			let length = 0;
			try {
				for (;;) {
					const next = await reader.read();
					if (next.done) break;
					length += next.value.length;
					if (length > 32 * 1024 * 1024) {
						await reader.cancel();
						return Response.json(
							{ error: 'body_too_large', message: 'This upload is too large.' },
							{ status: 413 }
						);
					}
					chunks.push(next.value);
				}
			} finally {
				reader.releaseLock();
			}
			body = new Uint8Array(length);
			let offset = 0;
			for (const chunk of chunks) {
				body.set(chunk, offset);
				offset += chunk.length;
			}
		}
		const response = await fetch(appviewOrigin() + target + url.search, {
			method: request.method,
			headers,
			body: body as BodyInit | undefined,
			redirect: 'manual',
			signal: AbortSignal.timeout(120000)
		});
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
		return safeFailure(error);
	}
};
export {
	handler as GET,
	handler as POST,
	handler as PUT,
	handler as PATCH,
	handler as DELETE
};
