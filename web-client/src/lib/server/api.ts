import { error } from '@sveltejs/kit';
import { env } from 'node:process';

// Cloudflare implements manual redirects, not redirect:error. Non-2xx
// responses below are rejected without following the Location header.
// Only a configured AppView origin is contacted. Incoming headers, host, cookies,
// bearer tokens and arbitrary query values are never forwarded upstream.
export async function publicAPI<T>(
	fetcher: typeof fetch,
	path: string
): Promise<T> {
	let base: URL;
	try {
		base = new URL(env.APPVIEW_ORIGIN || 'https://appview.craftsky.social');
	} catch {
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
	)
		error(503, 'Page temporarily unavailable');
	let response: Response;
	try {
		response = await fetcher(new URL(`/v1/public/${path}`, base), {
			headers: { accept: 'application/json' },
			redirect: 'manual',
			credentials: 'omit',
			signal: AbortSignal.timeout(9000)
		});
	} catch {
		error(503, 'Page temporarily unavailable');
	}
	if (response.status === 404) error(404, 'Page not found');
	if (response.status === 400) error(400, 'Invalid page request');
	if (response.status === 429)
		error(429, 'Too many requests. Please try again shortly.');
	if (!response.ok) error(503, 'Page temporarily unavailable');
	// Bound both declared and streamed bodies; no public JSON can exhaust Worker memory.
	const reader = response.body?.getReader();
	if (!reader) error(503, 'Page temporarily unavailable');
	const chunks: Uint8Array[] = [];
	let length = 0;
	try {
		for (;;) {
			const { value, done } = await reader.read();
			if (done) break;
			length += value.length;
			if (length > 2 * 1024 * 1024) {
				await reader.cancel();
				error(503, 'Page temporarily unavailable');
			}
			chunks.push(value);
		}
		const body = new Uint8Array(length);
		let offset = 0;
		for (const chunk of chunks) {
			body.set(chunk, offset);
			offset += chunk.length;
		}
		return JSON.parse(new TextDecoder().decode(body)) as T;
	} catch {
		error(503, 'Page temporarily unavailable');
	} finally {
		reader.releaseLock();
	}
}
