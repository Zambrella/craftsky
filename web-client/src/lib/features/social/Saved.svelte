<script lang="ts">
	import { captureScrollAnchor } from '#lib/scroll-anchor.ts';
	import { reloadPages } from '#lib/pagination.ts';
	import { goto } from '$app/navigation';
	import { contextMenu } from './context-menu';
	import { subscriptionsEnabled } from '../shell/release-config';
	import { useSubscriptionFeature } from '../shell/subscription-feature';
	import PlusActionIcon from '../shell/PlusActionIcon.svelte';
	import { onDestroy, tick, untrack } from 'svelte';
	import { page } from '$app/state';
	import type { CraftskyAPI, SavedItem, SavedFolder } from '#lib/api/client.ts';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import MemberPost from './MemberPost.svelte';
	import { allFolders, normalizeSavedFolderName } from './saved-model';
	import Icon from '../shell/Icon.svelte';
	import EmptyState from './EmptyState.svelte';
	import ChoiceMenu from './ChoiceMenu.svelte';
	import LoadMore from '#lib/LoadMore.svelte';

	let { api, did }: { api: CraftskyAPI; did: string } = $props();
	const premium = useSubscriptionFeature();
	const hasPlus = $derived(premium.allowed());
	const work = new Work('social.Saved.load');
	const folderWork = new Work('social.Saved.load');
	const mutation = new Work('social.Saved.action');
	let savedElement = $state<HTMLElement>();
	let loadingMore = $state(false);
	let items = $state<SavedItem[]>([]),
		folders = $state<SavedFolder[]>([]),
		cursor = $state<string | null>(null),
		folderName = $state('');
	// AuthenticatedPage remounts this page when its path or search changes.
	const folderId = subscriptionsEnabled()
		? page.url.searchParams.get('folder') || ''
		: '';
	const sort =
		page.url.searchParams.get('sort') === 'oldest' ? 'oldest' : 'newest';
	const sortURL = new URL(page.url.href);
	sortURL.searchParams.delete('all');
	sortURL.searchParams.delete('unfiled');
	const sortOptions = [
		{
			value: 'newest',
			label: 'Newest',
			icon: 'clock',
			description: 'Most recently saved first'
		},
		{
			value: 'oldest',
			label: 'Oldest',
			icon: 'history',
			description: 'Earliest saved first'
		}
	];
	let folderDialog = $state<HTMLDialogElement>(null!),
		deleteDialog = $state<HTMLDialogElement>(null!),
		folderMode = $state<'create' | 'rename'>('create');
	const currentFolder = $derived(
		hasPlus ? folders.find((folder) => folder.id === folderId) : undefined
	);
	const inFolder = $derived(hasPlus && !!folderId);
	const overview = $derived(hasPlus && !folderId);
	let reloadRequested = false;

	async function editFolder(mode: 'create' | 'rename') {
		if (!(await premium.require('Folders'))) return;
		folderMode = mode;
		folderName = mode === 'rename' ? currentFolder?.name || '' : '';
		mutation.error = '';
		mutation.retry = undefined;
		folderDialog.showModal();
	}
	async function loadFolders() {
		if (!hasPlus) return;
		await folderWork.run(
			() => allFolders(api),
			(value) => {
				if (hasPlus) folders = value;
			}
		);
	}
	async function load(more = false) {
		if (work.busy) {
			if (!more) reloadRequested = true;
			return;
		}
		const useFolders = hasPlus;
		loadingMore = more;
		await Promise.all([
			work.run(
				async () => {
					const fetchPage = (next?: string | null) =>
						api.saved.list({
							cursor: next,
							folderId: useFolders && folderId ? folderId : undefined,
							unfiled: useFolders && !folderId,
							sort
						});
					return more
						? fetchPage(cursor)
						: reloadPages(fetchPage, 1, (item) => item.post.uri, items.length);
				},
				(value) => {
					if (useFolders !== hasPlus) return;
					items = more ? [...items, ...value.items] : value.items;
					cursor = value.cursor || null;
				}
			),
			!more && useFolders ? loadFolders() : undefined
		]);
		loadingMore = false;
		if (reloadRequested) {
			reloadRequested = false;
			void load();
		}
	}
	async function saveFolder() {
		if (!(await premium.require('Folders'))) return;
		const mode = folderMode;
		const input = folderName;
		await mutation.run(
			() => {
				const name = normalizeSavedFolderName(input);
				return mode === 'create'
					? api.saved.createFolder(name)
					: api.saved.renameFolder(folderId, name);
			},
			(folder) => {
				folders =
					mode === 'create'
						? [...folders, folder]
						: folders.map((value) => (value.id === folder.id ? folder : value));
				folderDialog.close();
			},
			mode === 'create' ? 'Folder created.' : 'Folder renamed.',
			mode === 'create' ? 'social.Saved.create' : 'social.Saved.rename'
		);
	}
	async function remove(deleteSaves: boolean) {
		if (!(await premium.require('Folders'))) return;
		await mutation.run(
			() => api.saved.deleteFolder(folderId, deleteSaves),
			() => {
				deleteDialog.close();
				void goto('/saved');
			},
			'Folder deleted.',
			'social.Saved.remove'
		);
	}

	// A purchase or expiry can change folder access without navigating away.
	$effect(() => {
		const unlocked = hasPlus;
		untrack(() => {
			if (!unlocked) folders = [];
			void load();
		});
	});
	onDestroy(() => {
		work.dispose();
		folderWork.dispose();
		mutation.dispose();
	});
	$effect.pre(() => {
		void items;
		if (!savedElement) return;
		const restore = untrack(() => captureScrollAnchor(savedElement!));
		void tick().then(restore);
	});
