<script lang="ts">
	import { userErrorMessage } from '#lib/user-facing-error.ts';
	import { reportFailure } from '../diagnostics/client';
	import Snackbar from '../social/Snackbar.svelte';
	import EmptyState from '../social/EmptyState.svelte';
	import { getContext, onDestroy, onMount, untrack } from 'svelte';
	import type { CraftskyAPI, ScheduledPostSummary } from '#lib/api/client.ts';
	import { drafts, type Draft } from './store.ts';
	import { draftPreview, draftURL } from './presentation';
	import DraftThumbnail from './DraftThumbnail.svelte';
	import Icon from '../shell/Icon.svelte';
	const createPost = getContext<(event: MouseEvent) => void>(
		'craftsky-create-post'
	);
	const uid = $props.id();
	let deleteDialog: HTMLDialogElement;
	let deletion = $state<{ id: string; remote: boolean; owner: string }>();
	let {
		api,
		path,
		session
	}: {
		api: CraftskyAPI;
		path: string[];
		session: { id: string; did: string; handle: string };
	} = $props();
	let local = $state<Draft[]>([]),
		scheduled = $state<ScheduledPostSummary[]>([]),
		loading = $state(true),
		busy = $state(false),
		error = $state(''),
		notice = $state(''),
		mounted = $state(false);
	let generation = 0;
	let destroyed = false;
	const isScheduled = $derived(
		path[0] === 'scheduled' || path.includes('scheduled')
	);
	$effect(() => {
		const owner = session.did;
		const tab = isScheduled;
		if (mounted)
			untrack(() => {
				deleteDialog?.close();
				deletion = undefined;
				busy = false;
				void load(owner, tab);
			});
	});
	onMount(() => {
		mounted = true;
	});
	onDestroy(() => {
		destroyed = true;
		generation++;
	});
	async function load(owner = session.did, tab = isScheduled) {
		const seq = ++generation;
		loading = true;
		error = '';
		try {
			if (tab) {
				const result = await api.schedules.list();
				if (seq === generation && !destroyed) scheduled = result.items;
			} else {
				const result = await drafts.list(owner);
				if (seq === generation && !destroyed) local = result;
			}
		} catch (e) {
			void reportFailure(e, 'drafts.manage');
			if (seq === generation && !destroyed) error = userErrorMessage(e);
		} finally {
			if (seq === generation && !destroyed) loading = false;
		}
	}
	function askRemove(id: string, remote: boolean) {
		if (busy) return;
		deletion = { id, remote, owner: session.did };
		deleteDialog.showModal();
	}
	async function remove() {
		const target = deletion;
		if (busy || !target || target.owner !== session.did) return;
		const { id, remote, owner } = target;
		const seq = generation;
		busy = true;
		error = '';
		try {
			if (remote) await api.schedules.delete(id);
			else await drafts.remove(owner, id);
			if (seq === generation && !destroyed) {
				deleteDialog.close();
				deletion = undefined;
				notice = remote ? 'Scheduled post deleted.' : 'Draft deleted.';
				busy = false;
				await load();
			}
		} catch (e) {
			void reportFailure(e, 'drafts.manage');
			if (seq === generation && !destroyed) error = userErrorMessage(e);
		} finally {
			if (seq === generation && !destroyed) busy = false;
		}
	}
	const date = (value: string) =>
		new Date(value).toLocaleString(undefined, {
			dateStyle: 'medium',
			timeStyle: 'short'
		});
</script>

