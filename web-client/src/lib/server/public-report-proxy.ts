import { json } from '@sveltejs/kit';
import { publicReportPath } from '#lib/api/public-report-path.ts';
import {
	input,
	requireSameOrigin,
	upstream,
	safeFailure,
	UpstreamError
} from './auth/upstream';

export async function publicReportResponse(
	request: Request,
	url: URL,
	path: string
): Promise<Response> {
	const headers = { 'cache-control': 'no-store' };
	if (!requireSameOrigin(request, url))
		return new Response(null, { status: 403, headers });
	try {
		let canonical: string;
		try {
			canonical = publicReportPath(path, url.searchParams);
		} catch {
			throw new UpstreamError(400, 'invalid_request');
		}
		const body = await input(request, [
			'reasonType',
			'details',
			'contactEmail'
		]);
		// No browser cookies, caller headers, credentials or account identity are forwarded.
		const result = await upstream('/v1/public/' + canonical, '', body);
		if (
			!result ||
			typeof result.reportId !== 'string' ||
			result.status !== 'accepted'
		)
			throw new UpstreamError(502, 'invalid_upstream_response');
		return json(
			{ reportId: result.reportId, status: result.status },
			{ status: 201, headers }
		);
	} catch (error) {
		return safeFailure(error);
	}
}
