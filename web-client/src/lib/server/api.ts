import { logServerFailure } from './diagnostics';
import { error } from '@sveltejs/kit';
import { env } from 'node:process';

// Cloudflare implements manual redirects, not redirect:error. Non-2xx
// responses below are rejected without following the Location header.
// Only a configured AppView origin is contacted. Incoming headers, host, cookies,
// bearer tokens and arbitrary query values are never forwarded upstream.
export async function publicAPI<T>(
	fetcher: typeof fetch,
	path: string,
	format: 'json' | 'caption' = 'json'
): Promise<T> {
	let base: URL;
	try {
		base = new URL(env.APPVIEW_ORIGIN || 'https://appview.craftsky.social');
	} catch (cause) {
		logServerFailure(cause, 'public.read', 'configuration');
		error(503, 'Page temporarily unavailable');
	}
	if (
		base.username ||
		base.password ||
		base.pathname !== '/' ||
		base.search ||
		base.hash ||
		(base.protocol !== 'https:' &&
			!(
				base.protocol === 'http:' &&
				['localhost', '127.0.0.1'].includes(base.hostname)
			))
	) {
		logServerFailure(
			new Error('Invalid origin'),
			'public.read',
			'configuration'
		);
		error(503, 'Page temporarily unavailable');
	}
	let response: Response;
	try {
		response = await fetcher(new URL(`/v1/public/${path}`, base), {
			headers: {
				accept: format === 'caption' ? 'text/vtt' : 'application/json'
			},
			redirect: 'manual',
			credentials: 'omit',
			signal: AbortSignal.timeout(9000)
		});
	} catch (cause) {
		logServerFailure(cause, 'public.read', 'fetch');
		error(503, 'Page temporarily unavailable');
	}
	// Rejected payloads are never read. Release their network resources without
	// delaying the existing status mapping if upstream cancellation fails.
	if (!response.ok) void response.body?.cancel().catch(() => {});
	if (response.status === 403) error(403, 'Sign in to view this content.');
	if (response.status === 404) error(404, 'Page not found');
	if (response.status === 400) error(400, 'Invalid page request');
	if (response.status === 429)
		error(429, 'Too many requests. Please try again shortly.');
	if (!response.ok) {
		logServerFailure(
			new Error('Upstream response failed'),
			'public.read',
			'response',
			{
				status: response.status,
				requestId: response.headers.get('X-Request-ID') || undefined
			}
		);
		error(503, 'Page temporarily unavailable');
	}
	// Bound streamed bytes, including responses without a Content-Length header.
	const reader = response.body?.getReader();
	if (!reader) {
		logServerFailure(new Error('Missing body'), 'public.read', 'response');
		error(503, 'Page temporarily unavailable');
	}
	const decoder = new TextDecoder();
	let text = '';
	let length = 0;
	try {
		for (;;) {
			const { value, done } = await reader.read();
			if (done) break;
			length += value.length;
			if (length > (format === 'caption' ? 20000 : 2 * 1024 * 1024)) {
				await reader.cancel();
				error(503, 'Page temporarily unavailable');
			}
			text += decoder.decode(value, { stream: true });
		}
		text += decoder.decode();
		if (format === 'caption') {
			if (!/^WEBVTT(?:[ \t\r\n]|$)/.test(text))
				error(503, 'Page temporarily unavailable');
			return text as T;
		}
		return JSON.parse(text) as T;
	} catch (cause) {
		logServerFailure(cause, 'public.read', 'decode', {
			requestId: response.headers.get('X-Request-ID') || undefined
		});

		error(503, 'Page temporarily unavailable');
	} finally {
		reader.releaseLock();
	}
}
