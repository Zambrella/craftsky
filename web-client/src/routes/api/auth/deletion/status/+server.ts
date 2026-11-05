import type { RequestHandler } from './$types';
import {
	prefix,
	selected,
	readDeletionIntent
} from '#lib/server/auth/session.ts';
import { exactQuery } from '#lib/server/auth/upstream.ts';
export const GET: RequestHandler = ({ url, cookies, request }) => {
	if (
		!exactQuery(url, []) ||
		request.headers.get('sec-fetch-site') === 'cross-site'
	)
		return new Response(null, { status: 403 });
	const value = readDeletionIntent(
		cookies.get(prefix(url) + 'deletion_pending'),
		true
	);
	const account = selected(
		cookies,
		url,
		request.headers.get('X-Craftsky-Account-Id') ?? undefined
	);
	return Response.json(
		value && value.accountId === account?.id && value.did === account.did
			? { pending: true, jobId: value.jobId, did: value.did }
			: { pending: false },
		{ headers: { 'cache-control': 'no-store' } }
	);
};
