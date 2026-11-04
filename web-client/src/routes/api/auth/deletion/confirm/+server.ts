import type { RequestHandler } from './$types';
import {
	prefix,
	selected,
	device,
	remove,
	readDeletionIntent,
	type DeletionPending
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
		const value = await input(request, ['jobId', 'confirmationDid']);
		const pending = readDeletionIntent(
			cookies.get(prefix(url) + 'deletion_pending'),
			true
		) as DeletionPending | undefined;
		const account = selected(
			cookies,
			url,
			request.headers.get('X-Craftsky-Account-Id') ?? undefined
		);
		if (
			!account ||
			!pending ||
			pending.accountId !== account.id ||
			pending.did !== account.did ||
			pending.did !== value.confirmationDid ||
			pending.jobId !== value.jobId
		)
			return Response.json(
				{
					error: 'invalid_confirmation',
					message: 'Confirm the exact account shown.'
				},
				{ status: 400 }
			);
		await upstream(
			'/v1/account-deletions/' + encodeURIComponent(pending.jobId),
			device(cookies, url),
			{ reauthProof: pending.proof, confirmationDid: pending.did },
			account.token
		);
		remove(cookies, url, account.id);
		cookies.delete(prefix(url) + 'deletion_pending', { path: '/' });
		return new Response(null, { status: 204 });
	} catch (error) {
		return safeFailure(error);
	}
};
