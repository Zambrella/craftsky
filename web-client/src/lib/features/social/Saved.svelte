<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import {
		APIError,
		type CraftskyAPI,
		type SavedItem,
		type SavedFolder
	} from '#lib/api/client.ts';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import MemberPost from './MemberPost.svelte';
	import { allFolders } from './saved-model';
	let { api, did }: { api: CraftskyAPI; did: string } = $props();
	const work = new Work();
	let items = $state<SavedItem[]>([]),
		folders = $state<SavedFolder[]>([]),
		cursor = $state<string | null>(null),
		folderName = $state('');
	const folderId = page.url.searchParams.get('folder') || '';
	let folderSupport = $state(true),
		deleteSaves = $state(false);
	const unfiled = page.url.searchParams.get('unfiled') === 'true';
	async function load(more = false) {
		await work.run(
			async () => {
				if (!more)
					try {
						folders = await allFolders(api);
					} catch (error) {
						if (!(error instanceof APIError) || error.status !== 403)
							throw error;
						folderSupport = false;
					}
				return api.saved.list({
					cursor: more ? cursor : undefined,
					folderId: folderId || undefined,
					unfiled: page.url.searchParams.get('unfiled') === 'true',
					sort:
						page.url.searchParams.get('sort') === 'oldest' ? 'oldest' : 'newest'
				});
			},
			(value) => {
				items = more ? [...items, ...value.items] : value.items;
				cursor = value.cursor || null;
			}
		);
	}
	function create() {
		const value = folderName.trim();
		if (value)
			void work.run(
				() => api.saved.createFolder(value),
				(folder) => {
					folders = [...folders, folder];
					folderName = '';
				}
			);
	}
	function rename() {
		const folder = folders.find((f) => f.id === folderId);
		const value = prompt('Folder name', folder?.name);
		if (value?.trim())
			void work.run(
				() => api.saved.renameFolder(folderId, value.trim()),
				(next) => (folders = folders.map((f) => (f.id === next.id ? next : f)))
			);
	}
	function remove() {
		if (
			confirm(
				deleteSaves
					? 'Delete this folder and remove its saved posts?'
					: 'Delete this folder? Saved posts will remain unfiled.'
			)
		)
			void work.run(
				() => api.saved.deleteFolder(folderId, deleteSaves),
				() => location.assign('/app/saved')
			);
	}
	onMount(() => load());
	onDestroy(() => work.dispose());
</script>

<div class="workspace-heading"><h1>Saved</h1></div>
<nav class="tabs" aria-label="Saved folders">
	<a href="/app/saved" aria-current={!folderId && !unfiled ? 'page' : undefined}
		>All saved</a
	>{#if folderSupport}<a href="/app/saved?unfiled=true">Unfiled</a
		>{/if}{#each folders as folder}<a
			href={'/app/saved?folder=' + encodeURIComponent(folder.id)}
			aria-current={folderId === folder.id ? 'page' : undefined}
			>{folder.name}</a
		>{/each}
</nav>
<form class="toolbar" method="get" action="/app/saved">
	{#if folderId}<input
			type="hidden"
			name="folder"
			value={folderId}
		/>{/if}{#if unfiled}<input
			type="hidden"
			name="unfiled"
			value="true"
		/>{/if}<label
		>Sort saved posts<select
			name="sort"
			value={page.url.searchParams.get('sort') || 'newest'}
			><option value="newest">Newest saved first</option><option value="oldest"
				>Oldest saved first</option
			></select
		></label
	><button>Apply</button>
</form>
{#if folderSupport}<form
		class="panel inline-form"
		onsubmit={(event) => {
			event.preventDefault();
			create();
		}}
	>
		<label
			>New folder<input
				bind:value={folderName}
				maxlength="80"
				required
			/></label
		><button disabled={work.busy}>Create folder</button>{#if folderId}<button
				type="button"
				onclick={rename}>Rename folder</button
			><label class="check"
				><input type="checkbox" bind:checked={deleteSaves} />Remove the saved
				posts too</label
			><button type="button" onclick={remove}>Delete folder</button>{/if}
	</form>{/if}<Status {work} />{#each items as item}<MemberPost
		value={item.post}
		{api}
		{did}
	/>{:else}{#if !work.busy}<p class="empty">
			Your saved inspiration will appear here.
		</p>{/if}{/each}{#if cursor}<button
		class="button next"
		onclick={() => load(true)}>Load more</button
	>{/if}
