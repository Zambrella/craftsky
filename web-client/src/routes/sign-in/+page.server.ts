import { redirect } from '@sveltejs/kit';
import { selected, returnPath } from '#lib/server/auth/session.ts';
import { guestRoute } from '#lib/routes.ts';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = ({ url, cookies }) => {
	// Adding another account is an explicit authenticated account workflow.
	if (url.searchParams.get('add') === '1' && selected(cookies, url)) return {};
	const candidate = new URL(
		returnPath(url.searchParams.get('returnTo')),
		url.origin
	);
	const destination = new URL('/feed', url.origin);
	destination.searchParams.set('auth', '1');
	if (
		guestRoute(candidate.pathname) &&
		(candidate.pathname !== '/feed' || candidate.search !== '')
	)
		destination.searchParams.set(
			'returnTo',
			candidate.pathname + candidate.search
		);
	const error = url.searchParams.get('error');
	if (error)
		destination.searchParams.set(
			'error',
			error === 'confirmation_pending'
				? 'confirmation_pending'
				: 'session_expired'
		);
	redirect(303, destination.pathname + destination.search);
};
