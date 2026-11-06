<script lang="ts">
	import { reportFailure } from '../diagnostics/client';
	import { page } from '$app/state';
	import { getContext } from 'svelte';
	import type { CraftskyAPI } from '#lib/api/client.ts';
	const api = getContext<CraftskyAPI>('craftsky-api');
	import Feed from '#lib/features/social/Feed.svelte';
	import Tag from '#lib/features/social/Tag.svelte';
	import Saved from '#lib/features/social/Saved.svelte';
	import Activity from '#lib/features/social/Activity.svelte';
	let {
		data
	}: {
		data: Awaited<
			ReturnType<typeof import('#lib/server/auth/page.ts').authenticatedPage>
		>;
	} = $props();
	let path = $derived(data.path);
	let section = $derived(path[0] || 'feed');
	async function loadPage<T>(load: () => Promise<T>): Promise<T> {
		try {
			return await load();
		} catch (error) {
			void reportFailure(error, 'page.import');
			throw error;
		}
	}
</script>

{#snippet loading()}<p class="empty" role="status">Loading…</p>{/snippet}
{#snippet loadError()}<section class="notice" role="alert">
		This page could not load. <button onclick={() => location.reload()}
			>Reload</button
		>
	</section>{/snippet}

{#key ['profile', 'profiles'].includes(section) || (section === 'search' && path[1] !== 'tags') ? page.url.pathname : (section === 'feed' ? '/feed' : page.url.pathname) + page.url.search}
	{#if section === 'feed'}<Feed
			{api}
			did={data.session.did}
		/>{:else if section === 'search' && path[1] === 'tags'}<Tag
			{api}
			did={data.session.did}
			tag={page.url.searchParams.get('tag') || ''}
		/>{:else if section === 'search'}{#await loadPage(() => import('#lib/features/social/Search.svelte'))}{@render loading()}{:then Module}<Module.default
				{api}
				did={data.session.did}
			/>{:catch}{@render loadError()}{/await}{:else if section === 'profile' && path[1] === 'edit'}{#await loadPage(() => import('#lib/features/social/ProfileEditor.svelte'))}{@render loading()}{:then Module}<Module.default
				{api}
				session={data.session}
			/>{:catch}{@render loadError()}{/await}{:else if section === 'profiles' && path[2] === 'mutual-followers'}{#await loadPage(() => import('#lib/features/social/MutualFollowers.svelte'))}{@render loading()}{:then Module}<Module.default
				{api}
				id={path[1]?.replace(/^@/, '')}
			/>{:catch}{@render loadError()}{/await}{:else if section === 'profile' || section === 'profiles'}{#await loadPage(() => import('#lib/features/social/Profile.svelte'))}{@render loading()}{:then Module}<Module.default
				{api}
				session={data.session}
				id={section === 'profile'
					? data.session.did
					: path[1]?.replace(/^@/, '')}
			/>{:catch}{@render loadError()}{/await}{:else if section === 'posts' && path.length >= 3}{#await loadPage(() => import('#lib/features/social/Thread.svelte'))}{@render loading()}{:then Module}<Module.default
				{api}
				session={data.session}
				did={path[1]}
				rkey={path[2]}
				section={path[3]}
			/>{:catch}{@render loadError()}{/await}{:else if section === 'saved'}<Saved
			{api}
			did={data.session.did}
		/>{:else if section === 'notifications' && path[1] !== 'settings'}<Activity
			{api}
			did={data.session.did}
		/>{:else if section === 'messages'}{#await loadPage(() => import('#lib/features/messages/MessagesWorkspace.svelte'))}{@render loading()}{:then Module}<Module.default
				{api}
				{path}
				session={data.session}
			/>{:catch}{@render loadError()}{/await}{:else if section === 'settings' || (section === 'notifications' && path[1] === 'settings')}{#await loadPage(() => import('#lib/features/settings/SettingsWorkspace.svelte'))}{@render loading()}{:then Module}<Module.default
				{api}
				path={section === 'notifications'
					? ['settings', 'notifications']
					: path}
				session={data.session}
			/>{:catch}{@render loadError()}{/await}{:else if section === 'compose'}{#await loadPage(() => import('#lib/features/composer/ComposerWorkspace.svelte'))}{@render loading()}{:then Module}<Module.default
				{api}
				{path}
				session={data.session}
			/>{:catch}{@render loadError()}{/await}{:else if section === 'drafts' || section === 'scheduled'}{#await loadPage(() => import('#lib/features/drafts/DraftsWorkspace.svelte'))}{@render loading()}{:then Module}<Module.default
				{api}
				{path}
				session={data.session}
			/>{:catch}{@render loadError()}{/await}{:else if section === 'events' && path.length >= 3}{#await loadPage(() => import('#lib/features/social/Event.svelte'))}{@render loading()}{:then Module}<Module.default
				{api}
				did={path[1]}
				rkey={path[2]}
				viewerDid={data.session.did}
			/>{:catch}{@render loadError()}{/await}{:else}<section
			class="panel stack"
		>
			<h1>Page not found</h1>
			<a href="/feed">Back to your feed</a>
		</section>{/if}
{/key}
