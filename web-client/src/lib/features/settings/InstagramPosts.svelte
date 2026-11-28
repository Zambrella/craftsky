<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import type {
		CraftskyAPI,
		InstagramPublishedImport
	} from '#lib/api/client.ts';
	import { userErrorMessage } from '#lib/user-facing-error.ts';
	import { drafts, newDraft, type Draft } from '../drafts/store';
	import { prepareImage, inspectVideo } from '../composer/media';
	import { postLimits } from '../composer/limits';
	import { graphemeCount } from '#lib/text.ts';
	import type { InstagramArchive, InstagramPost } from '../instagram-import';
	import {
		clearInstagramPostSession,
		useInstagramPostSession,
		importedPostPath,
		type InstagramPostSession
	} from './instagram-post-session';
	import { dateTime } from './model';
	import InstagramExportPicker from './InstagramExportPicker.svelte';
	import SettingsIcon from './SettingsIcon.svelte';
	const importDate = new Intl.DateTimeFormat(undefined, {
		weekday: 'short',
		month: 'short',
		day: 'numeric'
	});

	let {
		api,
		owner,
		sessionId
	}: { api: CraftskyAPI; owner: string; sessionId: string } = $props();
	let cache: InstagramPostSession | undefined;
	let archive = $state<InstagramArchive>();
	let fileName = $state('');
	let visible = $state(20);
	let posted = $state<Record<string, InstagramPublishedImport>>({});
	let previews = $state<Record<string, string>>({});
	let busy = $state(false);
	let checking = $state(false);
	let error = $state('');
	let selected = $state<InstagramPost>();
	let selectedKind = $state<Draft['kind']>('standard');
	let mediaKind = $state<'images' | 'video'>('images');
	let imageIndices = $state<number[]>([]);
	let videoIndex = $state(0);
	let opening = $state<{ sourceKey: string; kind: Draft['kind'] }>();
	let composerFailure = $state.raw<{
		sourceKey: string;
		message: string;
		recovery?: Draft;
	}>();
	let active = true;
	let controller = new AbortController();
	let thumbnailController: AbortController | undefined;
	let lookupSequence = 0;
	const confirmedLookups = new Map<string, number>();
	const attemptedPreviews = new Set<string>();
	const shown = $derived(archive?.posts.slice(0, visible) ?? []);

	onMount(() => {
		cache = useInstagramPostSession(`${sessionId}:${owner}`);
		archive = cache.archive;
		fileName = cache.fileName;
		visible = cache.visible;
		posted = { ...cache.posted };
		previews = { ...cache.previews };
		void refresh();
		const refreshOnReturn = () => {
			if (!document.hidden && !busy) void refresh();
		};
		window.addEventListener('focus', refreshOnReturn);
		document.addEventListener('visibilitychange', refreshOnReturn);
		return () => {
			window.removeEventListener('focus', refreshOnReturn);
			document.removeEventListener('visibilitychange', refreshOnReturn);
		};
	});
	onDestroy(() => {
		active = false;
		stopThumbnails();
		controller.abort();
	});

	async function lookup(posts: readonly InstagramPost[]) {
		const session = cache;
		if (!session) return;
		const signal = AbortSignal.any([controller.signal, session.signal]);
		const sequence = ++lookupSequence;
		for (let start = 0; start < posts.length; start += 100) {
			const sourceKeys = posts
				.slice(start, start + 100)
				.map((post) => post.sourceKey);
			const result = await api.instagram.lookupPostImports(sourceKeys, signal);
			if (!active || signal.aborted || cache !== session) return;
			const matches = new Map(
				result.items.map((item) => [item.sourceKey, item])
			);
			for (const key of sourceKeys) {
				// The check before opening a composer can finish before a list refresh.
				if ((confirmedLookups.get(key) ?? 0) > sequence) continue;
				confirmedLookups.set(key, sequence);
				const match = matches.get(key);
				if (match) session.posted[key] = match;
				else delete session.posted[key];
			}
			posted = { ...session.posted };
		}
	}
	function stopThumbnails() {
		thumbnailController?.abort();
		thumbnailController = undefined;
	}
	async function thumbnails() {
		const current = archive;
		const session = cache;
		if (!active || !current || !session || thumbnailController) return;
		const task = new AbortController();
		const signal = AbortSignal.any([task.signal, session.signal]);
		thumbnailController = task;
		try {
			for (
				let index = 0;
				index < Math.min(visible, current.posts.length);
				index++
			) {
				const post = current.posts[index];
				if (
					(!post.imageCount && !post.videoCount) ||
					session.previews[post.sourceKey] ||
					attemptedPreviews.has(post.sourceKey)
				)
					continue;
				let blob: Blob | undefined;
				try {
					blob = await current.preview(post, signal);
				} catch {
					// Unsupported previews should not prevent opening a composer.
				}
				if (signal.aborted) return;
				attemptedPreviews.add(post.sourceKey);
				if (!blob) continue;
				session.previews[post.sourceKey] = URL.createObjectURL(blob);
				previews = { ...session.previews };
			}
		} finally {
			if (thumbnailController === task) thumbnailController = undefined;
		}
	}
	async function refresh() {
		if (!archive || checking) return;
		void thumbnails();
		checking = true;
		error = '';
		try {
			await lookup(shown);
		} catch (cause) {
			if (active) error = userErrorMessage(cause);
		} finally {
			if (active) checking = false;
		}
	}
	function clear() {
		stopThumbnails();
		attemptedPreviews.clear();
		confirmedLookups.clear();
		controller.abort();
		controller = new AbortController();
		clearInstagramPostSession();
		cache = useInstagramPostSession(`${sessionId}:${owner}`);
		archive = undefined;
		fileName = '';
		posted = {};
		previews = {};
		selected = undefined;
		composerFailure = undefined;
		visible = 20;
		error = '';
	}
	async function readFile(file: File) {
		clear();
		const session = cache;
		if (!session) return;
		const signal = AbortSignal.any([controller.signal, session.signal]);
		busy = true;
		try {
			const { openInstagramArchive } = await import('../instagram-import');
			// The cached worker survives navigation into the composer. Per-view requests
			// use controller.signal; explicit clearing/account cleanup closes the worker.
			const opened = await openInstagramArchive(file, signal);
			if (signal.aborted) {
				opened.close();
				return;
			}
			session.archive = opened;
			session.fileName = file.name;
			archive = opened;
			fileName = file.name;
			void refresh();
		} catch (cause) {
			if (!signal.aborted) error = userErrorMessage(cause);
		} finally {
			if (!signal.aborted) busy = false;
		}
	}
	function choose(post: InstagramPost, kind: Draft['kind']) {
		if (busy) return;
		composerFailure = undefined;
		selected = post;
		selectedKind = kind;
		mediaKind = post.videoCount && !post.imageCount ? 'video' : 'images';
		imageIndices = Array.from(
			{ length: Math.min(post.imageCount, postLimits.imageCount) },
			(_, index) => index
		);
		videoIndex = 0;
		if (
			!(post.imageCount && post.videoCount) &&
			post.imageCount <= postLimits.imageCount &&
			post.videoCount <= 1
		)
			void openComposer(post, kind);
	}
	function toggleImage(index: number, checked: boolean) {
		composerFailure = undefined;
		imageIndices = checked
			? [...imageIndices, index].sort((left, right) => left - right)
			: imageIndices.filter((value) => value !== index);
	}
	async function prepareDraft(
		post: InstagramPost,
		kind: Draft['kind'],
		source: InstagramArchive,
		signal: AbortSignal
	) {
		const draft = newDraft(owner, kind);
		draft.text = post.caption;
		draft.instagramImport = {
			sourceKey: post.sourceKey,
			createdAt: post.createdAt
		};
		const type = mediaKind;
		const indices = type === 'images' ? [...imageIndices] : [videoIndex];
		let mediaError: string | undefined;
		if (!post.imageCount && !post.videoCount) {
			mediaError = 'This post contains media CraftSky cannot import.';
		}
		for (const index of indices) {
			signal.throwIfAborted();
			try {
				const files = await source.media(post, signal, {
					kind: type,
					indices: [index]
				});
				signal.throwIfAborted();
				for (const file of files.images)
					draft.images.push(await prepareImage(file, file.name));
				if (files.videos.length)
					draft.video = await inspectVideo(files.videos[0], signal);
			} catch (cause) {
				signal.throwIfAborted();
				mediaError ??= userErrorMessage(cause);
			}
		}
		signal.throwIfAborted();
		if (mediaError) {
			composerFailure = {
				sourceKey: post.sourceKey,
				message: mediaError,
				recovery: draft
			};
			return;
		}
		return draft;
	}
	function recoveryAction(draft: Draft) {
		if (draft.video) return 'Continue with video';
		const count = draft.images.length;
		if (!count) return 'Continue without media';
		return `Continue with ${count} ${count === 1 ? 'photo' : 'photos'}`;
	}
	async function openComposer(
		post: InstagramPost,
		kind: Draft['kind'],
		recovery?: Draft
	) {
		const source = archive;
		const session = cache;
		if (!source || !session || busy) return;
		const signal = AbortSignal.any([controller.signal, session.signal]);
		busy = true;
		opening = { sourceKey: post.sourceKey, kind };
		composerFailure = undefined;
		try {
			await lookup([post]);
			if (signal.aborted || posted[post.sourceKey]) return;
			let draft = await drafts.findInstagramImport(owner, post.sourceKey);
			if (signal.aborted || posted[post.sourceKey]) return;
			if (draft && !draft.pending && draft.kind !== kind) {
				draft.kind = kind;
				await drafts.save(owner, draft);
			}
			if (!draft) {
				stopThumbnails();
				draft = recovery ?? (await prepareDraft(post, kind, source, signal));
				if (!draft || signal.aborted || posted[post.sourceKey]) return;
				await drafts.save(owner, draft);
			}
			if (signal.aborted || posted[post.sourceKey]) return;
			await goto(
				`/compose${draft.kind === 'project' ? '/project' : ''}?draft=${encodeURIComponent(draft.id)}`
			);
		} catch (cause) {
			if (!signal.aborted) {
				composerFailure = {
					sourceKey: post.sourceKey,
					message: userErrorMessage(cause),
					recovery
				};
			}
		} finally {
			if (!signal.aborted) {
				opening = undefined;
				busy = false;
				void thumbnails();
			}
		}
	}
	async function more() {
		visible += 20;
		if (cache) cache.visible = visible;
		await refresh();
	}
