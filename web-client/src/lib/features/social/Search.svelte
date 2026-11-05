<script lang="ts">
	import { dragScroll } from '#lib/drag-scroll.ts';
	import { onDestroy, getContext, untrack } from 'svelte';
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
	import { goto, beforeNavigate } from '$app/navigation';
	import Icon from '../shell/Icon.svelte';
	import EmptyState from './EmptyState.svelte';
	import CompactIdentity from './CompactIdentity.svelte';
	import {
		PageHistory,
		pageHistoryContext,
		type PageView
	} from '../shell/page-history';
	import { label } from '#lib/public.ts';
	let { api, did }: { api: CraftskyAPI; did: string } = $props();
	type SearchSnapshot = {
		authors: Author[];
		posts: PostPresentation[];
		tags: Hashtag[];
		recent: RecentSearch[];
		top?: TopHashtags;
		cursor: string | null;
		pages: number;
	};
	const history = getContext<PageHistory | undefined>(pageHistoryContext);
	let historyView: PageView<SearchSnapshot> | undefined;
	let work = $state(new Work());
	let loaded = false;
	let loadedPages = 0;
	let query = $state(page.url.searchParams.get('q') || ''),
		suggestions = $state<SearchSuggestions>();
	let suggesting = $state(false);
	let suggestionWork = $state(new Work());
	let waitingForSuggestions = $state(false);
	let suggestionController: AbortController | undefined;
	const showSuggestions = $derived(suggesting && !!query.trim());
	let top = $state<TopHashtags>();
	let authors = $state<Author[]>([]),
		posts = $state<PostPresentation[]>([]),
		tags = $state<Hashtag[]>([]),
		recent = $state<RecentSearch[]>([]),
		cursor = $state<string | null>(null);
	const tab = $derived(page.url.searchParams.get('tab') || 'posts');
	const q = $derived(page.url.searchParams.get('q') || '');
	let timer: ReturnType<typeof setTimeout>;
	let suggestionGeneration = 0;
	function resetSuggestions() {
		clearTimeout(timer);
		suggestionGeneration++;
		suggestionController?.abort();
		suggestionWork.dispose();
		suggestionWork = new Work();
		suggestions = undefined;
		waitingForSuggestions = false;
	}
	function suggest() {
		resetSuggestions();
		const term = query.trim();
		suggesting = !!term;
		if (!term) return;
		const generation = suggestionGeneration;
		const pending = suggestionWork;
		const controller = new AbortController();
		suggestionController = controller;
		waitingForSuggestions = true;
		timer = setTimeout(() => {
			waitingForSuggestions = false;
			void pending.run(
				() =>
					api.search.suggestions(term, {
						profileLimit: 5,
						hashtagLimit: 5,
						signal: controller.signal
					}),
				(value) => {
					if (generation === suggestionGeneration) suggestions = value;
				}
			);
		}, 300);
	}

	function saveView() {
		if (loaded)
			historyView?.save({
				authors,
				posts,
				tags,
				recent,
				top,
				cursor,
				pages: loadedPages
			});
	}
	beforeNavigate(() => {
		saveView();
		suggesting = false;
		resetSuggestions();
	});
	async function load(more = false) {
		const term = q,
			category = tab,
			pending = work;
		const options = { q: term, cursor: more ? cursor : undefined, limit: 30 };
		await pending.run(
			async (): Promise<
				Partial<SearchSnapshot> & { cursor?: string | null }
			> => {
				if (!term) {
					const [members, history, hashtags] = await Promise.all([
						api.profiles.recent(),
						api.search.recent(),
						api.search.topHashtags()
					]);
					return {
						authors: members.items,
						recent: history.items,
						top: hashtags
					};
				}
				if (category === 'profiles') {
					const result = await api.search.profiles(term, options);
					return { authors: result.items, cursor: result.cursor };
				}
				if (category === 'hashtags') {
					const result = await api.search.hashtags(term, options);
					return { tags: result.items, cursor: result.cursor };
				}
				const result =
					category === 'posts'
						? await api.search.posts(options)
						: await api.search.projects(options);
				return { posts: result.items, cursor: result.cursor };
			},
			(value) => {
				if (value.authors)
					authors = more ? [...authors, ...value.authors] : value.authors;
				if (value.posts)
					posts = more ? [...posts, ...value.posts] : value.posts;
				if (value.tags) tags = more ? [...tags, ...value.tags] : value.tags;
				if (value.recent) recent = value.recent;
				if (value.top) top = value.top;
				cursor = value.cursor || null;
				loadedPages = more ? loadedPages + 1 : 1;
				loaded = true;
				saveView();
			}
		);
	}
	$effect(() => {
		const term = q;
		untrack(() => {
			query = term;
			suggesting = false;
			resetSuggestions();
		});
		if (term)
			void api.search
				.saveRecent({ type: 'query', displayLabel: term, payload: { q: term } })
				.catch(() => {});
	});
	$effect(() => {
		// Even selecting the active/default tab can create a new history entry.
		page.url.search;
		const key = '/search?' + new URLSearchParams({ q, tab });
		return untrack(() => {
			const view = history?.view<SearchSnapshot>('search-results', key);
			historyView = view;
			const snapshot = view?.cachedTab ? view.value : undefined;
			const pending = new Work();
			let active = true;
			work = pending;
			loaded = !!snapshot;
			loadedPages = snapshot?.pages || 0;
			authors = snapshot?.authors || [];
			posts = snapshot?.posts || [];
			tags = snapshot?.tags || [];
			recent = snapshot?.recent || [];
			top = snapshot?.top;
			cursor = snapshot?.cursor || null;
			void (async () => {
				if (snapshot) pending.hasLoaded = true;
				else {
					const pages = view?.value?.pages || 1;
					await load();
					while (
						active &&
						pending === work &&
						cursor &&
						!pending.error &&
						loadedPages < pages
					)
						await load(true);
				}
				view?.ready();
			})();
			return () => {
				active = false;
				pending.dispose();
				view?.ready();
			};
		});
	});
	onDestroy(() => {
		saveView();
		work.dispose();
		clearTimeout(timer);
		suggestionGeneration++;
		suggestionController?.abort();
		suggestionWork.dispose();
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
					crafts: author.crafts ?? []
				}
			})
			.catch(() => {});
	}
