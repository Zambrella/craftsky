<script lang="ts">
	import { appendPage } from '#lib/pagination.ts';
	import { dragScroll } from '#lib/drag-scroll.ts';
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import type {
		CraftskyAPI,
		Author,
		Profile,
		PostPresentation
	} from '#lib/api/client.ts';
	import MemberPost from './MemberPost.svelte';
	import { rememberPage } from '../shell/page-history';
	import ProjectFiltersPage from './ProjectFiltersPage.svelte';
	import { goto } from '$app/navigation';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import { label } from '#lib/public.ts';
	import { browseFilters } from './filter-model';
	import Icon from '../shell/Icon.svelte';
	import ChoiceMenu from './ChoiceMenu.svelte';
	import EmptyState from './EmptyState.svelte';
	import { craftFeeds, craftKey } from './craft-feeds';
	let { api, did }: { api: CraftskyAPI; did: string } = $props();
	const historyView = rememberPage<number>('feed-pages');
	let loadedPages = 0;
	let disposed = false;
	const work = new Work();
	let profile = $state<Profile>();
	let items = $state<
			{ key: string; post: PostPresentation; reason?: Author }[]
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
					? appendPage(items, result.items, (item) => item.key)
					: result.items;
				cursor = result.cursor || null;
				loadedPages = more ? loadedPages + 1 : 1;
				historyView?.save(loadedPages);
			}
		);
	}
	onMount(() => {
		const refresh = () => void load();
		window.addEventListener('craftsky-refresh', refresh);
		void (async () => {
			const pages = historyView?.value || 1;
			await load();
			while (!disposed && cursor && !work.error && loadedPages < pages)
				await load(true);
			historyView?.ready();
		})();
		return () => window.removeEventListener('craftsky-refresh', refresh);
	});
	onDestroy(() => {
		disposed = true;
		work.dispose();
	});
</script>

<div hidden={page.url.pathname.endsWith('/filters')}>
	<div class="workspace-heading">
		<h1>Home</h1>
		{#if craft}<div class="heading-actions">
				<ChoiceMenu
					label="Sort projects"
					appBar
					value={page.url.searchParams.get('sort') || 'chronological'}
					url={page.url}
					options={[
						{ value: 'chronological', label: 'Newest', icon: 'sort' },
						{ value: 'popular', label: 'Popular', icon: 'growth' }
					]}
				/>
				<button
					class="filter-trigger icon-button"
					aria-label="Filter projects"
					onclick={() => goto('/app/home/filters' + page.url.search)}
					><Icon name="filter" /><span class="filter-label">Filters</span
					>{#if Object.keys(filters).length}<span
							class="filter-dot"
							aria-label="Filters applied"
						></span>{/if}</button
				>
			</div>{/if}
	</div>
	<nav class="tabs" use:dragScroll aria-label="Home feeds">
		<a href="/app/home" aria-current={!craft ? 'page' : undefined}
			><Icon name="users" size={18} />Following</a
		>{#each craftFeeds(profile?.crafts) as item}{@const key = craftKey(
				item.id
			)}<a
				href={'/app/home?craft=' + encodeURIComponent(item.id)}
				aria-current={craftKey(craft) === key ? 'page' : undefined}
				>{#if ['knitting', 'crochet', 'sewing', 'embroidery', 'quilting'].includes(key)}<span
						class="feed-craft-icon"
						style:--craft-icon={`url('/craft_icons/${key}.svg')`}
						aria-hidden="true"
					></span>{/if}{label(key)}</a
			>{/each}
	</nav>

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
</div>
{#if page.url.pathname.endsWith('/filters')}<ProjectFiltersPage />{/if}

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
	.filter-trigger {
		display: flex;
		align-items: center;
		gap: 6px;
		min-height: 44px;
		border: 0;
		box-shadow: none;
		background: transparent;
		font-size: 14px;
	}
	.filter-label {
		display: none;
	}
	.filter-dot {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		background: var(--primary);
	}
	@media (max-width: 900px) {
		.filter-label {
			display: inline;
		}
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
</style>
