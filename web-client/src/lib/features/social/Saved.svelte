<script lang="ts">
	import { contextMenu } from './context-menu';
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
	import Icon from '../shell/Icon.svelte';
	import EmptyState from './EmptyState.svelte';
	let { api, did }: { api: CraftskyAPI; did: string } = $props();
	const work = new Work();
	let items = $state<SavedItem[]>([]),
		folders = $state<SavedFolder[]>([]),
		cursor = $state<string | null>(null),
		folderName = $state('');
	const folderId = page.url.searchParams.get('folder') || '';
	let folderDialog = $state<HTMLDialogElement>(null!),
		deleteDialog = $state<HTMLDialogElement>(null!),
		folderMode = $state<'create' | 'rename'>('create');
	const currentFolder = $derived(
		folders.find((folder) => folder.id === folderId)
	);
	function editFolder(mode: 'create' | 'rename') {
		folderMode = mode;
		folderName = mode === 'rename' ? currentFolder?.name || '' : '';
		folderDialog.showModal();
	}
	let folderSupport = $state(true),
		deleteSaves = $state(false);
	const unfiled = page.url.searchParams.get('unfiled') === 'true';
	const showAll = page.url.searchParams.get('all') === 'true';
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
					unfiled: !folderId && (unfiled || (folderSupport && !showAll)),
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
					folderDialog.close();
				}
			);
	}
	function rename() {
		const value = folderName.trim();
		if (value)
			void work.run(
				() => api.saved.renameFolder(folderId, value),
				(next) => {
					folders = folders.map((f) => (f.id === next.id ? next : f));
					folderDialog.close();
				}
			);
	}
	function remove() {
		void work.run(
			() => api.saved.deleteFolder(folderId, deleteSaves),
			() => location.assign('/app/saved')
		);
	}

	onMount(() => load());
	onDestroy(() => work.dispose());
</script>

