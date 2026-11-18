<script lang="ts">
	import { getContext, onMount, onDestroy } from 'svelte';
	import { appendPage } from '#lib/pagination.ts';
	import type { Author, CraftskyAPI } from '#lib/api/client.ts';
	import Avatar from '#lib/Avatar.svelte';
	import CompactIdentity from './CompactIdentity.svelte';
	import Icon from '../shell/Icon.svelte';
	import Status from './Status.svelte';
	import EmptyState from './EmptyState.svelte';
	import { Work } from './work.svelte';
	import LoadMore from '#lib/LoadMore.svelte';
	let { api, id }: { api: CraftskyAPI; id: string } = $props();
	const back = getContext<((event: MouseEvent) => void) | undefined>(
		'craftsky-back'
	);
	const work = new Work('social.MutualFollowers.action');
	let items = $state<Author[]>([]);
	let cursor = $state<string | null | undefined>();
	const profileURL = $derived('/profiles/' + encodeURIComponent(id));
	function load(more = false) {
		return work.run(
			() =>
				api.profiles.mutualFollowers(id, { cursor: more ? cursor : undefined }),
			(value) => {
				items = more
					? appendPage(items, value.items, (item) => item.did)
					: value.items;
				cursor = value.cursor;
			},
			undefined,
			'social.MutualFollowers.load'
		);
	}
	onMount(() => void load());
	onDestroy(() => work.dispose());
</script>

<header class="workspace-heading mutual-heading">
	<a
		class="icon-button"
		href={profileURL}
		aria-label="Back to profile"
		onclick={back}><Icon name="back" /></a
	>
	<h1>Mutual followers</h1>
</header>
<Status {work} skeleton="people" reload={() => load()} />
<section aria-label="Mutual followers">
	{#each items as author (author.did)}
		<a class="people-row" href={'/profiles/' + encodeURIComponent(author.did)}>
			<Avatar {author} /><CompactIdentity {author} />
		</a>
	{:else}{#if !work.busy && !work.error}
			<EmptyState
				title="Mutual followers"
				detail="No mutual followers to show."
				icon="profile"
			/>{/if}
	{/each}
	{#if cursor}<LoadMore
			loading={work.busy}
			disabled={!!work.retry}
			label="Load more mutual followers"
			loadingLabel="Loading more mutual followers"
			onclick={() => load(true)}
		/>{/if}
</section>

<style>
	.mutual-heading {
		display: flex;
		align-items: center;
		justify-content: flex-start;
		gap: 12px;
	}
</style>
