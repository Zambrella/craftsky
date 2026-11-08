import { isHttpError, json } from '@sveltejs/kit';
import { publicReadPath } from '#lib/api/public-path.ts';
import { publicAPI } from './api';

/** No caller headers/cookies enter this boundary; publicAPI also caps the
 * upstream body, redirects and timeout. Private proxy access stays protected. */
export async function publicReadResponse(
	fetcher: typeof fetch,
	path: string,
	query: URLSearchParams
): Promise<Response> {
	const headers: Record<string, string> = { 'cache-control': 'no-store' };
	let canonical: string;
	try {
		canonical = publicReadPath(path, query);
	} catch {
		return json(
			{ error: 'invalid_request', message: 'Invalid public read request' },
			{ status: 400, headers }
		);
	}
	try {
		if (/\/video-captions\/[^/?]+$/.test(canonical))
			return new Response(
				await publicAPI<string>(fetcher, canonical, 'caption'),
				{ headers: { ...headers, 'content-type': 'text/vtt; charset=utf-8' } }
			);
		return json(await publicAPI(fetcher, canonical), { headers });
	} catch (error) {
		if (!isHttpError(error)) throw error;
		const status = error.status;
		if (status === 429 || status === 503) headers['retry-after'] = '1';
		const code =
			status === 403
				? 'authentication_required'
				: status === 404
					? 'not_found'
					: status === 400
						? 'invalid_request'
						: status === 429
							? 'rate_limited'
							: 'unavailable';
		return json(
			{ error: code, message: error.body.message },
			{ status, headers }
		);
	}
}
