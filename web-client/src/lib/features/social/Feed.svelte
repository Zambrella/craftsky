<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import type {
		CraftskyAPI,
		Profile,
		PostPresentation
	} from '#lib/api/client.ts';
	import MemberPost from './MemberPost.svelte';
	import Filters from './Filters.svelte';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import { label } from '#lib/public.ts';
	import { browseFilters } from './filter-model';
	let { api, did }: { api: CraftskyAPI; did: string } = $props();
	const work = new Work();
	let profile = $state<Profile>();
	let items = $state<
			{ key: string; post: PostPresentation; reason?: string }[]
		>([]),
		cursor = $state<string | null>(null);
	const craft = page.url.searchParams.get('craft') || '';
	const filters = browseFilters(page.url.searchParams);
	async function load(more = false) {
		await work.run(
			async () => {
				if (!profile) profile = await api.profiles.me();
				const options = { limit: 30, cursor: more ? cursor : undefined };
				if (!craft) {
					const result = await api.posts.timeline(options);
					return {
						items: result.items.map((item) => ({
							key: item.itemKey,
							post: item.post,
							reason: item.reason?.by.displayName || item.reason?.by.handle
						})),
						cursor: result.cursor
					};
				}
				const result = await api.search.browseProjects({
					...options,
					craftType: [
						craft.includes('#') ? craft : `social.craftsky.feed.defs#${craft}`
					],
					sort:
						page.url.searchParams.get('sort') === 'popular'
							? 'popular'
							: 'chronological',
					filters
				});
				return {
					items: result.items.map((post) => ({
						key:
							'uri' in post
								? post.uri || crypto.randomUUID()
								: crypto.randomUUID(),
						post
					})),
					cursor: result.cursor
				};
			},
			(result) => {
				items = more
					? [
							...items,
							...result.items.filter(
								(item) => !items.some((old) => old.key === item.key)
							)
						]
					: result.items;
				cursor = result.cursor || null;
			}
		);
	}
	onMount(() => {
		const refresh = () => void load();
		window.addEventListener('craftsky-refresh', refresh);
		void load();
		return () => window.removeEventListener('craftsky-refresh', refresh);
	});
	onDestroy(() => work.dispose());
</script>

<div class="workspace-heading">
	<h1>Home</h1>
	<a class="button" href="/app/compose">Create post</a>
</div>
<nav class="tabs" aria-label="Home feeds">
	<a href="/app/home" aria-current={!craft ? 'page' : undefined}>Following</a
	>{#each profile?.crafts || [] as item}<a
			href={'/app/home?craft=' + encodeURIComponent(item)}
			aria-current={craft === item ? 'page' : undefined}>{label(item)}</a
		>{/each}
</nav>
<div class="toolbar">
	<button disabled={work.busy} onclick={() => load()}>Refresh</button
	>{#if craft}<form method="get" action="/app/home">
			<input type="hidden" name="craft" value={craft} /><label
				>Sort<select
					name="sort"
					value={page.url.searchParams.get('sort') || 'chronological'}
					><option value="chronological">Newest first</option><option
						value="popular">Popular</option
					></select
				></label
			><Filters {craft} /><button>Apply</button><a
				href={'/app/home?craft=' + encodeURIComponent(craft)}>Clear filters</a
			>
		</form>{/if}
</div>
<Status {work} />{#each items as item (item.key)}{#if item.reason}<p
			class="repost-reason"
		>
			Reposted by {item.reason}
		</p>{/if}<MemberPost
		value={item.post}
		{api}
		{did}
		onRemove={() => (items = items.filter((old) => old.key !== item.key))}
	/>{:else}{#if !work.busy && !work.error}<div class="empty">
			<h2>Your next inspiration is out there.</h2>
			<p>
				{craft
					? 'No projects match these filters yet.'
					: 'Follow some makers to see their posts here.'}
			</p>
			<a href="/app/search">Find your people</a>
		</div>{/if}{/each}{#if cursor}<button
		class="button next"
		disabled={work.busy}
		onclick={() => load(true)}>Load more</button
	>{/if}
