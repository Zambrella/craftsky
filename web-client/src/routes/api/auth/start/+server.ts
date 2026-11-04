import type { RequestHandler } from './$types';
import {
	device,
	selected,
	nonce,
	prefix,
	encode,
	options,
	returnPath,
	credentials,
	type Intent
} from '#lib/server/auth/session.ts';
import {
	requireSameOrigin,
	upstream,
	safeFailure,
	input,
	exactQuery,
	authorizationURL,
	UpstreamError
} from '#lib/server/auth/upstream.ts';
export const POST: RequestHandler = async ({ request, url, cookies }) => {
	if (!requireSameOrigin(request, url))
		return new Response(null, { status: 403 });
	try {
		if (!exactQuery(url, [])) throw new UpstreamError(400, 'invalid_request');
		const value = await input(request, ['handle', 'kind', 'returnTo']);
		if (
			!['login', 'register', 'chat'].includes(String(value.kind)) ||
			(value.returnTo !== undefined && typeof value.returnTo !== 'string') ||
			(value.kind === 'login' &&
				(typeof value.handle !== 'string' || value.handle.length > 253))
		)
			throw new UpstreamError(400, 'invalid_request');
		const kind = value.kind as Intent['kind'];
		const explicit = request.headers.get('X-Craftsky-Account-Id');
		const account = selected(
			cookies,
			url,
			explicit === null ? undefined : explicit
		);
		if ((explicit !== null && !account) || (kind === 'chat' && !account))
			throw new UpstreamError(401, 'authentication_required');
		const list = credentials(cookies, url);
		if (
			kind !== 'chat' &&
			list.length >= 8 &&
			!list.some((item) => item.handle === String(value.handle).trim())
		)
			throw new UpstreamError(409, 'account_limit');
		const state = nonce();
		const callback = new URL('/auth/browser/complete', url.origin);
		callback.searchParams.set('state', state);
		const intent: Intent = {
			state,
			active: selected(cookies, url)?.id || '',
			kind,
			returnTo: returnPath(value.returnTo),
			expiresAt: Date.now() + 600000,
			...(kind === 'chat' ? { source: account!.id } : {})
		};
		if (kind === 'chat') intent.active = account!.id;
		const body = {
			handoffMode: 'browser',
			browserRedirectUri: callback.href,
			...(kind === 'login' ? { handle: String(value.handle).trim() } : {})
		};
		const result = (await upstream(
			kind === 'chat'
				? '/v1/chat/authorization'
				: kind === 'register'
					? '/v1/auth/registrations'
					: '/v1/auth/login',
			device(cookies, url),
			body,
			kind === 'chat' ? account?.token : undefined
		)) as { authUrl?: unknown };
		const authUrl = authorizationURL(result?.authUrl, url.protocol === 'http:');
		cookies.set(prefix(url) + 'intent', encode(intent), options(url, 600));
		return Response.json(
			{ authUrl },
			{ headers: { 'cache-control': 'no-store' } }
		);
	} catch (error) {
		return safeFailure(error);
	}
};
