<script lang="ts">
	import { getContext } from 'svelte';
	import type { CraftskyAPI } from '#lib/api/client.ts';
	import Feed from '../social/Feed.svelte';
	import Thread from '../social/Thread.svelte';
	import Event from '../social/Event.svelte';
	import ProjectFiltersPage from '../social/ProjectFiltersPage.svelte';
	import Search from '../social/Search.svelte';
	import Tag from '../social/Tag.svelte';
	import { page } from '$app/state';
	let { path }: { path: string[] } = $props();
	const api = getContext<CraftskyAPI>('craftsky-api');
</script>

{#key path[0] === 'search' && path[1] !== 'tags' ? path[0] : page.url.pathname + page.url.search}
	{#if path[0] === 'feed' && path[1] === 'filters'}<ProjectFiltersPage />
	{:else if path[0] === 'feed'}<Feed {api} />
	{:else if path[0] === 'search' && path[1] === 'tags'}<Tag
			{api}
			tag={page.url.searchParams.get('tag') || ''}
		/>
	{:else if path[0] === 'search'}<Search {api} />
	{:else if path[0] === 'posts' && path.length >= 3}<Thread
			{api}
			did={path[1]}
			rkey={path[2]}
			section={path[3]}
		/>
	{:else if path[0] === 'events' && path.length >= 3}<Event
			{api}
			did={path[1]}
			rkey={path[2]}
		/>
	{/if}
{/key}