</script>

<section
	class="panel stack instagram-export"
	aria-label="Instagram posts export"
>
	{#if archive}
		<div class="export-summary">
			<h2 class="export-filename">{fileName}</h2>
			<p>
				{archive.posts.length}
				{archive.posts.length === 1 ? 'post' : 'posts'}
			</p>
			{#if archive.skipped}
				<p>
					{archive.skipped} unsupported or invalid {archive.skipped === 1
						? 'entry'
						: 'entries'} skipped
				</p>
			{/if}
			<button
				type="button"
				class="import-text-action"
				disabled={busy || checking}
				onclick={clear}>Clear export</button
			>
			{#if !archive.posts.length}<p>
					No supported posts were found in this export.
				</p>{/if}
		</div>
	{:else}
		<p class="helper">
			In your Instagram export, select Posts and Reels, JSON format, and All
			time. Then choose the downloaded ZIP.
		</p>
		<InstagramExportPicker
			accept="application/zip,.zip"
			disabled={busy || checking}
			onSelect={readFile}
		/>
		<p class="helper">
			Your archive stays on this device. CraftSky checks a reference for each
			post to identify duplicates. Only the posts you publish share their
			caption, selected media and original date.
		</p>
	{/if}
	{#if error}<p class="notice" role="alert">
			{error}
			{#if archive}<button disabled={busy || checking} onclick={refresh}
					>Check imports again</button
				>{/if}
		</p>{/if}
	{#if checking}<p role="status">
			Checking previous imports…
		</p>{:else if busy && !archive}<p role="status">
			Reading your export…
		</p>{/if}
</section>

{#if archive && shown.length}
	<section aria-label="Posts in your export">
		{#each shown as post (post.sourceKey)}
			{@const imported = posted[post.sourceKey]}
			{@const failure =
				composerFailure?.sourceKey === post.sourceKey
					? composerFailure
					: undefined}
			{@const recovery = failure?.recovery}
			{@const destination = imported
				? importedPostPath(imported.uri)
				: undefined}
			<article class="panel post-import-row">
				<div class="post-overview">
					{#if previews[post.sourceKey]}<img
							class="post-thumbnail"
							src={previews[post.sourceKey]}
							alt=""
						/>{:else}<div class="post-thumbnail placeholder" aria-hidden="true">
							<SettingsIcon
								name={post.videoCount && !post.imageCount ? 'video' : 'image'}
							/>
						</div>{/if}
					<div class="post-copy">
						<h3>
							<time datetime={post.createdAt} title={dateTime(post.createdAt)}
								>{importDate.format(new Date(post.createdAt))}</time
							>
						</h3>
						<p class="caption">{post.caption || 'No caption'}</p>
						<p class="helper">
							{post.imageCount}
							{post.imageCount === 1 ? 'photo' : 'photos'} · {post.videoCount}
							{post.videoCount === 1 ? 'video' : 'videos'} · {graphemeCount(
								post.caption
							)} characters
						</p>
					</div>
				</div>
				{#each post.warnings as warning}<p class="helper warning">
						{warning}
					</p>{/each}
				{#if imported}<div class="actions row-actions published-actions">
						<strong
							>{imported.kind === 'project'
								? 'Project created'
								: 'Posted'}</strong
						>{#if destination}
							<a class="action import-text-action" href={destination}
								>View {imported.kind === 'project' ? 'project' : 'post'}</a
							>{/if}
					</div>
				{:else}<div class="actions row-actions">
						<button
							type="button"
							class="import-text-action"
							disabled={busy}
							onclick={() => choose(post, 'standard')}
							><SettingsIcon name="textPost" size={18} bold />Create post</button
						><button
							type="button"
							class="import-text-action"
							disabled={busy}
							onclick={() => choose(post, 'project')}
							><SettingsIcon name="projectPost" size={18} bold />Create project</button
						>
					</div>{/if}
				{#if !imported && opening?.sourceKey === post.sourceKey}
					<p class="helper" role="status">
						Opening the {opening.kind === 'project' ? 'project' : 'post'} composer…
					</p>
				{/if}
				{#if selected?.sourceKey === post.sourceKey && !imported && ((post.imageCount && post.videoCount) || post.imageCount > postLimits.imageCount || post.videoCount > 1)}
					<div class="media-selection stack">
						<h4>
							Choose media for this {selectedKind === 'project'
								? 'project'
								: 'post'}
						</h4>
						<p class="helper">
							A post can contain up to eight photos or one video. Mixed
							carousels are not supported. Only your selection will be added.
						</p>
						{#if post.imageCount && post.videoCount}<div
								class="segmented"
								aria-label="Media type"
							>
								<button
									aria-pressed={mediaKind === 'images'}
									disabled={busy}
									onclick={() => {
										mediaKind = 'images';
										composerFailure = undefined;
									}}>Photos</button
								><button
									aria-pressed={mediaKind === 'video'}
									disabled={busy}
									onclick={() => {
										mediaKind = 'video';
										composerFailure = undefined;
									}}>One video</button
								>
							</div>{/if}
						{#if mediaKind === 'images'}<fieldset disabled={busy}>
								<legend
									>Select up to eight photos ({imageIndices.length} selected)</legend
								>
								<div class="media-options">
									{#each Array(post.imageCount) as _, index}<label
											><input
												type="checkbox"
												checked={imageIndices.includes(index)}
												disabled={!imageIndices.includes(index) &&
													imageIndices.length >= postLimits.imageCount}
												onchange={(event) =>
													toggleImage(index, event.currentTarget.checked)}
											/>Photo {index + 1}</label
										>{/each}
								</div>
							</fieldset>
						{:else}<label
								>Video<select
									bind:value={videoIndex}
									disabled={busy}
									onchange={() => (composerFailure = undefined)}
									>{#each Array(post.videoCount) as _, index}<option
											value={index}>Video {index + 1}</option
										>{/each}</select
								></label
							>{/if}
						<button
							class="primary"
							disabled={busy ||
								(mediaKind === 'images' && !imageIndices.length)}
							onclick={() => openComposer(post, selectedKind)}
							>Continue to {selectedKind === 'project' ? 'project' : 'post'} composer</button
						>
					</div>
				{/if}
				{#if !imported && failure}
					<p class="notice" role="alert">{failure.message}</p>
					{#if recovery}
						<button
							disabled={busy}
							onclick={() => openComposer(post, recovery.kind, recovery)}
							>{recoveryAction(recovery)}</button
						>
					{/if}
				{/if}
			</article>
		{/each}
		{#if visible < archive.posts.length}<button
				disabled={busy || checking}
				onclick={more}>Show more posts</button
			>{/if}
	</section>
{/if}

<style>
	.export-summary {
		display: grid;
		justify-items: start;
		gap: 12px;
	}
	.export-summary .export-filename {
		display: -webkit-box;
		-webkit-box-orient: vertical;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		overflow: hidden;
		overflow-wrap: anywhere;
		margin: 0;
		font-size: 16px;
		line-height: 1.3;
	}
	.export-summary p {
		margin: 0;
		font-size: 14px;
	}
	.post-overview {
		display: flex;
		gap: 12px;
		align-items: flex-start;
	}
	.post-thumbnail {
		width: 72px;
		height: 72px;
		object-fit: cover;
		border-radius: 6px;
		flex-shrink: 0;
		background: var(--paper);
	}
	.placeholder {
		display: grid;
		place-items: center;
		color: var(--primary);
	}
	.row-actions {
		margin-top: 12px;
	}
	.actions.published-actions {
		gap: 8px;
	}
	.published-actions strong {
		font-size: 15px;
		font-weight: 600;
	}
	:is(button, a).import-text-action {
		padding: 0 12px;
		border: 0;
		border-radius: 999px;
		background: transparent;
		color: var(--primary);
		box-shadow: none;
		text-decoration: none;
	}
	.import-text-action :global(svg) {
		flex: none;
	}
	:is(button, a).import-text-action:not(:disabled):hover {
		background: color-mix(in srgb, currentColor 8%, transparent);
	}
	:is(button, a).import-text-action:not(:disabled):active {
		background: color-mix(in srgb, currentColor 12%, transparent);
	}
	.post-copy {
		min-width: 0;
		flex: 1;
		display: grid;
		gap: 4px;
	}
	.post-copy :is(h3, p) {
		margin: 0;
		font-size: 14px;
	}
	.post-copy .helper {
		font-size: 13px;
	}
	.caption {
		display: -webkit-box;
		-webkit-line-clamp: 3;
		line-clamp: 3;
		-webkit-box-orient: vertical;
		overflow: hidden;
		overflow-wrap: anywhere;
		white-space: pre-wrap;
	}
	.media-selection {
		margin-top: 1rem;
		padding: 1rem;
		background: var(--paper);
		border-radius: 8px;
	}
	.media-options {
		display: flex;
		flex-wrap: wrap;
		gap: 0.75rem 1rem;
	}
	.media-options label {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.media-options input {
		width: auto;
	}
	.warning {
		margin: 0.5rem 0;
	}
</style>