</script>

{#snippet sortMenu()}
	<ChoiceMenu
		label="Sort saved posts"
		value={sort}
		url={sortURL}
		triggerIcon="filter"
		options={sortOptions}
		disabled={work.busy}
	/>
{/snippet}

<div class="workspace-heading">
	<div class="saved-title">
		{#if inFolder}<a
				class="icon-button"
				href="/saved"
				aria-label="Back to saved posts"><Icon name="back" /></a
			>{/if}
		<h1>{inFolder ? currentFolder?.name || 'Saved posts' : 'Saved posts'}</h1>
	</div>
	<div class="saved-actions">
		{#if inFolder}
			{@render sortMenu()}
			<details class="folder-actions" use:contextMenu>
				<summary class="icon-button" aria-label="Folder actions"
					><Icon name="more" /></summary
				>
				<div class="menu-items">
					<button onclick={() => editFolder('rename')}
						><Icon name="edit" />Rename folder</button
					>
					<button
						class="delete-action"
						onclick={() => {
							mutation.error = '';
							mutation.retry = undefined;
							deleteDialog.showModal();
						}}><Icon name="trash" />Delete folder</button
					>
				</div>
			</details>
		{:else}
			<button
				class="icon-button"
				aria-label={subscriptionsEnabled()
					? 'New folder'
					: 'Folders is coming soon'}
				title="New folder"
				onclick={() => editFolder('create')}
				><PlusActionIcon name="newFolder" locked={!hasPlus} /></button
			>
		{/if}
	</div>
</div>
<Status {work} skeleton="management" reload={() => load()} />
{#if hasPlus && folderWork.error}
	<div class="folder-failure" role="alert">
		<span>Folders couldn't load.</span>
		<button disabled={folderWork.busy} onclick={loadFolders}>Retry</button>
	</div>
{/if}
{#if overview && (items.length || folders.length)}
	<section class="folders-overview" aria-labelledby="folders-heading">
		<div class="saved-section-heading">
			<h2 id="folders-heading">Folders</h2>
		</div>
		{#each folders as folder (folder.id)}
			<a href={'/saved/folder?folder=' + encodeURIComponent(folder.id)}
				><Icon name="folder" /><span>{folder.name}</span><Icon
					name="chevron"
					bold
				/></a
			>
		{/each}
	</section>
{/if}
<div bind:this={savedElement} class="saved-posts-section">
	{#if items.length && !inFolder}
		<div class="saved-section-heading saved-list-heading">
			<h2>{overview ? 'Unfiled' : 'Saved posts'}</h2>
			{@render sortMenu()}
		</div>
	{/if}
	{#each items as item (item.post.uri)}
		<div data-scroll-anchor={item.post.uri}>
			<MemberPost
				value={item.post}
				{api}
				{did}
				savedAt={item.savedAt}
				savedFolderId={item.folderId}
				onMoved={() => load()}
				onRemove={() => {
					items = items.filter((old) => old !== item);
				}}
			/>
		</div>
	{:else}
		{#if !work.busy && !work.error && !folderWork.busy && !folderWork.error && (!overview || !folders.length)}
			<EmptyState
				fill
				title={inFolder ? 'Nothing saved yet' : 'Saved posts'}
				detail={inFolder ? 'Saved posts' : 'Nothing saved yet'}
				icon={inFolder ? 'folder' : 'saved'}
			/>
		{/if}
	{/each}
	{#if cursor}
		<LoadMore
			loading={work.busy && loadingMore}
			disabled={work.busy}
			autoLoad={!work.error}
			pageKey={cursor}
			loadingLabel="Loading more saved posts"
			onclick={() => load(true)}
		/>
	{/if}
</div>
<dialog
	bind:this={folderDialog}
	aria-labelledby="folder-dialog-title"
	oncancel={(event) => {
		if (mutation.busy) event.preventDefault();
	}}
>
	<form
		class="stack"
		onsubmit={(event) => {
			event.preventDefault();
			void saveFolder();
		}}
	>
		<h2 id="folder-dialog-title">
			{folderMode === 'create' ? 'New folder' : 'Rename folder'}
		</h2>
		<label
			>Folder name<input
				name="folderName"
				bind:value={folderName}
				required
				disabled={mutation.busy}
			/></label
		>
		{#if mutation.error}<p class="field-error" role="alert">
				{mutation.error}
			</p>{/if}
		<div class="dialog-actions">
			<button
				type="button"
				disabled={mutation.busy}
				onclick={() => folderDialog.close()}>Cancel</button
			>
			<button class="button" disabled={mutation.busy || !folderName.trim()}
				>{folderMode === 'create' ? 'Create folder' : 'Rename folder'}</button
			>
		</div>
	</form>
</dialog>
<dialog
	bind:this={deleteDialog}
	aria-labelledby="delete-folder-title"
	oncancel={(event) => {
		if (mutation.busy) event.preventDefault();
	}}
>
	<h2 id="delete-folder-title">Delete folder</h2>
	<p>What should happen to the posts in this folder?</p>
	{#if mutation.error}<p class="field-error" role="alert">
			{mutation.error}
		</p>{/if}
	<div class="dialog-actions delete-actions">
		<button disabled={mutation.busy} onclick={() => deleteDialog.close()}
			>Cancel</button
		>
		<button disabled={mutation.busy} onclick={() => remove(false)}
			>Keep saved posts</button
		>
		<button
			class="button danger"
			disabled={mutation.busy}
			onclick={() => remove(true)}>Delete saved posts</button
		>
	</div>
</dialog>

<style>
	.saved-title,
	.saved-actions,
	.saved-section-heading {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.saved-title {
		min-width: 0;
	}
	.saved-title h1 {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.saved-actions {
		flex-shrink: 0;
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
	.saved-section-heading {
		position: sticky;
		top: var(--app-bar-height);
		z-index: 3;
		min-height: 48px;
		padding: 12px 16px 8px;
		background: var(--paper);
	}
	.saved-section-heading h2 {
		margin: 0;
		font-size: 16px;
		font-weight: 600;
	}
	.folders-overview > a {
		display: flex;
		gap: 16px;
		align-items: center;
		padding: 8px 16px;
		min-height: 56px;
		color: var(--ink);
		text-decoration: none;
	}
	.folders-overview > a:hover {
		background: color-mix(in srgb, var(--primary) 8%, transparent);
	}
	.folders-overview > a > span {
		font-size: 16px;
		overflow-wrap: anywhere;
		flex: 1;
	}
	.folders-overview > a > :global(svg) {
		flex-shrink: 0;
	}
	.saved-list-heading {
		justify-content: space-between;
	}
	.folder-failure {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		padding: 12px 16px;
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
		margin: 0;
	}
	.dialog-actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: flex-end;
		align-items: center;
		gap: 12px;
		margin-top: 20px;
	}
	.delete-actions {
		flex-direction: column;
		align-items: flex-end;
	}
	dialog .dialog-actions > button:not(.button) {
		background: transparent;
		border: 0;
		box-shadow: none;
		color: var(--primary);
	}
	dialog button.danger {
		background: var(--danger-fill);
		color: white;
	}
	dialog button.danger:hover {
		background: color-mix(in srgb, var(--danger-fill) 88%, black);
	}
</style>
