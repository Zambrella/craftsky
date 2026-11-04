<script lang="ts">
	import { dragScroll } from '#lib/drag-scroll.ts';
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import type {
		CraftskyAPI,
		Author,
		PostPresentation,
		Hashtag,
		RecentSearch,
		SearchSuggestions,
		TopHashtags
	} from '#lib/api/client.ts';
	import MemberPost from './MemberPost.svelte';
	import Avatar from '#lib/Avatar.svelte';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import { profileStyle } from './model';
	import { recentDestination, searchDestination } from './search-model';
	import { goto } from '$app/navigation';
	import Icon from '../shell/Icon.svelte';
	import EmptyState from './EmptyState.svelte';
	import CompactIdentity from './CompactIdentity.svelte';
	import ChoiceMenu from './ChoiceMenu.svelte';
	import { label } from '#lib/public.ts';
	let { api, did }: { api: CraftskyAPI; did: string } = $props();
	const work = new Work();
	let query = $state(page.url.searchParams.get('q') || ''),
		suggestions = $state<SearchSuggestions>();
	let top = $state<TopHashtags>();
	let authors = $state<Author[]>([]),
		posts = $state<PostPresentation[]>([]),
		tags = $state<Hashtag[]>([]),
		recent = $state<RecentSearch[]>([]),
		cursor = $state<string | null>(null);
	const tab = page.url.searchParams.get('tab') || 'posts';
	const q = page.url.searchParams.get('q') || '';
	let timer: ReturnType<typeof setTimeout>;
	let suggestionGeneration = 0;
	function suggest() {
		clearTimeout(timer);
		const generation = ++suggestionGeneration;
		if (query.trim().length < 2) {
			suggestions = undefined;
			return;
		}
		timer = setTimeout(async () => {
			try {
				const value = await api.search.suggestions(query, {
					profileLimit: 4,
					hashtagLimit: 4
				});
				if (generation === suggestionGeneration) suggestions = value;
			} catch {}
		}, 250);
	}
	async function load(more = false) {
		await work.run(
			async () => {
				if (!q) {
					const [members, history, hashtags] = await Promise.all([
						api.profiles.recent(),
						api.search.recent(),
						api.search.topHashtags()
					]);
					authors = members.items;
					recent = history.items;
					top = hashtags;
					return null;
				}
				const options = {
					q,
					cursor: more ? cursor : undefined,
					limit: 30,
					sort:
						page.url.searchParams.get('sort') === 'popular'
							? ('popular' as const)
							: ('chronological' as const)
				};
				if (tab === 'profiles') {
					const result = await api.search.profiles(q, options);
					authors = more ? [...authors, ...result.items] : result.items;
					return result.cursor;
				}
				if (tab === 'hashtags') {
					const result = await api.search.hashtags(q, options);
					tags = more ? [...tags, ...result.items] : result.items;
					return result.cursor;
				}
				const result =
					tab === 'posts'
						? await api.search.posts(options)
						: await api.search.projects(options);
				posts = more ? [...posts, ...result.items] : result.items;
				return result.cursor;
			},
			(value) => (cursor = value || null)
		);
	}
	onMount(() => {
		void load();
		if (q)
			void api.search
				.saveRecent({ type: 'query', displayLabel: q, payload: { q } })
				.catch(() => {});
	});
	onDestroy(() => {
		work.dispose();
		clearTimeout(timer);
		suggestionGeneration++;
	});
	function authorValue(author: Author) {
		return {
			did: author.did,
			handle: author.handle,
			displayName: author.displayName || undefined,
			avatar: author.avatar || undefined
		};
	}
	function remember(author: Author) {
		void api.search
			.saveRecent({
				type: 'profile',
				displayLabel: '@' + author.handle,
				payload: {
					did: author.did,
					handle: author.handle,
					displayName: author.displayName,
					avatar: author.avatar,
					crafts: author.crafts
				}
			})
			.catch(() => {});
	}
</script>

<div class="workspace-heading"><h1>Search</h1></div>
<form
	class="search-form"
	method="get"
	action="/app/search"
	onsubmit={(event) => {
		event.preventDefault();
		const target = searchDestination(
			query,
			String(new FormData(event.currentTarget).get('sort') || '')
		);
		if (target) void goto(target);
	}}
