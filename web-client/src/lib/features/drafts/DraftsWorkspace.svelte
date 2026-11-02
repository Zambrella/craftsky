<script lang="ts">
	import { onDestroy, onMount, untrack } from 'svelte';
	import type { CraftskyAPI, ScheduledPostSummary } from '#lib/api/client.ts';
	import { drafts, type Draft } from './store.ts';
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
	async function remove(id: string, remote: boolean) {
		if (
			busy ||
			!confirm(
				remote
					? 'Delete this scheduled post? It will not be published.'
					: 'Delete this local draft and its media from this device?'
			)
		)
			return;
		const seq = generation;
		busy = true;
		error = '';
		try {
			if (remote) await api.schedules.delete(id);
			else await drafts.remove(session.did, id);
			if (seq === generation && !destroyed) {
				notice = remote ? 'Scheduled post deleted.' : 'Draft deleted.';
				await load();
			}
		} catch (e) {
			if (seq === generation && !destroyed)
				error = e instanceof Error ? e.message : 'Could not delete this item.';
		} finally {
			if (!destroyed) busy = false;
		}
	}
	const date = (value: string) =>
		new Date(value).toLocaleString(undefined, {
			dateStyle: 'medium',
			timeStyle: 'short'
		});
</script>

<section class="draft-workspace">
	<header>
		<div>
			<p class="eyebrow">Your work in progress</p>
			<h1>{isScheduled ? 'Scheduled posts' : 'Drafts'}</h1>
		</div>
		<a class="new-post" href="/app/compose">＋ New post</a>
	</header>
	<nav aria-label="Drafts and scheduled posts">
		<a class:active={!isScheduled} href="/app/drafts">On this device</a><a
			class:active={isScheduled}
			href="/app/scheduled">Scheduled</a
		>
	</nav>
	{#if error}<div class="feedback" role="alert">
			{error}<button onclick={() => load()}>Try again</button>
		</div>{/if}{#if notice}<p role="status">{notice}</p>{/if}
	{#if loading}<p role="status" class="empty">
			Loading your {isScheduled ? 'scheduled posts' : 'drafts'}…
		</p>{:else if isScheduled}
		<p class="muted">
			Scheduled posts and their media stay private until publication.
		</p>
		{#if !scheduled.length}<div class="empty">
				<h2>Something to look forward to</h2>
				<p>Schedule a post from the editor and it will appear here.</p>
				<a href="/app/compose">Create a post</a>
			</div>{/if}
		{#each scheduled as item (item.id)}<article>
				<div class="copy">
					<span class="badge"
						>{item.kind === 'project' ? 'Project' : 'Post'} · {item.status.replaceAll(
							'_',
							' '
						)}</span
					>
					<h2>{item.projectTitle || item.textPreview || 'Untitled post'}</h2>
					<p class="muted">{date(item.scheduledAt)}</p>
					{#if item.lastErrorCode || item.subscriptionRequired}<p
							class="attention"
						>
							{item.subscriptionRequired
								? 'An active CraftSky Plus or Business subscription is required.'
								: `Needs attention: ${item.lastErrorCode?.replaceAll('_', ' ')}`}
						</p>{/if}{#if item.needsAttentionExpiresAt}<p class="muted">
							Review before {date(item.needsAttentionExpiresAt)}
						</p>{/if}
				</div>
				<div class="actions">
					<a href={`/app/compose?schedule=${encodeURIComponent(item.id)}`}
						>Edit / publish now</a
					><button disabled={busy} onclick={() => remove(item.id, true)}
						>Delete</button
					>
				</div>
			</article>{/each}
	{:else}<p class="muted">
			Drafts and media are stored privately in this browser for @{session.handle}.
			They do not sync between devices.
		</p>
		{#if !local.length}<div class="empty">
				<h2>A little space for your next idea</h2>
				<p>
					Start a post or project. Your draft and photos will be saved here.
				</p>
				<a href="/app/compose">Create a post</a>
			</div>{/if}
		{#each local as item (item.id)}<article>
				<div class="copy">
					<span class="badge"
						>{item.pending
							? 'Awaiting confirmation'
							: item.edit
								? 'Post edit'
								: item.kind === 'project'
									? 'Project'
									: 'Post'}</span
					>
					<h2>{item.text.slice(0, 140) || 'Untitled draft'}</h2>
					<p class="muted">
						{date(item.updatedAt)}{item.images.length
							? ` · ${item.images.length} images`
							: ''}{item.video ? ' · Video' : ''}
					</p>
				</div>
				<div class="actions">
					<a href={`/app/compose?draft=${encodeURIComponent(item.id)}`}
						>{item.pending ? 'Check action' : 'Continue editing'}</a
					>{#if !item.pending}<button
							disabled={busy}
							onclick={() => remove(item.id, false)}>Delete</button
						>{/if}
				</div>
			</article>{/each}{/if}
</section>

<style>
	.draft-workspace {
		max-width: 850px;
		padding: 24px;
		margin: auto;
		display: grid;
		gap: 20px;
	}
	header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 16px;
	}
	h1 {
		font-family: 'DM Serif Display', serif;
		font-size: 2.1rem;
		font-weight: 400;
	}
	.eyebrow {
		font-size: 0.8rem;
		text-transform: uppercase;
		letter-spacing: 0.09em;
		color: var(--ink-3);
	}
	nav {
		display: flex;
		gap: 12px;
		border-bottom: 1px solid var(--hairline);
	}
	nav a {
		padding: 10px 12px;
		text-decoration: none;
		color: var(--ink-3);
	}
	nav .active {
		color: var(--primary);
		border-bottom: 3px solid var(--primary);
	}
	article {
		display: flex;
		justify-content: space-between;
		gap: 18px;
		background: var(--surface);
		border: 1px solid var(--ink);
		border-radius: 14px;
		padding: 20px;
		box-shadow: 2px 2px 0 var(--shadow);
	}
	h2 {
		font-size: 1.15rem;
		white-space: pre-wrap;
	}
	.copy {
		display: grid;
		gap: 8px;
		min-width: 0;
	}
	.muted {
		color: var(--ink-3);
		font-size: 0.9rem;
	}
	.badge {
		font-size: 0.8rem;
		color: var(--primary);
	}
	.actions {
		display: flex;
		flex-direction: column;
		align-items: flex-end;
		gap: 10px;
		flex-shrink: 0;
	}
	button,
	.new-post {
		border: 1px solid var(--ink);
		padding: 8px 12px;
		background: var(--paper);
		border-radius: 8px;
		color: var(--ink);
		font: inherit;
		text-decoration: none;
		cursor: pointer;
	}
	.empty {
		padding: 50px 20px;
		text-align: center;
		display: grid;
		gap: 16px;
	}
	.feedback,
	.attention {
		padding: 12px;
		background: var(--red-soft);
		color: var(--red-ink);
		border-radius: 8px;
	}
	.feedback {
		display: flex;
		justify-content: space-between;
		align-items: center;
	}
	@media (max-width: 650px) {
		.draft-workspace {
			padding: 16px;
		}
		article {
			flex-direction: column;
		}
		.actions {
			flex-direction: row;
			align-items: center;
			justify-content: flex-end;
		}
		header {
			align-items: flex-start;
		}
		.new-post {
			white-space: nowrap;
		}
	}
</style>