</script>

<div class="workspace-heading"><h1>Search</h1></div>
<form
	class="search-form"
	method="get"
	action="/search"
	onsubmit={(event) => {
		event.preventDefault();
		const target = searchDestination(query);
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
						suggesting = false;
						resetSuggestions();
						document.getElementById('search')?.focus();
					}}
					><span class="clear-circle"><Icon name="close" size={14} bold /></span
					></button
				>{/if}
		</div>
		<button type="submit" class="visually-hidden" tabindex="-1">Search</button
		>{#if query || q}<a
				class="cancel-search"
				href="/search"
				onclick={(event) => {
					if (!q) event.preventDefault();
					query = '';
					suggesting = false;
					resetSuggestions();
				}}>Cancel</a
			>{/if}
	</div>
	<input type="hidden" name="tab" value={tab} />
	{#if showSuggestions}
		<div class="suggestions">
			{#if waitingForSuggestions}<p role="status">Searching…</p>{/if}
			<Status work={suggestionWork} skeleton="people" reload={suggest} />
			{#if suggestions}
				{#if suggestions.profiles.items?.length}
					<section aria-labelledby="suggestion-profiles-heading">
						<div class="suggestion-heading">
							<h2 id="suggestion-profiles-heading">Profiles</h2>
							{#if suggestions.profiles.hasMore}<a
									class="view-all"
									href={'/search?' +
										new URLSearchParams({ q: query.trim(), tab: 'profiles' })}
									>View all</a
								>{/if}
						</div>
						{#each suggestions.profiles.items as author}<a
								class="suggestion-profile account-row"
								href={'/profiles/' + encodeURIComponent(author.did)}
								onclick={() => remember(author)}
								><Avatar author={authorValue(author)} /><CompactIdentity
									{author}
								/></a
							>{/each}
					</section>
				{/if}
				{#if suggestions.hashtags.items?.length}
					<section aria-labelledby="suggestion-hashtags-heading">
						<div class="suggestion-heading">
							<h2 id="suggestion-hashtags-heading">Hashtags</h2>
							{#if suggestions.hashtags.hasMore}<a
									class="view-all"
									href={'/search?' +
										new URLSearchParams({ q: query.trim(), tab: 'hashtags' })}
									>View all</a
								>{/if}
						</div>
						{#each suggestions.hashtags.items as tag}<a
								class="suggestion-tag account-row"
								href={'/search/tags?tag=' + encodeURIComponent(tag.tag)}
								><span class="hashtag-icon" aria-hidden="true">#</span><span
									><span>#{tag.tag}</span><small
										>{tag.postCount}
										{tag.postCount === 1 ? 'post' : 'posts'}</small
									></span
								></a
							>{/each}
					</section>
				{/if}
				{#if !suggestions.profiles.items?.length && !suggestions.hashtags.items?.length}<EmptyState
						title="No results found"
						detail="Try another search."
						icon="search"
					/>{/if}
			{/if}
		</div>
	{/if}
</form>
{#if !showSuggestions}
	{#if q}<nav class="tabs" use:dragScroll aria-label="Search categories">
			{#each ['posts', 'projects', 'profiles', 'hashtags'] as item}<a
					href={'/search?' + new URLSearchParams({ q, tab: item })}
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
									href={'/search/tags?tag=' + encodeURIComponent(tag.tag)}
									>#{tag.tag}</a
								>{/each}
						</div>
					</div>{/each}
			</section>{/if}
		{#if authors.length}<h2 class="section-heading">New makers</h2>{/if}
	{/if}
	{#each authors as author}<a
			class="account-row maker-result"
			style={profileStyle(author.customisation)}
			href={'/profiles/' + encodeURIComponent(author.did)}
			onclick={() => remember(author)}
			><Avatar author={authorValue(author)} /><CompactIdentity {author} />
			></a
		>{/each}
	{#each posts as post}<MemberPost value={post} {api} {did} />{/each}
	{#each tags as tag}<a
			class="account-row tag-result"
			href={'/search/tags?tag=' + encodeURIComponent(tag.tag)}
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
{/if}

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

	.cancel-search {
		font-size: 14px;
		font-weight: 600;
		text-decoration: none;
		color: var(--primary);
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
		padding: 8px 0;
		gap: 24px;
		background: transparent;
		border: 0;
	}
	.suggestion-heading {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		margin-bottom: 8px;
	}
	.suggestion-heading h2 {
		font-size: 16px;
		margin: 0;
	}
	.suggestions a {
		text-decoration: none;
		color: var(--ink);
	}
	.suggestions .view-all {
		padding: 8px 0 8px 12px;
		color: var(--primary);
		font-weight: 600;
	}
	.suggestions .account-row {
		padding: 8px 16px;
		gap: 12px;
		min-height: 60px;
	}
	.suggestions .suggestion-tag {
		padding: 12px 0;
		gap: 16px;
	}
	.suggestion-tag {
		font-size: 16px;
	}
	.suggestion-tag small {
		font-size: 14px;
		margin-top: 2px;
	}
	.hashtag-icon {
		flex-shrink: 0;
		color: var(--ink);
	}
	.clear-circle {
		width: 20px;
		height: 20px;
		border: 2px solid currentColor;
		border-radius: 50%;
		display: grid;
		place-items: center;
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
