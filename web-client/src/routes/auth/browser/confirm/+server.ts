import { logServerFailure } from '#lib/server/diagnostics.ts';
import { redirect, isRedirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import {
	device,
	prefix,
	credentials,
	store,
	readPending,
	sameState,
	returnPath
} from '#lib/server/auth/session.ts';
import {
	upstream,
	UpstreamError,
	exactQuery
} from '#lib/server/auth/upstream.ts';
import { Buffer } from 'node:buffer';
export const GET: RequestHandler = async ({ url, cookies, request }) => {
	const pending = readPending(cookies.get(prefix(url) + 'pending'));
	if (
		!pending ||
		!exactQuery(url, []) ||
		(request.headers.get('sec-fetch-site') === 'cross-site' &&
			request.headers.get('sec-fetch-mode') !== 'navigate')
	)
		redirect(303, '/sign-in?error=invalid_handoff');
	try {
		if (pending.kind === 'chat') {
			const source = credentials(cookies, url).find(
				(item) => item.id === pending.source
			);
			const hash = source
				? Buffer.from(
						await crypto.subtle.digest(
							'SHA-256',
							new TextEncoder().encode(source.token)
						)
					).toString('base64url')
				: '';
			if (
				!source ||
				source.did !== pending.did ||
				!sameState(hash, pending.sourceTokenHash)
			) {
				cookies.delete(prefix(url) + 'pending', { path: '/' });
				redirect(303, '/sign-in?error=source_unavailable');
			}
		}
		await upstream(
			'/v1/auth/handoffs/confirm',
			device(cookies, url),
			{ receiptId: pending.receiptId },
			pending.token
		);
		// Cookie metadata is a presentation hint; the confirmed bearer owns identity.
		const identity = (await upstream(
			'/v1/whoami',
			device(cookies, url),
			undefined,
			pending.token
		)) as { did?: unknown; handle?: unknown };
		if (
			identity?.did !== pending.did ||
			typeof identity.handle !== 'string' ||
			identity.handle.length === 0 ||
			identity.handle.length > 253
		) {
			cookies.delete(prefix(url) + 'pending', { path: '/' });
			redirect(303, '/sign-in?error=identity_unavailable');
		}
		pending.handle = identity.handle;
		store(cookies, url, pending);
	} catch (error) {
		if (isRedirect(error)) throw error;
		logServerFailure(
			error,
			'auth.callback',
			'request',
			error instanceof UpstreamError
				? { status: error.status, code: error.code, requestId: error.requestId }
				: {}
		);
		if (
			error instanceof UpstreamError &&
			[400, 401, 403, 410].includes(error.status)
		)
			cookies.delete(prefix(url) + 'pending', { path: '/' });
		redirect(303, '/sign-in?error=confirmation_pending');
	}
	redirect(303, returnPath(pending.returnTo));
};
