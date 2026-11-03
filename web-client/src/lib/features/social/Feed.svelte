<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import type {
		CraftskyAPI,
		Author,
		Profile,
		PostPresentation
	} from '#lib/api/client.ts';
	import MemberPost from './MemberPost.svelte';
	import Filters from './Filters.svelte';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import { label } from '#lib/public.ts';
	import { browseFilters } from './filter-model';
	import Icon from '../shell/Icon.svelte';
	import ChoiceMenu from './ChoiceMenu.svelte';
	import EmptyState from './EmptyState.svelte';
	import { craftFeeds, craftKey } from './craft-feeds';
	let { api, did }: { api: CraftskyAPI; did: string } = $props();
	const work = new Work();
	let profile = $state<Profile>();
	let items = $state<
			{ key: string; post: PostPresentation; reason?: Author }[]
		>([]),
		cursor = $state<string | null>(null);
	const craft = page.url.searchParams.get('craft') || '';
	const filters = browseFilters(page.url.searchParams);
	let filterDialog = $state<HTMLDialogElement>(null!);
	let filterFields = $state<{ clearAll: () => void }>(null!);
	let filterVersion = $state(0);
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
							reason: item.reason?.by
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
	<div class="heading-actions">
		<button
			class="refresh icon-button"
			aria-label="Refresh"
			disabled={work.busy}
			onclick={() => load()}><Icon name="refresh" /></button
		>
	</div>
