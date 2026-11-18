import { isHttpError, json } from '@sveltejs/kit';
import { publicReadPath } from '#lib/api/public-path.ts';
import { publicAPI } from './api';
import { appViewRequestId } from '#lib/api/request-id.ts';

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
	const publicFetch: typeof fetch = async (input, init) => {
		const response = await fetcher(input, init);
		const requestId = appViewRequestId(response.headers.get('X-Request-ID'));
		if (requestId) headers['X-Request-ID'] = requestId;
		return response;
	};
	try {
		if (/\/video-captions\/[^/?]+$/.test(canonical))
			return new Response(
				await publicAPI<string>(publicFetch, canonical, 'caption'),
				{ headers: { ...headers, 'content-type': 'text/vtt; charset=utf-8' } }
			);
		return json(await publicAPI(publicFetch, canonical), { headers });
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
			{
				error: code,
				message: error.body.message,
				...(headers['X-Request-ID']
					? { requestId: headers['X-Request-ID'] }
					: {})
			},
			{ status, headers }
		);
	}
}
