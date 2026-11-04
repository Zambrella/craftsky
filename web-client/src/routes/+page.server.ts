import { redirect } from '@sveltejs/kit';
import { selected } from '#lib/server/auth/session.ts';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = ({ cookies, url }) => {
	if (selected(cookies, url)) redirect(303, '/app/home');
	return {};
};
