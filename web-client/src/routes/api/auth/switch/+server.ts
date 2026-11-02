import type { RequestHandler } from './$types';
import {
	prefix,
	credentials,
	options,
	accountID
} from '#lib/server/auth/session.ts';
import {
	requireSameOrigin,
	input,
	safeFailure,
	exactQuery,
	UpstreamError
} from '#lib/server/auth/upstream.ts';
export const POST: RequestHandler = async ({ request, url, cookies }) => {
	if (!requireSameOrigin(request, url))
		return new Response(null, { status: 403 });
	try {
		if (!exactQuery(url, [])) throw new UpstreamError(400, 'invalid_request');
		const value = await input(request, ['accountId']);
		if (!accountID(value.accountId))
			throw new UpstreamError(400, 'invalid_request');
		if (
			!credentials(cookies, url).some(
				(account) => account.id === value.accountId
			)
		)
			return new Response(null, { status: 404 });
		cookies.set(prefix(url) + 'active', value.accountId, options(url));
		return new Response(null, { status: 204 });
	} catch (error) {
		return safeFailure(error);
	}
};
