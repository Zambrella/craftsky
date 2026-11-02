import { redirect } from '@sveltejs/kit';
import { selected } from '#lib/server/auth/session.ts';
import { publicAPI } from '#lib/server/api.ts';
import type { Post } from '#lib/public.ts';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ params, fetch, cookies, url }) => {
	if (selected(cookies, url))
		redirect(
			303,
			'/app/posts/' +
				encodeURIComponent(params.did) +
				'/' +
				encodeURIComponent(params.rkey) +
				(url.searchParams.get('focus')
					? '?focus=' + encodeURIComponent(url.searchParams.get('focus')!)
					: '')
		);
	return {
		post: await publicAPI<Post>(
			fetch,
			`posts/${encodeURIComponent(params.did)}/${encodeURIComponent(params.rkey)}`
		)
	};
};
