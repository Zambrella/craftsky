import { redirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import {
	device,
	prefix,
	options,
	encode,
	sameState,
	readIntent,
	handoffResult,
	credentials,
	capability,
	type Pending
} from '#lib/server/auth/session.ts';
import { upstream, exactQuery } from '#lib/server/auth/upstream.ts';
export const GET: RequestHandler = async ({ url, cookies, request }) => {
	const intent = readIntent(cookies.get(prefix(url) + 'intent'));
	if (
		(request.headers.get('sec-fetch-site') === 'cross-site' &&
			request.headers.get('sec-fetch-mode') !== 'navigate') ||
		!intent ||
		!exactQuery(url, ['state', 'code', 'error']) ||
		!sameState(intent.state, url.searchParams.get('state') || undefined)
	)
		redirect(303, '/sign-in?error=invalid_intent');
	if (url.searchParams.has('error')) {
		if (url.searchParams.has('code'))
			redirect(303, '/sign-in?error=invalid_handoff');
		cookies.delete(prefix(url) + 'intent', { path: '/' });
		redirect(303, '/sign-in?error=authorization_canceled');
	}
	const code = url.searchParams.get('code');
	if (!capability(code)) redirect(303, '/sign-in?error=invalid_handoff');
	try {
		const result = handoffResult(
			await upstream('/v1/auth/handoffs/exchange', device(cookies, url), {
				code
			})
		);
		if (
			!result ||
			(intent.kind === 'chat' && result.purpose !== 'chat') ||
			(intent.kind !== 'chat' && result.purpose !== undefined)
		)
			throw new Error('Invalid handoff');
		if (
			intent.kind === 'chat' &&
			!credentials(cookies, url).some(
				(account) => account.id === intent.source && account.did === result.did
			)
		)
			throw new Error('Messaging source unavailable');
		const pending: Pending = {
			...result,
			active: intent.active,
			returnTo: intent.returnTo,
			kind: intent.kind,
			...(intent.source ? { source: intent.source } : {})
		};
		cookies.set(prefix(url) + 'pending', encode(pending), options(url, 600));
		cookies.delete(prefix(url) + 'intent', { path: '/' });
	} catch {
		redirect(303, '/sign-in?error=handoff_unavailable');
	}
	// The pending cookie must survive its own response before activation is confirmed.
	redirect(303, '/auth/browser/confirm');
};
