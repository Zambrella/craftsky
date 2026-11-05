import { redirect } from '@sveltejs/kit';
import { authenticatedPage } from '#lib/server/auth/page.ts';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async (event) => {
	if (event.url.pathname === '/projects')
		redirect(303, '/feed' + event.url.search);
	return authenticatedPage(event);
};
