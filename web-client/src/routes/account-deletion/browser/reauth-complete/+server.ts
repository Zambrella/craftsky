import { redirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import {
	prefix,
	encode,
	options,
	sameState,
	readDeletionIntent,
	capability,
	credentials
} from '#lib/server/auth/session.ts';
import { exactQuery } from '#lib/server/auth/upstream.ts';
export const GET: RequestHandler = ({ url, cookies }) => {
	const intent = readDeletionIntent(
		cookies.get(prefix(url) + 'deletion_intent')
	);
	const proof = url.searchParams.get('proof'),
		jobId = url.searchParams.get('job-id');
	if (
		!intent ||
		!exactQuery(url, ['state', 'job-id', 'proof']) ||
		!sameState(intent.state, url.searchParams.get('state') || undefined) ||
		intent.jobId !== jobId ||
		!capability(proof) ||
		!credentials(cookies, url).some(
			(account) => account.id === intent.accountId && account.did === intent.did
		)
	)
		redirect(303, '/app/settings/account?deletion=failed');
	cookies.set(
		prefix(url) + 'deletion_pending',
		encode({ ...intent, proof }),
		options(
			url,
			Math.max(1, Math.floor((intent.expiresAt - Date.now()) / 1000))
		)
	);
	cookies.delete(prefix(url) + 'deletion_intent', { path: '/' });
	redirect(303, '/app/settings/account?deletion=ready');
};
