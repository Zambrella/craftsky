import { authenticatedPage } from '#lib/server/auth/page.ts';
import { selected } from '#lib/server/auth/session.ts';
import { publicAPI } from '#lib/server/api.ts';
import type { Post } from '#lib/public.ts';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ params, fetch, cookies, url }) => {
	if (selected(cookies, url))
		return { authenticated: await authenticatedPage({ cookies, url }) };
	return {
		authenticated: null,
		post: await publicAPI<Post>(
			fetch,
			`posts/${encodeURIComponent(params.did)}/${encodeURIComponent(params.rkey)}`
		)
	};
};
