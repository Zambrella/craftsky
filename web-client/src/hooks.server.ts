import { logServerFailure } from '#lib/server/diagnostics.ts';
import type { HandleServerError } from '@sveltejs/kit/hooks';
import type { Handle } from '@sveltejs/kit/hooks';
import login from '../../verified-links/auth/complete.html?raw';
import deletion from '../../verified-links/account-deletion/reauth-complete.html?raw';
import android from '../../verified-links/.well-known/assetlinks.json?raw';
import apple from '../../verified-links/.well-known/apple-app-site-association?raw';
const callbackPolicy =
	"default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'";
export const handle: Handle = async ({ event, resolve }) => {
	const path = event.url.pathname;
	let response: Response;
	if (
		path === '/auth/complete' ||
		path === '/account-deletion/reauth-complete'
	) {
		response = new Response(path === '/auth/complete' ? login : deletion, {
			headers: {
				'content-type': 'text/html; charset=utf-8',
				'content-security-policy': callbackPolicy
			}
		});
	} else if (
		path === '/.well-known/assetlinks.json' ||
		path === '/.well-known/apple-app-site-association'
	) {
		response = new Response(
			path.endsWith('assetlinks.json') ? android : apple,
			{
				headers: {
					'content-type': 'application/json; charset=utf-8',
					'cache-control': 'public, max-age=3600'
				}
			}
		);
	} else {
		response = await resolve(event);
	}
	// Never retain public content after a moderation/deletion change. OAuth query
	// values must never enter a cache, canonical tag, referrer or reflected page.
	if (!path.startsWith('/.well-known/'))
		response.headers.set('cache-control', 'no-store');
	response.headers.set('referrer-policy', 'no-referrer');
	response.headers.set('x-content-type-options', 'nosniff');
	response.headers.set('x-frame-options', 'DENY');
	response.headers.set(
		'permissions-policy',
		'camera=(), microphone=(), geolocation=()'
	);
	if (event.url.hostname !== 'app.craftsky.social')
		response.headers.set('x-robots-tag', 'noindex, nofollow');
	if (response.status >= 400)
		response.headers.set('x-robots-tag', 'noindex, nofollow');
	if (response.status === 429 || response.status === 503)
		response.headers.set('retry-after', '1');
	return response;
};

export const handleError: HandleServerError = ({ error, kind }) => {
	if (kind !== 'unknown') return;
	logServerFailure(error, 'worker.unhandled', 'request');
	return { message: 'Something went wrong. Please try again.' };
};
