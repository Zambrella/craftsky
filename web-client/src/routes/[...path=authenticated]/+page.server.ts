import { guestRoute } from '#lib/routes.ts';
import { selected } from '#lib/server/auth/session.ts';
import { redirect } from '@sveltejs/kit';
import { authenticatedPage } from '#lib/server/auth/page.ts';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async (event) => {
	if (event.url.pathname === '/projects')
		redirect(303, '/feed' + event.url.search);
	if (!selected(event.cookies, event.url) && guestRoute(event.url.pathname))
		return {
			guest: {
				path: event.url.pathname.slice(1).split('/').map(decodeURIComponent)
			}
		};
	return authenticatedPage(event);
};