>
	<label class="visually-hidden" for="search">Search CraftSky</label>
	<div class="search-input-row">
		<div class="search-field">
			<Icon name="search" /><input
				id="search"
				name="q"
				type="search"
				bind:value={query}
				oninput={suggest}
				placeholder="Search hashtags, people or projects..."
			/>
			{#if query}<button
					type="button"
					class="clear-search icon-button"
					aria-label="Clear search"
					onclick={() => {
						query = '';
						suggestions = undefined;
						suggestionGeneration++;
						document.getElementById('search')?.focus();
					}}><Icon name="close" size={18} /></button
				>{/if}
		</div>
		<button class="search-submit">Search</button>{#if query || q}<a
				class="cancel-search"
				href="/app/search">Cancel</a
			>{/if}
	</div>
	<input type="hidden" name="tab" value={tab} />
	{#if suggestions}<div class="suggestions">
			{#each suggestions.profiles.items as author}<a
					class="suggestion-profile"
					href={'/app/profile/' + encodeURIComponent(author.did)}
					onclick={() => remember(author)}
					><Avatar author={authorValue(author)} /><CompactIdentity
						{author}
					/></a
				>{/each}{#each suggestions.hashtags.items as tag}<a
					href={'/app/tags/' + encodeURIComponent(tag.tag)}>#{tag.tag}</a
				>{/each}
		</div>{/if}
	<input
		type="hidden"
		name="sort"
		value={page.url.searchParams.get('sort') || 'chronological'}
	/>
	{#if q && tab === 'posts'}<div class="search-sort">
			<ChoiceMenu
				label="Sort results"
				value={page.url.searchParams.get('sort') || 'chronological'}
				url={page.url}
				options={[
					{ value: 'chronological', label: 'Newest', icon: 'sort' },
					{ value: 'popular', label: 'Popular', icon: 'growth' }
				]}
			/>
		</div>{/if}
</form>
{#if q}<nav class="tabs" use:dragScroll aria-label="Search categories">
		{#each ['posts', 'projects', 'profiles', 'hashtags'] as item}<a
				href={'/app/search?' + new URLSearchParams({ q, tab: item })}
				aria-current={tab === item ? 'page' : undefined}
				>{item === 'hashtags'
					? 'Tags'
					: item[0].toUpperCase() + item.slice(1)}</a
			>{/each}
	</nav>{/if}
<Status
	{work}
	skeleton={tab === 'profiles' || !q ? 'people' : 'posts'}
	reload={() => load()}
/>
{#if !q}{#if recent.length}<section class="search-discovery">
			<h2>Recent searches</h2>
			{#each recent as item}<div class="recent-row">
					<a href={recentDestination(item)}>{item.displayLabel}</a><button
						aria-label="Remove recent search"
						disabled={work.busy}
						onclick={() =>
							void work.run(
								() => api.search.deleteRecent(item.id),
								() => (recent = recent.filter((old) => old.id !== item.id))
							)}><Icon name="close" size={18} /></button
					>
				</div>{/each}
		</section>{/if}
	{#if top?.groups.length}<section class="search-discovery stack">
			<h2>Trending hashtags</h2>
			{#each top.groups as group}<div class="stack">
					<h3>
						<span
							class="trending-craft"
							style:--craft-icon={`url('/craft_icons/${group.craftType.split('#').at(-1)}.svg')`}
							aria-hidden="true"
						></span>{label(group.craftType)}
					</h3>
					<div class="tag-chips">
						{#each group.items as tag}<a
								href={'/app/tags/' + encodeURIComponent(tag.tag)}>#{tag.tag}</a
							>{/each}
					</div>
				</div>{/each}
		</section>{/if}
	{#if authors.length}<h2 class="section-heading">New makers</h2>{/if}
{/if}
{#each authors as author}<a
		class="account-row maker-result"
		style={profileStyle(author.customisation)}
		href={'/app/profile/' + encodeURIComponent(author.did)}
		onclick={() => remember(author)}
		><Avatar author={authorValue(author)} /><CompactIdentity {author} />
		></a
	>{/each}
{#each posts as post}<MemberPost value={post} {api} {did} />{/each}
{#each tags as tag}<a
		class="account-row tag-result"
		href={'/app/tags/' + encodeURIComponent(tag.tag)}
		><span class="hashtag-icon">#</span><span
			><strong>#{tag.tag}</strong><small>{tag.postCount} posts</small></span
		></a
	>{/each}
{#if q && !work.busy && !work.error && !authors.length && !posts.length && !tags.length}<EmptyState
		title="No results found"
		detail="Try another search."
		icon="search"
	/>{/if}
{#if cursor}<button
		class="button next"
		disabled={work.busy}
		onclick={() => load(true)}>Load more</button
	>{/if}

<style>
	.search-form {
		padding: 12px 16px;
		margin: 0;
		position: relative;
	}
	.search-input-row {
		display: flex;
		gap: 10px;
		align-items: center;
	}
	.search-field {
		min-width: 0;
		flex: 1;
		display: flex;
		align-items: center;
		gap: 8px;
		border: 1.5px solid var(--ink-3);
		border-radius: 14px;
		transform: translate(3px, 3px);
		margin-right: 3px;
		margin-bottom: 3px;
		transition:
			transform 120ms,
			box-shadow 120ms;
		padding: 0 10px;
		background: var(--surface);
	}
	.search-field:focus-within {
		outline: 0;
		border-color: var(--ink);
		outline: 2px solid var(--ink);
		outline-offset: -2px;
		box-shadow: 3px 3px var(--shadow);
		transform: none;
	}
	.search-field input,
	.search-field input:focus,
	.search-field input:focus-visible {
		min-width: 0;
		width: 100%;
		border: 0;
		margin: 0;
		padding: 12px 0;
		background: transparent;
		box-shadow: none;
		outline: 0;
	}
	@media (prefers-reduced-motion: reduce) {
		.search-field {
			transition: none;
		}
	}
	.search-field input::-webkit-search-cancel-button {
		display: none;
	}
	.search-field > :global(svg) {
		flex-shrink: 0;
		color: var(--ink-2);
	}
	.clear-search {
		border: 0;
		padding: 8px 0 8px 8px;
		min-width: 32px;
		background: transparent;
		box-shadow: none;
	}
	.search-submit {
		padding: 10px 12px;
		min-height: 44px;
		font-size: 14px;
	}
	.cancel-search {
		font-size: 14px;
		text-decoration: none;
		color: var(--primary);
	}
	.search-sort {
		display: flex;
		margin-top: 12px;
		align-items: center;
		justify-content: flex-end;
		font-size: 14px;
		gap: 8px;
	}
	.search-discovery {
		padding: 16px;
		border-bottom: 1px solid var(--hairline);
	}
	.search-discovery h2 {
		font-size: 16px;
		margin: 0 0 12px;
	}
	.search-discovery h3 {
		font-size: 14px;
		display: flex;
		align-items: center;
		gap: 8px;
		color: var(--ink);
		font-weight: 600;
	}
	.trending-craft {
		width: 18px;
		height: 18px;
		background: currentColor;
		mask: var(--craft-icon) center / contain no-repeat;
	}
	.tag-chips {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.tag-chips a {
		display: inline-flex;
		gap: 8px;
		border: 1px solid var(--hairline);
		background: var(--surface);
		border-radius: 20px;
		padding: 6px 10px;
		font-size: 13px;
		text-decoration: none;
	}
	.recent-row {
		display: flex;
		gap: 8px;
		align-items: center;
		min-height: 48px;
	}
	.recent-row a {
		display: flex;
		gap: 12px;
		align-items: center;
		text-decoration: none;
		color: var(--ink);
		flex: 1;
	}
	.recent-row button {
		border: 0;
		box-shadow: none;
		background: transparent;
		padding: 10px;
	}
	.maker-result,
	.tag-result {
		margin: 0;
		padding: 12px 16px;
		gap: 12px;
		border: 0;
		border-bottom: 1px solid var(--hairline);
		border-radius: 0;
		text-decoration: none;
		color: var(--ink);
	}

	.hashtag-icon {
		width: 44px;
		height: 44px;
		display: grid;
		place-items: center;
		background: var(--primary-soft);
		color: var(--primary);
		border-radius: 50%;
		font-size: 24px;
	}
	.suggestions {
		z-index: 20;
		position: relative;
		padding: 8px 0;
		margin-top: 8px;
		background: var(--surface);
		border: 1px solid var(--hairline);
		border-radius: 12px;
	}
	.suggestions a {
		display: block;
		padding: 10px 12px;
		text-decoration: none;
		color: var(--ink);
		font-size: 14px;
	}
	.suggestions .suggestion-profile {
		display: flex;
		align-items: center;
		gap: 12px;
	}
	@media (max-width: 600px) {
		.search-submit {
			position: absolute;
			width: 1px;
			height: 1px;
			padding: 0;
			overflow: hidden;
			clip-path: inset(50%);
		}
	}

	.search-form .search-field input#search {
		border: 0;
		box-shadow: none;
		border-radius: 0;
		padding: 15px 0;
		margin: 0;
		background: transparent;
		outline: 0;
	}
	.search-field {
		border-color: var(--ink-2);
	}
	.search-sort {
		flex-direction: row;
	}
	.search-discovery {
		border: 0;
	}
	.tag-chips a {
		border: 1.5px solid var(--ink);
		color: var(--ink);
		font-weight: 600;
		padding: 6px 12px;
	}
</style>
