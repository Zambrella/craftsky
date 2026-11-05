import type { RequestHandler } from './$types';
import {
	device,
	selected,
	nonce,
	prefix,
	encode,
	options,
	accountID
} from '#lib/server/auth/session.ts';
import {
	requireSameOrigin,
	upstream,
	safeFailure,
	input,
	exactQuery,
	authorizationURL,
	UpstreamError
} from '#lib/server/auth/upstream.ts';
export const POST: RequestHandler = async ({ request, url, cookies }) => {
	if (!requireSameOrigin(request, url))
		return new Response(null, { status: 403 });
	try {
		if (!exactQuery(url, [])) throw new UpstreamError(400, 'invalid_request');
		await input(request, []);
		const account = selected(
			cookies,
			url,
			request.headers.get('X-Craftsky-Account-Id') ?? undefined
		);
		if (!account) throw new UpstreamError(401, 'authentication_required');
		const state = nonce();
		const callback = new URL(
			'/account-deletion/browser/reauth-complete',
			url.origin
		);
		callback.searchParams.set('state', state);
		const result = (await upstream(
			'/v1/account-deletion/intents',
			device(cookies, url),
			{ browserRedirectUri: callback.href },
			account.token
		)) as {
			jobId: string;
			authUrl: string;
			confirmationDid: string;
			expiresAt: string;
			warning?: unknown;
		};
		const expiresAt = Date.parse(result?.expiresAt);
		if (
			!accountID(result?.jobId) ||
			result.confirmationDid !== account.did ||
			!Number.isFinite(expiresAt) ||
			expiresAt <= Date.now()
		)
			throw new UpstreamError(502, 'invalid_deletion_response');
		const authUrl = authorizationURL(result.authUrl, url.protocol === 'http:');
		cookies.set(
			prefix(url) + 'deletion_intent',
			encode({
				state,
				accountId: account.id,
				did: account.did,
				jobId: result.jobId,
				expiresAt: Math.min(expiresAt, Date.now() + 600000)
			}),
			options(url, 600)
		);
		return Response.json(
			{
				jobId: result.jobId,
				authUrl,
				confirmationDid: result.confirmationDid,
				expiresAt: result.expiresAt,
				...(result.warning &&
				typeof result.warning === 'object' &&
				(result.warning as { code?: unknown }).code ===
					'provider_billing_not_canceled'
					? {
							warning: {
								code: 'provider_billing_not_canceled',
								message:
									'Deleting your CraftSky account does not cancel provider billing and may end CraftSky access.'
							}
						}
					: {})
			},
			{ headers: { 'cache-control': 'no-store' } }
		);
	} catch (error) {
		return safeFailure(error);
	}
};