<div class="workspace-heading">
	<div class="saved-title">
		{#if folderId || unfiled}<a
				class="icon-button"
				href="/app/saved"
				aria-label="Back to saved posts"><Icon name="back" /></a
			>{/if}
		<h1>
			{currentFolder?.name || (unfiled ? 'Unfiled posts' : 'Saved posts')}
		</h1>
	</div>
	{#if folderSupport}{#if folderId}<details
				class="folder-actions"
				use:contextMenu
			>
				<summary class="icon-button" aria-label="Folder actions"
					><Icon name="more" /></summary
				>
				<div class="menu-items">
					<button onclick={() => editFolder('rename')}
						><Icon name="edit" />Rename folder</button
					><button
						onclick={() => {
							deleteSaves = false;
							deleteDialog.showModal();
						}}><Icon name="trash" />Delete folder</button
					>
				</div>
			</details>{:else}<button
				class="icon-button"
				aria-label="Create folder"
				title="Create folder"
				disabled={work.busy}
				onclick={() => editFolder('create')}><Icon name="folder" /></button
			>{/if}{/if}
</div>
<Status {work} skeleton="posts" reload={() => load()} />
{#if folderSupport && !folderId && !unfiled && folders.length}<section
		class="folders-overview"
	>
		<h2>Folders</h2>
		{#each folders as folder}<a
				href={'/app/saved?folder=' + encodeURIComponent(folder.id)}
				><Icon name="folder" /><strong>{folder.name}</strong><span
					aria-hidden="true">›</span
				></a
			>{/each}
	</section>{/if}
<div class="saved-list-heading">
	<h2>
		{folderId
			? 'Saved posts'
			: folderSupport && !showAll
				? 'Unfiled'
				: 'Saved posts'}
	</h2>
	<form
		method="get"
		action="/app/saved"
		onchange={(event) => event.currentTarget.requestSubmit()}
	>
		{#if showAll}<input type="hidden" name="all" value="true" />{/if}
		{#if folderId}<input
				type="hidden"
				name="folder"
				value={folderId}
			/>{/if}{#if unfiled}<input
				type="hidden"
				name="unfiled"
				value="true"
			/>{/if}<label
			><Icon name="sort" size={18} /><span class="visually-hidden"
				>Sort saved posts</span
			><select
				aria-label="Sort saved posts"
				name="sort"
				value={page.url.searchParams.get('sort') || 'newest'}
				><option value="newest">Newest</option><option value="oldest"
					>Oldest</option
				></select
			></label
		>
	</form>
</div>
{#if folderSupport && !folderId}<nav
		class="saved-scope"
		aria-label="Saved posts"
	>
		<a href="/app/saved?all=true" aria-current={showAll ? 'page' : undefined}
			>All saved</a
		><a
			href="/app/saved?unfiled=true"
			aria-current={!showAll ? 'page' : undefined}>Unfiled</a
		>
	</nav>{/if}
{#each items as item}<MemberPost
		value={item.post}
		{api}
		{did}
		savedAt={item.savedAt}
		onMoved={() => load()}
		onRemove={() => {
			items = items.filter((old) => old !== item);
		}}
	/>{:else}{#if !work.busy && !work.error}<EmptyState
			title="Saved posts"
			detail="Your saved inspiration will appear here."
			icon="saved"
		/>{/if}{/each}
{#if cursor}<button
		class="button next"
		disabled={work.busy}
		onclick={() => load(true)}>Load more</button
	>{/if}
<dialog bind:this={folderDialog} aria-labelledby="folder-dialog-title">
	<form
		class="stack"
		onsubmit={(event) => {
			event.preventDefault();
			folderMode === 'create' ? create() : rename();
		}}
	>
		<h2 id="folder-dialog-title">
			{folderMode === 'create' ? 'Create folder' : 'Rename folder'}
		</h2>
		<label
			>Folder name<input
				bind:value={folderName}
				maxlength="80"
				required
			/></label
		>
		<div class="dialog-actions">
			<button type="button" onclick={() => folderDialog.close()}>Cancel</button
			><button class="button" disabled={work.busy || !folderName.trim()}
				>{folderMode === 'create' ? 'Create folder' : 'Save name'}</button
			>
		</div>
	</form>
</dialog>
<dialog bind:this={deleteDialog} aria-labelledby="delete-folder-title">
	<h2 id="delete-folder-title">Delete folder?</h2>
	<p>Saved posts will remain unfiled unless you choose to remove them too.</p>
	<label class="check"
		><input type="checkbox" bind:checked={deleteSaves} />Remove the saved posts
		too</label
	>
	<div class="dialog-actions">
		<button onclick={() => deleteDialog.close()}>Cancel</button><button
			class="danger"
			disabled={work.busy}
			onclick={remove}>Delete folder</button
		>
	</div>
</dialog>

<style>
	.saved-title,
	.saved-list-heading,
	.saved-list-heading label {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.saved-title h1 {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.folder-actions {
		position: relative;
	}
	.folder-actions summary {
		list-style: none;
		cursor: pointer;
	}
	.folder-actions button {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.folders-overview {
		border-bottom: 1px dashed var(--hairline);
		padding: 12px 0;
	}
	.folders-overview h2 {
		padding: 0 16px;
		margin: 0 0 8px;
		font-size: 18px;
	}
	.folders-overview > a {
		display: flex;
		gap: 16px;
		align-items: center;
		padding: 14px 20px;
		min-height: 52px;
		color: var(--ink);
		text-decoration: none;
	}
	.folders-overview > a:hover {
		background: var(--paper-2);
	}
	.folders-overview strong {
		font-size: 16px;
		font-weight: 500;
		flex: 1;
	}
	.folders-overview > a > :global(svg) {
		color: var(--primary);
	}
	.saved-list-heading {
		padding: 16px;
		justify-content: space-between;
	}
	.saved-list-heading h2 {
		margin: 0;
		font-size: 18px;
	}
	.saved-list-heading select {
		width: auto;
		margin: 0;
		max-width: 175px;
		font-size: 13px;
	}
	.saved-scope {
		display: flex;
		gap: 16px;
		padding: 0 16px 16px;
		font-size: 14px;
	}
	.saved-scope a {
		text-decoration: none;
		color: var(--ink-2);
	}
	.saved-scope a[aria-current] {
		color: var(--primary);
		font-weight: 600;
	}
	dialog {
		width: min(420px, calc(100vw - 32px));
	}
	dialog h2 {
		font-size: 22px;
		margin: 0 0 16px;
	}
	dialog p {
		font-size: 15px;
		margin-bottom: 16px;
	}
	.dialog-actions {
		display: flex;
		justify-content: flex-end;
		gap: 12px;
		margin-top: 20px;
	}

	.saved-list-heading label {
		flex-direction: row;
	}
</style>
