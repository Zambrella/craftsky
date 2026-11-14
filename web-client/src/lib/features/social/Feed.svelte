<script lang="ts">
	import { captureScrollAnchor } from '#lib/scroll-anchor.ts';
	let feedElement = $state<HTMLElement>();

	import { appendPage, reloadPages } from '#lib/pagination.ts';
	import { dragScroll } from '#lib/drag-scroll.ts';
	import { getContext, onMount, onDestroy, untrack, tick } from 'svelte';
	import {
		Publications,
		publicationsContext,
		matchesProject
	} from './publications.svelte';
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
	import LoadMore from '#lib/LoadMore.svelte';
	import { craftFeeds, craftKey } from './craft-feeds';
	let { api, did }: { api: CraftskyAPI; did?: string } = $props();
	const guest = $derived(api.audience === 'guest');
	type FeedItem = {
		key: string;
		post: PostPresentation;
		reason?: Author;
		followedBy?: Author;
	};
	type FeedSnapshot = {
		profile?: Profile;
		items: FeedItem[];
		cursor: string | null;
		pages: number;
	};
	const historyView = rememberPage<FeedSnapshot>(
		'feed',
		'/feed' + page.url.search
	);
	const currentProfile = getContext<(() => Profile | undefined) | undefined>(
		'craftsky-profile'
	);
	const snapshot = historyView?.value;
	let loadedPages = snapshot?.pages || 0;
	let disposed = false;
	let loaded = $state(!!snapshot);
	let work = $state(new Work('social.Feed.action'));
	let profile = $state<Profile | undefined>(
		snapshot?.profile ?? untrack(() => currentProfile?.())
	);
	let items = $state<FeedItem[]>(snapshot?.items || []),
		cursor = $state<string | null>(snapshot?.cursor || null);
	function saveView() {
		if (loaded)
			historyView?.save({ profile, items, cursor, pages: loadedPages });
	}
	const craft = page.url.searchParams.get('craft') || '';
	const extendedNetwork = $derived(
		!guest &&
			!craft &&
			page.url.searchParams.get('source') === 'extended-network'
	);
	const filters = browseFilters(page.url.searchParams);
	const publications = getContext<Publications | undefined>(
		publicationsContext
	);
	const displayedItems = $derived.by(() => {
		const merged = publications?.merge(
			items.map((item) => item.post),
			(post) =>
				!extendedNetwork &&
				!post.reply &&
				(!craft || matchesProject(post, craft, filters))
		);
		if (!merged) return items;
		// Several timeline activities can present the same post. Match each
		// occurrence in order so overlays retain its own item key and reposter.
		const byUri = new Map<PostPresentation['uri'], FeedItem[]>();
		for (const item of items) {
			const matches = byUri.get(item.post.uri) || [];
			matches.push(item);
			byUri.set(item.post.uri, matches);
		}
		return merged.map((post) => {
			const item = byUri.get(post.uri)?.shift();
			return { ...item, key: item?.key ?? 'published:' + post.uri, post };
		});
	});
	$effect.pre(() => {
		void displayedItems;
		if (!feedElement) return;
		const restore = untrack(() => captureScrollAnchor(feedElement!));
		void tick().then(restore);
	});
	async function load(more = false) {
		if (disposed || work.busy || (more && !cursor)) return;
		const request = historyView?.beginLoad();
		await work.run(
			async () => {
				if (!guest && !profile) profile = await api.profiles.me();
				const fetchPage = async (nextCursor?: string | null) => {
					const options = {
						limit: guest ? 20 : 30,
						cursor: nextCursor
					};
					if (!craft && !guest) {
						const result = await (extendedNetwork
							? api.posts.extendedNetwork(options)
							: api.posts.timeline(options));
						return {
							items: result.items.map((item) => ({
								key: item.itemKey,
								post: item.post,
								reason: item.reason?.by,
								followedBy: item.followedBy
							})),
							cursor: result.cursor
						};
					}
					const result = await api.search.browseProjects({
						...options,
						craftType: craft
							? [
									craft.includes('#')
										? craft
										: `social.craftsky.feed.defs#${craft}`
								]
							: undefined,
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
				};
				return more
					? { ...(await fetchPage(cursor)), pages: loadedPages + 1 }
					: reloadPages(
							fetchPage,
							Math.max(1, loadedPages),
							(item) => item.key
						);
			},
			(result) => {
				if (request && !request.isCurrent()) return;
				publications?.observe(result.items.map((item) => item.post));
				items = more
					? appendPage(items, result.items, (item) => item.key)
					: result.items;
				cursor = result.cursor || null;
				loadedPages = result.pages;
				loaded = true;
				if (request)
					request.save({ profile, items, cursor, pages: loadedPages });
				else saveView();
			},
			undefined,
			'social.Feed.load'
		);
	}
	onMount(() => {
		const refresh = () => {
			work.dispose();
			work = new Work('social.Feed.action');
			work.hasLoaded = loaded;
			void load();
		};
		const unsubscribe = historyView?.onInvalidate(refresh);
		window.addEventListener('craftsky-refresh', refresh);
		void (async () => {
			if (!snapshot) {
				const pages = historyView?.value?.pages || 1;
				await load();
				while (!disposed && cursor && !work.error && loadedPages < pages)
					await load(true);
			} else work.hasLoaded = true;
			historyView?.ready();
		})();
		return () => {
			unsubscribe?.();
			window.removeEventListener('craftsky-refresh', refresh);
		};
	});
	onDestroy(() => {
		disposed = true;
		saveView();
		work.dispose();
	});
</script>

<div
	bind:this={feedElement}
	class="feed-content"
	hidden={page.url.pathname.endsWith('/filters')}
>
	<div class="workspace-heading">
		<h1>Home</h1>
		{#if craft || guest}<div class="heading-actions">
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
				{#if craft}
					<button
						class="filter-trigger icon-button"
						aria-label="Filter projects"
						onclick={() => goto('/feed/filters' + page.url.search)}
						><Icon name="filter" /><span class="filter-label">Filters</span
						>{#if Object.keys(filters).length}<span
								class="filter-dot"
								aria-label="Filters applied"
							></span>{/if}</button
					>{/if}
			</div>{/if}
	</div>
	<nav class="tabs" use:dragScroll aria-label="Home feeds">
		<a
			href="/feed"
			aria-current={!craft && !extendedNetwork ? 'page' : undefined}
			><Icon name={guest ? 'home' : 'users'} size={18} />{guest
				? 'All crafts'
				: 'Following'}</a
		>
		{#if !guest}<a
				href="/feed?source=extended-network"
				aria-current={extendedNetwork ? 'page' : undefined}
				><Icon name="extendedNetwork" size={18} />Extended network</a
			>{/if}
		{#each craftFeeds(currentProfile?.()?.crafts ?? profile?.crafts) as item}{@const key =
				craftKey(item.id)}<a
				href={'/feed?craft=' + encodeURIComponent(item.id)}
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
		showLoading={!loaded}
		skeleton="posts"
		reload={() => load()}
	/>{#each displayedItems as item (item.key)}<div data-scroll-anchor={item.key}>
			<MemberPost
				value={item.post}
				reason={item.reason}
				followedBy={item.followedBy}
				{api}
				{did}
				onRemove={() => (items = items.filter((old) => old.key !== item.key))}
			/>
		</div>{:else}{#if !work.busy && !work.error}<EmptyState
				title={craft
					? label(craft)
					: extendedNetwork
						? 'Extended network'
						: 'Feed'}
				detail={craft
					? 'No projects match these filters yet.'
					: extendedNetwork
						? 'Posts from people followed by accounts you follow will appear here.'
						: 'Your feed is quiet.'}
				icon="home"
			>
				{#snippet actions()}{#if extendedNetwork}<a href="/feed">Following</a><a
							href="/search">Search CraftSky</a
						>{:else if !craft && !guest}<a href="/settings/instagram"
							>Connect Instagram</a
						><a href="/search">Search CraftSky</a><a
							href={'/feed?craft=' +
								encodeURIComponent(
									profile?.crafts[0] || 'social.craftsky.feed.defs#knitting'
								)}>Explore projects</a
						>{:else}<a href={'/feed?craft=' + encodeURIComponent(craft)}
							>Clear filters</a
						>{/if}{/snippet}
			</EmptyState>{/if}{/each}{#if cursor}<LoadMore
			loading={work.busy}
			loadingLabel="Loading more posts"
			autoLoad={!work.error && !page.url.pathname.endsWith('/filters')}
			pageKey={cursor}
			onclick={() => load(true)}
		/>{/if}
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
			background: var(--primary);
			color: var(--on-primary);
			transition:
				transform 120ms ease,
				box-shadow 120ms ease,
				background 120ms ease;
		}
		:global(.app-main) .filter-trigger.icon-button:hover {
			background: var(--primary-hover);
			color: var(--on-primary);
			transform: translate(-1px, -1px);
			box-shadow: 4px 4px 0 var(--shadow);
		}
		:global(.app-main) .filter-trigger.icon-button:active {
			background: var(--primary-pressed);
			transform: translate(3px, 3px);
			box-shadow: none;
		}
		.filter-dot {
			background: var(--on-primary);
		}
	}
	@media (max-width: 900px) and (prefers-reduced-motion: reduce) {
		:global(.app-main) .filter-trigger.icon-button:hover,
		:global(.app-main) .filter-trigger.icon-button:active {
			transform: none;
		}
	}
</style>
