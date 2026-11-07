import { logServerFailure } from '../diagnostics';
import { redirect } from '@sveltejs/kit';
import { env } from 'node:process';
import type { RequestEvent } from '@sveltejs/kit';
import {
	selected,
	credentials,
	publicAccount,
	device,
	remove
} from '#lib/server/auth/session.ts';
import { guestRoute } from '#lib/routes.ts';
import { upstream, UpstreamError } from '#lib/server/auth/upstream.ts';
export async function authenticatedPage({
	cookies,
	url
}: Pick<RequestEvent, 'cookies' | 'url'>) {
	const signInLocation = (expired = false) => {
		const query = new URLSearchParams();
		if (guestRoute(url.pathname))
			query.set('returnTo', url.pathname + url.search);
		if (expired) query.set('error', 'session_expired');
		return '/sign-in' + (query.size ? '?' + query : '');
	};
	const account = selected(cookies, url);
	if (!account) redirect(303, signInLocation());
	try {
		const identity = (await upstream(
			'/v1/whoami',
			device(cookies, url),
			undefined,
			account.token
		)) as { did: string; handle: string };
		if (identity.did !== account.did) {
			remove(cookies, url, account.id);
			redirect(303, signInLocation(true));
		}
		if (
			typeof identity.handle === 'string' &&
			identity.handle !== account.handle
		) {
			account.handle = identity.handle;
		}
	} catch (error) {
		if (error instanceof UpstreamError && error.status === 401) {
			remove(cookies, url, account.id);
			redirect(303, signInLocation(true));
		}
		if (
			error &&
			typeof error === 'object' &&
			'status' in error &&
			error.status === 303
		)
			throw error;
		logServerFailure(
			error,
			'auth.session',
			'fetch',
			error instanceof UpstreamError
				? { status: error.status, code: error.code, requestId: error.requestId }
				: {}
		);
	}
	return {
		subscriptionsEnabled: env.SUBSCRIPTIONS_ENABLED === 'true',
		session: publicAccount(account),
		accounts: credentials(cookies, url).map(publicAccount),
		path: url.pathname.slice(1).split('/').map(decodeURIComponent)
	};
}
