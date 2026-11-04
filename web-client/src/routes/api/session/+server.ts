import type { RequestHandler } from './$types';
import {
	credentials,
	selected,
	publicAccount
} from '#lib/server/auth/session.ts';
import { exactQuery } from '#lib/server/auth/upstream.ts';
export const GET: RequestHandler = ({ cookies, url, request }) => {
	if (
		!exactQuery(url, []) ||
		request.headers.get('sec-fetch-site') === 'cross-site'
	)
		return new Response(null, { status: 403 });
	return Response.json(
		{
			accounts: credentials(cookies, url).map(publicAccount),
			active: selected(cookies, url)?.id || null
		},
		{ headers: { 'cache-control': 'no-store' } }
	);
};
