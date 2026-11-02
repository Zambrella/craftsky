import { publicAPI } from '#lib/server/api.ts';
import type { Post } from '#lib/public.ts';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ params, fetch }) => ({
	post: await publicAPI<Post>(
		fetch,
		`posts/${encodeURIComponent(params.did)}/${encodeURIComponent(params.rkey)}`
	)
});
