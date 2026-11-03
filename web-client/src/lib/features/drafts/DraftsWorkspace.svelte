<script lang="ts">
	import { onDestroy, onMount, untrack } from 'svelte';
	import type { CraftskyAPI, ScheduledPostSummary } from '#lib/api/client.ts';
	import { drafts, type Draft } from './store.ts';
	import { draftPreview, draftURL } from './presentation';
	import DraftThumbnail from './DraftThumbnail.svelte';
	import Icon from '../shell/Icon.svelte';
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
			if (seq === generation && !destroyed)
				error = e instanceof Error ? e.message : 'Could not load your drafts.';
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
			if (seq === generation && !destroyed)
				error = e instanceof Error ? e.message : 'Could not delete this item.';
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
	<header class="draft-appbar">
		<h1>{isScheduled ? 'Scheduled posts' : 'Drafts'}</h1>
		<button
			class="icon-action"
			aria-label="Refresh drafts"
			disabled={loading || busy}
			onclick={() => load()}><Icon name="refresh" /></button
		>
	</header>
	<nav aria-label="Drafts and scheduled posts">
		<a aria-current={!isScheduled ? 'page' : undefined} href="/app/drafts"
			>On this device</a
		>
		<a aria-current={isScheduled ? 'page' : undefined} href="/app/scheduled"
			>Scheduled</a
		>
	</nav>
	{#if error}<div class="feedback error" role="alert">
			{error}<button onclick={() => load()}>Try again</button>
		</div>{/if}
	{#if notice}<p class="feedback" role="status">{notice}</p>{/if}
	{#if loading}<div
			class="draft-skeleton"
			role="status"
			aria-label="Loading drafts"
		>
			{#each [1, 2, 3, 4] as row}<div><span></span><i></i></div>{/each}
		</div>
	{:else if isScheduled}
		{#if !scheduled.length}<div class="empty">
				<Icon name="calendar" />
				<h2>Scheduled posts</h2>
				<p>No scheduled posts yet.</p>
				<a href="/app/compose">Create a post</a>
			</div>{:else}<p class="privacy-note">
				Scheduled posts and their media stay private until publication.
			</p>{/if}
		<ul class="draft-list">
			{#each scheduled as item (item.id)}<li class="draft-row scheduled-row">
					<a
						class="draft-main"
						href={`/app/compose?schedule=${encodeURIComponent(item.id)}`}
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
							href={`/app/compose?schedule=${encodeURIComponent(item.id)}`}
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
		{#if !local.length}<div class="empty">
				<Icon name="drafts" />
				<h2>Drafts</h2>
				<p>No drafts yet. Save a post or project to come back to it later.</p>
			</div>{:else}<p class="privacy-note">
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
	oncancel={(event) => {
		if (busy) event.preventDefault();
		else deletion = undefined;
	}}
>
	<h2>{deletion?.remote ? 'Delete scheduled post?' : 'Delete draft?'}</h2>
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
		><button class="danger" disabled={busy} onclick={remove}
			>{busy ? 'Deleting…' : 'Delete'}</button
		>
	</div>
</dialog>

<style>
	.draft-workspace {
		max-width: 760px;
		margin: auto;
		min-height: 70vh;
		background: var(--paper);
	}
	.draft-appbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		min-height: 64px;
		padding: 8px 16px;
	}
	h1 {
		margin: 0;
		font-size: 22px;
		font-weight: 700;
		font-family: inherit;
	}
	nav {
		display: flex;
		border-bottom: 1px solid var(--hairline);
	}
	nav a {
		flex: 1;
		text-align: center;
		padding: 13px 12px;
		color: var(--ink-3);
		font-size: 14px;
		font-weight: 600;
		text-decoration: none;
		border-bottom: 3px solid transparent;
	}
	nav a[aria-current='page'] {
		color: var(--primary);
		border-color: var(--primary);
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
		background: var(--paper-2);
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
		background: var(--hairline);
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
	.empty {
		min-height: 55vh;
		max-width: 400px;
		padding: 48px 24px;
		margin: auto;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
		text-align: center;
	}
	.empty :global(svg) {
		width: 48px;
		height: 48px;
		color: var(--primary);
		stroke-width: 1.6;
	}
	.empty h2 {
		font-size: 22px;
		font-weight: 600;
	}
	.empty p {
		color: var(--ink-3);
		line-height: 1.6;
	}
	.empty a {
		color: var(--primary);
		padding: 8px;
	}
	.feedback {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 12px;
		padding: 14px 16px;
		margin: 12px 16px;
		border-radius: 10px;
		background: var(--primary-soft);
	}
	.feedback.error {
		background: var(--red-soft);
		color: var(--red-ink);
	}
	.feedback button,
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
		max-width: min(400px, calc(100vw - 32px));
		padding: 24px;
		border: 1px solid var(--ink);
		border-radius: 20px;
		color: var(--ink);
		background: var(--paper);
		box-shadow: 4px 4px 0 var(--ink);
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
		color: var(--red-ink);
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
