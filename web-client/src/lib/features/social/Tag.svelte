<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { CraftskyAPI, PostPresentation } from '#lib/api/client.ts';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import MemberPost from './MemberPost.svelte';
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
	<h1>#{tag}</h1>
	<button onclick={() => load()}>Refresh</button>
</div>
<Status {work} />{#each posts as post}<MemberPost
		value={post}
		{api}
		{did}
	/>{:else}{#if !work.busy}<p class="empty">
			No posts found for this hashtag.
		</p>{/if}{/each}{#if cursor}<button
		class="button next"
		onclick={() => load(true)}>Load more</button
	>{/if}
