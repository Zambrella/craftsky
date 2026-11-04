import { error, redirect } from '@sveltejs/kit';
import { selected } from '#lib/server/auth/session.ts';
import { publicAPI } from '#lib/server/api.ts';
import type { Profile, PostPage } from '#lib/public.ts';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ params, url, fetch, cookies }) => {
	if (selected(cookies, url))
		redirect(
			303,
			'/app/profile/' +
				encodeURIComponent(params.handleOrDid) +
				(url.searchParams.get('tab')
					? '?tab=' + encodeURIComponent(url.searchParams.get('tab')!)
					: '')
		);
	const tab = url.searchParams.get('tab') || 'posts';
	if (!['posts', 'projects'].includes(tab)) error(400, 'Invalid profile tab');
	const cursor = url.searchParams.get('cursor') || '';
	if (cursor.length > 2048) error(400, 'Invalid cursor');
	const id = encodeURIComponent(params.handleOrDid);
	const [profile, posts] = await Promise.all([
		publicAPI<Profile>(fetch, `profiles/${id}`),
		publicAPI<PostPage>(
			fetch,
			`profiles/${id}/${tab}?${new URLSearchParams({ cursor })}`
		)
	]);
	return { profile, posts, tab, cursor };
};