<section class="draft-workspace" aria-busy={loading || busy}>
	<header class="workspace-header draft-appbar">
		<h1>{isScheduled ? 'Scheduled posts' : 'Drafts'}</h1>
	</header>
	{#if error}<Snackbar message={error} warning onDismiss={() => (error = '')}
			><button type="button" onclick={() => (deletion ? remove() : load())}
				>Try again</button
			></Snackbar
		>{/if}
	{#if notice}<Snackbar message={notice} onDismiss={() => (notice = '')} />{/if}
	{#if loading}<div
			class="draft-skeleton"
			role="status"
			aria-label="Loading drafts"
		>
			{#each [1, 2, 3, 4] as row}<div><span></span><i></i></div>{/each}
		</div>
	{:else if isScheduled}
		{#if !scheduled.length}<EmptyState
				fill
				icon="scheduled"
				title="Scheduled posts"
				detail="No scheduled posts"
			/>{:else}<p class="privacy-note">
				Scheduled posts and their media stay private until publication.
			</p>{/if}
		<ul class="draft-list">
			{#each scheduled as item (item.id)}<li class="draft-row scheduled-row">
					<a
						class="draft-main"
						href={`/compose?schedule=${encodeURIComponent(item.id)}`}
					>
						<span class="schedule-icon" aria-hidden="true"
							><Icon name="clock" size={32} /></span
						>
						<span class="copy"
							><strong
								>{item.projectTitle ||
									item.textPreview ||
									'Untitled post'}</strong
							>
							<span
								>{item.kind === 'project' ? 'Project' : 'Post'} · {date(
									item.scheduledAt
								)}</span
							>
							<span class="schedule-status"
								>{item.status.replaceAll('_', ' ')}</span
							>
							{#if item.lastErrorCode || item.subscriptionRequired}<span
									class="attention"
									>{item.subscriptionRequired
										? 'CraftSky Plus or Business is required.'
										: `Needs attention: ${item.lastErrorCode?.replaceAll('_', ' ')}`}</span
								>{/if}
							{#if item.needsAttentionExpiresAt}<span
									>Review before {date(item.needsAttentionExpiresAt)}</span
								>{/if}
						</span>
					</a>
					<div class="row-actions">
						<a
							class="icon-action"
							href={`/compose?schedule=${encodeURIComponent(item.id)}`}
							aria-label="Edit scheduled post"><Icon name="edit" /></a
						><button
							class="icon-action"
							aria-label="Delete scheduled post"
							disabled={busy}
							onclick={() => askRemove(item.id, true)}
							><Icon name="trash" /></button
						>
					</div>
				</li>{/each}
		</ul>
	{:else}
		{#if !local.length}<EmptyState
				fill
				icon="drafts"
				title="Drafts"
				detail="No drafts"
			/>{:else}<p class="privacy-note">
				Saved on this device for @{session.handle}. Drafts do not sync between
				devices.
			</p>{/if}
		<ul class="draft-list">
			{#each local as item (item.id)}{@const preview = draftPreview(item)}
				<li class="draft-row">
					<a class="draft-main" href={draftURL(item)}
						><DraftThumbnail draft={item} /><span class="copy"
							><strong>{preview.title}</strong>{#if preview.body}<span
									class="story">{preview.body}</span
								>{/if}<span>{preview.kind} · {date(item.updatedAt)}</span></span
						></a
					>
					<div class="row-actions">
						<a
							class="icon-action"
							href={draftURL(item)}
							aria-label={item.pending ? 'Check action' : 'Continue editing'}
							><Icon name="edit" /></a
						>{#if !item.pending}<button
								class="icon-action"
								aria-label="Delete draft"
								disabled={busy}
								onclick={() => askRemove(item.id, false)}
								><Icon name="trash" /></button
							>{/if}
					</div>
				</li>{/each}
		</ul>
	{/if}
</section>
<dialog
	bind:this={deleteDialog}
	class="draft-dialog"
	aria-labelledby={uid + '-delete-title'}
	oncancel={(event) => {
		if (busy) event.preventDefault();
		else deletion = undefined;
	}}
>
	<h2 id={uid + '-delete-title'}>
		{deletion?.remote ? 'Delete scheduled post?' : 'Delete draft?'}
	</h2>
	<p>
		{deletion?.remote
			? 'This scheduled post will be deleted and will not be published.'
			: 'This draft and its media will be deleted from this device. This cannot be undone.'}
	</p>
	<div class="dialog-actions">
		<button
			disabled={busy}
			onclick={() => {
				deletion = undefined;
				deleteDialog.close();
			}}>Cancel</button
		><button class="danger button" disabled={busy} onclick={remove}
			>{busy ? 'Deleting…' : 'Delete'}</button
		>
	</div>
</dialog>

<style>
	.draft-workspace {
		width: 100%;
		margin: auto;
		min-height: 70vh;
		background: var(--paper);
	}
	.draft-appbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		min-height: 56px;
		padding: 8px 16px;
	}
	h1 {
		margin: 0;
		font-size: 20px;
		font-weight: 700;
		font-family: inherit;
	}
	.draft-list {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.draft-row {
		display: flex;
		align-items: center;
		padding: 4px 8px 4px 16px;
		min-height: 84px;
		gap: 4px;
	}
	.draft-row:hover {
		background: color-mix(in srgb, var(--primary) 8%, transparent);
	}
	.draft-main {
		display: flex;
		flex: 1;
		gap: 16px;
		align-items: center;
		padding: 12px 0;
		min-width: 0;
		color: var(--ink);
		text-decoration: none;
	}
	.copy {
		display: grid;
		gap: 4px;
		min-width: 0;
	}
	.copy strong,
	.story {
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
		overflow-wrap: anywhere;
	}
	.copy strong {
		font-size: 16px;
		font-weight: 500;
		line-height: 1.4;
	}
	.copy span {
		color: var(--ink-3);
		font-size: 12px;
		line-height: 1.5;
	}
	.copy .story {
		font-size: 14px;
	}
	.row-actions {
		display: flex;
		flex: none;
	}
	.icon-action {
		width: 44px;
		height: 44px;
		padding: 11px;
		display: inline-grid;
		place-items: center;
		border: 0;
		border-radius: 50%;
		background: transparent;
		color: var(--ink);
		font: inherit;
		font-size: 24px;
		cursor: pointer;
		text-decoration: none;
	}
	.icon-action:hover {
		background: color-mix(in srgb, var(--primary) 8%, transparent);
	}
	.schedule-icon {
		color: var(--primary);
		font-size: 36px;
		width: 48px;
		text-align: center;
		flex: none;
	}
	.copy .schedule-status {
		color: var(--primary);
		text-transform: capitalize;
	}
	.copy .attention {
		color: var(--red-ink);
	}
	.privacy-note {
		padding: 12px 16px;
		color: var(--ink-3);
		font-size: 12px;
		line-height: 1.6;
	}

	.dialog-actions button {
		border: 0;
		padding: 12px;
		background: transparent;
		color: var(--primary);
		font: inherit;
		font-weight: 600;
		cursor: pointer;
	}
	.draft-dialog {
		width: min(360px, calc(100vw - 48px));
		padding: 24px;
		border: 1.5px solid var(--ink);
		border-radius: 14px;
		color: var(--ink);
		background: var(--surface);
		box-shadow: 10px 10px 0 var(--shadow);
	}
	.draft-dialog::backdrop {
		background: #0006;
	}
	.draft-dialog h2 {
		font-size: 21px;
		margin-bottom: 16px;
	}
	.draft-dialog p {
		line-height: 1.6;
		color: var(--ink-3);
	}
	.dialog-actions {
		display: flex;
		justify-content: flex-end;
		margin-top: 20px;
		gap: 8px;
	}
	.dialog-actions .danger {
		color: #fff;
		background: var(--danger-fill);
		border: 1.5px solid var(--ink);
		box-shadow: 3px 3px var(--shadow);
	}
	.dialog-actions .danger:hover {
		background: color-mix(in srgb, var(--danger-fill) 92%, #000);
	}
	button:disabled {
		opacity: 0.5;
		cursor: default;
	}
	.draft-skeleton {
		padding: 16px;
		display: grid;
		gap: 24px;
	}
	.draft-skeleton > div {
		display: flex;
		gap: 16px;
		align-items: center;
	}
	.draft-skeleton span {
		width: 48px;
		height: 48px;
		background: var(--paper-2);
		border-radius: 8px;
	}
	.draft-skeleton i {
		width: 60%;
		height: 32px;
		border-radius: 6px;
		background: var(--paper-2);
	}
	@media (max-width: 420px) {
		.row-actions {
			flex-direction: column;
		}
		.icon-action {
			width: 40px;
			height: 40px;
			padding: 10px;
		}
		.draft-row {
			padding-left: 12px;
		}
		.draft-main {
			gap: 10px;
		}
	}
</style>
