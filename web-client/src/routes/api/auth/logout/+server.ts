import type { RequestHandler } from './$types';
import {
	device,
	selected,
	remove,
	accountID
} from '#lib/server/auth/session.ts';
import {
	requireSameOrigin,
	upstream,
	safeFailure,
	UpstreamError,
	input,
	exactQuery
} from '#lib/server/auth/upstream.ts';
export const POST: RequestHandler = async ({ request, url, cookies }) => {
	if (!requireSameOrigin(request, url))
		return new Response(null, { status: 403 });
	try {
		if (!exactQuery(url, [])) throw new UpstreamError(400, 'invalid_request');
		const value = await input(request, ['accountId']);
		if (!accountID(value.accountId))
			throw new UpstreamError(400, 'invalid_request');
		const account = selected(cookies, url, value.accountId);
		if (account) {
			try {
				await upstream(
					'/v1/auth/logout',
					device(cookies, url),
					undefined,
					account.token,
					'POST'
				);
			} catch (error) {
				if (!(error instanceof UpstreamError && error.status === 401))
					return safeFailure(error);
			}
			remove(cookies, url, account.id);
		}
		return new Response(null, { status: 204 });
	} catch (error) {
		return safeFailure(error);
	}
};
