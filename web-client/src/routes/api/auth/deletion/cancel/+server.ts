import type { RequestHandler } from './$types';
import {
	prefix,
	selected,
	device,
	readDeletionIntent,
	accountID
} from '#lib/server/auth/session.ts';
import {
	requireSameOrigin,
	upstream,
	safeFailure,
	input,
	exactQuery,
	UpstreamError
} from '#lib/server/auth/upstream.ts';
export const POST: RequestHandler = async ({ request, url, cookies }) => {
	if (!requireSameOrigin(request, url))
		return new Response(null, { status: 403 });
	try {
		if (!exactQuery(url, [])) throw new UpstreamError(400, 'invalid_request');
		const value = await input(request, ['jobId']);
		if (!accountID(value.jobId))
			throw new UpstreamError(400, 'invalid_request');
		const account = selected(
			cookies,
			url,
			request.headers.get('X-Craftsky-Account-Id') ?? undefined
		);
		if (!account) throw new UpstreamError(401, 'authentication_required');
		const pending =
			readDeletionIntent(cookies.get(prefix(url) + 'deletion_pending'), true) ||
			readDeletionIntent(cookies.get(prefix(url) + 'deletion_intent'));
		if (
			!pending ||
			pending.accountId !== account.id ||
			pending.did !== account.did ||
			pending.jobId !== value.jobId
		)
			throw new UpstreamError(400, 'invalid_confirmation');
		await upstream(
			'/v1/account-deletion/intents/' + encodeURIComponent(value.jobId),
			device(cookies, url),
			undefined,
			account.token,
			'DELETE'
		);
		for (const suffix of ['deletion_pending', 'deletion_intent'])
			cookies.delete(prefix(url) + suffix, { path: '/' });
		return new Response(null, { status: 204 });
	} catch (error) {
		return safeFailure(error);
	}
};
