<script lang="ts">
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
	import { name } from './model';
	import { recentDestination } from './search-model';
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
	const tab = page.url.searchParams.get('tab') || 'projects';
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
</script>

<div class="workspace-heading"><h1>Discover</h1></div>
<form class="panel search-form" method="get" action="/app/search">
	<label for="search">Search CraftSky</label>
	<div class="row">
		<input
			id="search"
			name="q"
			type="search"
			bind:value={query}
			oninput={suggest}
			placeholder="Patterns, projects, people…"
		/><button class="button">Search</button>
	</div>
	<input type="hidden" name="tab" value={tab} />
	{#if suggestions}<div class="suggestions">
			{#each suggestions.profiles.items as author}<a
					href={'/app/profile/' + encodeURIComponent(author.did)}
					onclick={() => {
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
					}}>{name(author)} · @{author.handle}</a
				>{/each}{#each suggestions.hashtags.items as tag}<a
					href={'/app/tags/' + encodeURIComponent(tag.tag)}>#{tag.tag}</a
				>{/each}
		</div>{/if}
	{#if tab === 'posts'}<label
			>Sort<select
				name="sort"
				value={page.url.searchParams.get('sort') || 'chronological'}
				><option value="chronological">Newest first</option><option
					value="popular">Popular</option
				></select
			></label
		>{/if}
</form>
<nav class="tabs" aria-label="Search categories">
	{#each ['projects', 'posts', 'profiles', 'hashtags'] as item}<a
			href={'/app/search?' + new URLSearchParams({ q, tab: item })}
			aria-current={tab === item ? 'page' : undefined}
			>{item[0].toUpperCase() + item.slice(1)}</a
		>{/each}
</nav>
<Status {work} />
{#if !q}<section class="panel">
		<h2>Recently searched</h2>
		{#each recent as item}<div class="row">
				<a href={recentDestination(item)}>{item.displayLabel}</a><button
					aria-label="Remove recent search"
					onclick={() =>
						void work.run(
							() => api.search.deleteRecent(item.id),
							() => (recent = recent.filter((old) => old.id !== item.id))
						)}>×</button
				>
			</div>{:else}<p>No recent searches yet.</p>{/each}
	</section>
	<section class="panel stack">
		<h2>Popular hashtags</h2>
		{#each top?.groups || [] as group}<div class="stack">
				<h3>{group.craftType.split('#').pop()}</h3>
				<div class="actions">
					{#each group.items as tag}<a
							href={'/app/tags/' + encodeURIComponent(tag.tag)}
							>#{tag.tag} <small>{tag.count}</small></a
						>{/each}
				</div>
			</div>{/each}
	</section>
	<h2 class="section-heading">New makers</h2>{/if}
{#each authors as author}<a
		class="panel account-row"
		href={'/app/profile/' + encodeURIComponent(author.did)}
		onclick={() => {
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
		}}
		><Avatar author={authorValue(author)} /><span
			><strong>{name(author)}</strong><small>@{author.handle}</small>
			<p>{author.description || ''}</p></span
		></a
	>{/each}
{#each posts as post}<MemberPost value={post} {api} {did} />{/each}
{#each tags as tag}<a
		class="panel account-row"
		href={'/app/tags/' + encodeURIComponent(tag.tag)}
		><strong>#{tag.tag}</strong><span>{tag.postCount} posts</span></a
	>{/each}
{#if q && !work.busy && !work.error && !authors.length && !posts.length && !tags.length}<p
		class="empty"
	>
		No results yet. Try another search.
	</p>{/if}
{#if cursor}<button
		class="button next"
		disabled={work.busy}
		onclick={() => load(true)}>Load more</button
	>{/if}
