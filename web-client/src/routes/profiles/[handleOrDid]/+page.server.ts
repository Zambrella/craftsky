import { publicProfileForDisplay } from '#lib/api/public-client.ts';
import { profileTabs } from '#lib/features/social/presentation.ts';
import { authenticatedPage } from '#lib/server/auth/page.ts';
import { error } from '@sveltejs/kit';
import { selected } from '#lib/server/auth/session.ts';
import { publicAPI } from '#lib/server/api.ts';
import type { Profile, PostPage } from '#lib/public.ts';
import type { BusinessEvent, Page } from '#lib/api/types.ts';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ params, url, fetch, cookies }) => {
	if (selected(cookies, url))
		return { authenticated: await authenticatedPage({ cookies, url }) };
	const requestedTab = url.searchParams.get('tab');
	const tab = requestedTab || 'projects';
	if (
		![
			'posts',
			'projects',
			'comments',
			'reposts',
			'about',
			'products',
			'events'
		].includes(tab)
	)
		error(400, 'Invalid profile tab');
	const cursor = url.searchParams.get('cursor') || '';
	if (cursor.length > 2048) error(400, 'Invalid cursor');
	const id = encodeURIComponent(params.handleOrDid.replace(/^@/, ''));
	const profile = await publicAPI<Profile>(fetch, `profiles/${id}`);
	const selectedTab =
		requestedTab || profileTabs(publicProfileForDisplay(profile), false)[0];
	if (
		['about', 'products', 'events'].includes(selectedTab) &&
		!profile.business
	)
		error(400, 'Invalid profile tab');
	const query = new URLSearchParams({ cursor });
	const posts = ['posts', 'projects', 'comments', 'reposts'].includes(
		selectedTab
	)
		? await publicAPI<PostPage>(fetch, `profiles/${id}/${selectedTab}?${query}`)
		: { items: [] };
	const events: Page<BusinessEvent> =
		selectedTab === 'events'
			? await publicAPI(fetch, `profiles/${id}/events?${query}`)
			: { items: [] };
	return {
		authenticated: null,
		profile,
		posts,
		events,
		tab: selectedTab,
		cursor
	};
};
