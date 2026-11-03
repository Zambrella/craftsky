<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { CraftskyAPI, PostPresentation } from '#lib/api/client.ts';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import MemberPost from './MemberPost.svelte';
	import Icon from '../shell/Icon.svelte';
	import EmptyState from './EmptyState.svelte';
	let { api, did, tag }: { api: CraftskyAPI; did: string; tag: string } =
		$props();
	const work = new Work();
	let posts = $state<PostPresentation[]>([]),
		cursor = $state<string | null>(null);
	async function load(more = false) {
		await work.run(
			() => api.search.hashtagPosts(tag, { cursor: more ? cursor : undefined }),
			(value) => {
				posts = more ? [...posts, ...value.items] : value.items;
				cursor = value.cursor || null;
			}
		);
	}
	onMount(() => {
		void load();
		void api.search
			.saveRecent({
				type: 'hashtag',
				displayLabel: '#' + tag,
				payload: { tag }
			})
			.catch(() => {});
	});
	onDestroy(() => work.dispose());
</script>

<div class="workspace-heading">
	<a href="/app/search" class="icon-button" aria-label="Back to search"
		><Icon name="back" /></a
	>
	<h1>#{tag}</h1>
</div>
<Status
	{work}
	skeleton="posts"
	reload={() => load()}
/>{#each posts as post}<MemberPost
		value={post}
		{api}
		{did}
	/>{:else}{#if !work.busy && !work.error}<EmptyState
			title={'#' + tag}
			detail="No posts found for this hashtag."
			icon="search"
		/>{/if}{/each}{#if cursor}<button
		class="button next"
		onclick={() => load(true)}>Load more</button
	>{/if}

<style>
	.workspace-heading {
		justify-content: flex-start;
		gap: 8px;
	}
	.workspace-heading h1 {
		flex: 1;
		overflow-wrap: anywhere;
	}
	.icon-button {
		display: grid;
		place-items: center;
		width: 40px;
		min-height: 44px;
	}
</style>
