import { authenticatedPage } from '#lib/server/auth/page.ts';
import { error } from '@sveltejs/kit';
import { selected } from '#lib/server/auth/session.ts';
import { publicAPI } from '#lib/server/api.ts';
import type { Profile, PostPage } from '#lib/public.ts';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ params, url, fetch, cookies }) => {
	if (selected(cookies, url))
		return { authenticated: await authenticatedPage({ cookies, url }) };
	const tab = url.searchParams.get('tab') || 'posts';
	if (!['posts', 'projects'].includes(tab)) error(400, 'Invalid profile tab');
	const cursor = url.searchParams.get('cursor') || '';
	if (cursor.length > 2048) error(400, 'Invalid cursor');
	const id = encodeURIComponent(params.handleOrDid.replace(/^@/, ''));
	const [profile, posts] = await Promise.all([
		publicAPI<Profile>(fetch, `profiles/${id}`),
		publicAPI<PostPage>(
			fetch,
			`profiles/${id}/${tab}?${new URLSearchParams({ cursor })}`
		)
	]);
	return { authenticated: null, profile, posts, tab, cursor };
};