</div>
<nav class="tabs" aria-label="Home feeds">
	<a href="/app/home" aria-current={!craft ? 'page' : undefined}
		><Icon name="users" size={18} />Following</a
	>{#each craftFeeds(profile?.crafts) as item}{@const key = craftKey(item.id)}<a
			href={'/app/home?craft=' + encodeURIComponent(item.id)}
			aria-current={craftKey(craft) === key ? 'page' : undefined}
			>{#if ['knitting', 'crochet', 'sewing', 'embroidery', 'quilting'].includes(key)}<span
					class="feed-craft-icon"
					style:--craft-icon={`url('/craft_icons/${key}.svg')`}
					aria-hidden="true"
				></span>{/if}{label(key)}</a
		>{/each}
</nav>
{#if craft}<div class="feed-tools">
		<ChoiceMenu
			label="Sort projects"
			value={page.url.searchParams.get('sort') || 'chronological'}
			url={page.url}
			options={[
				{ value: 'chronological', label: 'Newest', icon: 'sort' },
				{ value: 'popular', label: 'Popular', icon: 'growth' }
			]}
		/>
		<button
			class="filter-trigger"
			onclick={() => {
				filterVersion++;
				filterDialog.showModal();
			}}
			><Icon name="filter" />Filter projects{#if Object.keys(filters).length}<span
					class="filter-dot"
					aria-label="Filters applied"
				></span>{/if}</button
		>
	</div>{/if}
<Status
	{work}
	skeleton="posts"
	reload={() => load()}
/>{#each items as item (item.key)}<MemberPost
		value={item.post}
		reason={item.reason}
		{api}
		{did}
		onRemove={() => (items = items.filter((old) => old.key !== item.key))}
	/>{:else}{#if !work.busy && !work.error}<EmptyState
			title={craft ? label(craft) : 'Feed'}
			detail={craft
				? 'No projects match these filters yet.'
				: 'Your feed is quiet.'}
			icon="home"
		>
			{#snippet actions()}{#if !craft}<a href="/app/settings/instagram"
						>Connect Instagram</a
					><a href="/app/search">Search CraftSky</a><a
						href={'/app/home?craft=' +
							encodeURIComponent(
								profile?.crafts[0] || 'social.craftsky.feed.defs#knitting'
							)}>Explore projects</a
					>{:else}<a href={'/app/home?craft=' + encodeURIComponent(craft)}
						>Clear filters</a
					>{/if}{/snippet}
		</EmptyState>{/if}{/each}{#if cursor}<button
		class="button next"
		disabled={work.busy}
		onclick={() => load(true)}>Load more</button
	>{/if}

<dialog
	bind:this={filterDialog}
	class="project-filter-dialog"
	aria-labelledby="filter-title"
>
	<form method="get" action="/app/home">
		<header>
			<button
				type="button"
				class="icon-button"
				aria-label="Close filters"
				onclick={() => filterDialog.close()}><Icon name="close" /></button
			>
			<h2 id="filter-title">
				{#if ['knitting', 'crochet', 'sewing', 'embroidery', 'quilting'].includes(craft
						.split('#')
						.at(-1)
						?.toLowerCase() || '')}<span
						class="feed-craft-icon"
						style:--craft-icon={`url('/craft_icons/${craft.split('#').at(-1)?.toLowerCase()}.svg')`}
						aria-hidden="true"
					></span>{/if}
				Filter {label(craft)} projects
			</h2>
		</header>
		<input type="hidden" name="craft" value={craft} /><input
			type="hidden"
			name="sort"
			value={page.url.searchParams.get('sort') || 'chronological'}
		/>
		<div class="filter-scroll">
			{#key filterVersion}<Filters
					bind:this={filterFields}
					{craft}
					expanded
				/>{/key}
		</div>
		<footer>
			<button
				type="button"
				class="clear-all-filters"
				onclick={() => filterFields.clearAll()}>Clear all</button
			><button class="button">Apply filters</button>
		</footer>
	</form>
</dialog>

<style>
	.tabs a {
		gap: 6px;
	}
	.feed-craft-icon {
		width: 18px;
		height: 18px;
		flex-shrink: 0;
		background: currentColor;
		mask: var(--craft-icon) center / contain no-repeat;
	}
	.heading-actions {
		display: flex;
		gap: 8px;
		align-items: center;
	}
	.refresh {
		width: 44px;
		height: 44px;
	}
	.feed-tools {
		display: flex;
		justify-content: flex-end;
		align-items: center;
		gap: 8px;
		padding: 12px 16px;
	}
	.filter-trigger {
		display: flex;
		align-items: center;
		gap: 6px;
		min-height: 42px;
		font-size: 14px;
	}
	.filter-dot {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		background: var(--primary);
	}
	.project-filter-dialog {
		width: min(620px, calc(100vw - 32px));
		padding: 0;
		height: min(900px, calc(100dvh - 48px));
		max-height: none;
		overflow: hidden;
		background: var(--paper);
	}
	.project-filter-dialog form {
		display: flex;
		flex-direction: column;
		height: 100%;
	}
	.filter-scroll {
		flex: 1;
		min-height: 0;
		overflow: auto;
	}
	.project-filter-dialog header,
	.project-filter-dialog footer {
		padding: 16px;
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 16px;
	}
	.project-filter-dialog header {
		border-bottom: 1.5px solid var(--ink);
		flex: none;
		min-height: 56px;
		padding: 4px 8px;
		justify-content: flex-start;
		gap: 8px;
	}
	.project-filter-dialog footer {
		position: sticky;
		bottom: 0;
		background: var(--paper);
		border: 0;
		margin: 0;
		width: 100%;
		flex: none;
	}
	.project-filter-dialog footer > .button {
		box-shadow: none;
		border: 0;
		font-size: 16px;
	}
	.clear-all-filters {
		font-size: 16px;
		border: 0;
		background: transparent;
		box-shadow: none;
		color: var(--primary);
	}
	.project-filter-dialog h2 {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 20px;
		margin: 0;
	}
	@media (max-width: 900px) {
		.filter-trigger {
			position: fixed;
			right: 16px;
			bottom: calc(80px + env(safe-area-inset-bottom));
			z-index: 7;
			border-radius: 28px;
			min-height: 48px;
			padding: 10px 18px;
			box-shadow: 3px 3px var(--shadow);
			background: var(--surface);
		}
	}
	@media (max-width: 900px) {
		.project-filter-dialog {
			margin: 0;
			inset: 0;
			width: 100vw;
			max-width: 100vw;
			height: 100dvh;
			max-height: none;
			border: 0;
			border-radius: 0;
		}
	}
</style>
