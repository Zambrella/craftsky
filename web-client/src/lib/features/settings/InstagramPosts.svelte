<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import type { CraftskyAPI } from '#lib/api/client.ts';
	import { userErrorMessage } from '#lib/user-facing-error.ts';
	import { drafts, newDraft, type Draft } from '../drafts/store';
	import { prepareImage, inspectVideo } from '../composer/media';
	import { textLength } from '../composer/model';
	import type { InstagramArchive, InstagramPost } from '../instagram-import';
	import {
		clearInstagramPostSession,
		useInstagramPostSession,
		importedPostPath,
		type InstagramPostSession,
		type ImportedInstagramPost
	} from './instagram-post-session';
	import { dateTime } from './model';

	let {
		api,
		owner,
		sessionId
	}: { api: CraftskyAPI; owner: string; sessionId: string } = $props();
	let cache: InstagramPostSession | undefined;
	let archive = $state<InstagramArchive>();
	let fileName = $state('');
	let visible = $state(20);
	let posted = $state<Record<string, ImportedInstagramPost>>({});
	let previews = $state<Record<string, string>>({});
	let busy = $state(false);
	let checking = $state(false);
	let error = $state('');
	let selected = $state<InstagramPost>();
	let selectedKind = $state<Draft['kind']>('standard');
	let mediaKind = $state<'images' | 'video'>('images');
	let imageIndices = $state<number[]>([]);
	let videoIndex = $state(0);
	let active = true;
	let controller = new AbortController();
	let fileInput: HTMLInputElement;
	const shown = $derived(archive?.posts.slice(0, visible) ?? []);

	onMount(() => {
		cache = useInstagramPostSession(`${sessionId}:${owner}`);
		archive = cache.archive;
		fileName = cache.fileName;
		visible = cache.visible;
		posted = { ...cache.posted };
		previews = { ...cache.previews };
		void refresh();
	});
	onDestroy(() => {
		active = false;
		controller.abort();
	});

	async function lookup(posts: readonly InstagramPost[]) {
		for (let start = 0; start < posts.length; start += 100) {
			const result = await api.transport.request<{
				items: ImportedInstagramPost[];
			}>('POST', '/v1/migrations/instagram/post-imports/lookup', {
				body: {
					sourceKeys: posts
						.slice(start, start + 100)
						.map((post) => post.sourceKey)
				},
				signal: controller.signal
			});
			if (!active || !cache) return;
			for (const item of result.items) cache.posted[item.sourceKey] = item;
			posted = { ...cache.posted };
		}
	}
	async function thumbnails() {
		const current = archive;
		if (!current || !cache) return;
		for (const post of shown) {
			if (!active || current !== archive) return;
			if (!post.imageCount || cache.previews[post.sourceKey]) continue;
			try {
				const blob = await current.preview(post, controller.signal);
				if (!active || current !== archive || !blob) continue;
				cache.previews[post.sourceKey] = URL.createObjectURL(blob);
				previews = { ...cache.previews };
			} catch {
				// A missing thumbnail does not discard a recoverable source post.
			}
		}
	}
	async function refresh() {
		if (!archive || checking) return;
		checking = true;
		error = '';
		try {
			await lookup(shown);
			await thumbnails();
		} catch (cause) {
			if (active) error = userErrorMessage(cause);
		} finally {
			if (active) checking = false;
		}
	}
	function clear() {
		controller.abort();
		controller = new AbortController();
		clearInstagramPostSession();
		cache = useInstagramPostSession(`${sessionId}:${owner}`);
		archive = undefined;
		fileName = '';
		posted = {};
		previews = {};
		selected = undefined;
		visible = 20;
		error = '';
		if (fileInput) fileInput.value = '';
	}
	async function readFile(event: Event) {
		const file = (event.currentTarget as HTMLInputElement).files?.[0];
		if (!file) return;
		clear();
		busy = true;
		try {
			const { openInstagramArchive } = await import('../instagram-import');
			// The cached worker survives navigation into the composer. Per-view requests
			// use controller.signal; explicit clearing/account cleanup closes the worker.
			const opened = await openInstagramArchive(file);
			if (!active || !cache) {
				opened.close();
				return;
			}
			cache.archive = opened;
			cache.fileName = file.name;
			archive = opened;
			fileName = file.name;
			await refresh();
		} catch (cause) {
			if (active) error = userErrorMessage(cause);
		} finally {
			if (active) busy = false;
		}
	}
	function choose(post: InstagramPost, kind: Draft['kind']) {
		if (busy || checking) return;
		selected = post;
		selectedKind = kind;
		mediaKind = post.imageCount ? 'images' : 'video';
		imageIndices = Array.from(
			{ length: Math.min(post.imageCount, 8) },
			(_, index) => index
		);
		videoIndex = 0;
		if (
			!(post.imageCount && post.videoCount) &&
			post.imageCount <= 8 &&
			post.videoCount <= 1
		)
			void openComposer(post, kind);
	}
	function toggleImage(index: number, checked: boolean) {
		imageIndices = checked
			? [...imageIndices, index].sort((left, right) => left - right)
			: imageIndices.filter((value) => value !== index);
	}
	async function openComposer(post: InstagramPost, kind: Draft['kind']) {
		if (!archive || busy || checking) return;
		busy = true;
		error = '';
		try {
			await lookup([post]);
			if (!active || posted[post.sourceKey]) return;
			const existing = (await drafts.list(owner)).find(
				(draft) => draft.instagramImport?.sourceKey === post.sourceKey
			);
			if (!active) return;
			let draft = existing;
			if (draft && !draft.pending && draft.kind !== kind) {
				draft.kind = kind;
				await drafts.save(owner, draft);
			}
			if (!draft) {
				const files = await archive.media(post, controller.signal, {
					kind: mediaKind,
					indices: mediaKind === 'images' ? [...imageIndices] : [videoIndex]
				});
				if (!active) return;
				draft = newDraft(owner, kind);
				draft.text = post.caption;
				draft.instagramImport = {
					sourceKey: post.sourceKey,
					createdAt: post.createdAt
				};
				for (const file of files.images)
					draft.images.push(await prepareImage(file, file.name));
				if (files.videos.length)
					draft.video = await inspectVideo(files.videos[0]);
				if (!active) return;
				await drafts.save(owner, draft);
			}
			if (!active) return;
			await goto(
				`/compose${draft.kind === 'project' ? '/project' : ''}?draft=${encodeURIComponent(draft.id)}`
			);
		} catch (cause) {
			if (active) error = userErrorMessage(cause);
		} finally {
			if (active) busy = false;
		}
	}
	async function more() {
		visible += 20;
		if (cache) cache.visible = visible;
		await refresh();
	}
</script>

<section
	class="panel stack instagram-posts"
	aria-labelledby="instagram-posts-heading"
>
	<h2 id="instagram-posts-heading" class="section-heading">
		Import your posts and projects
	</h2>
	<p class="helper">
		Choose your Instagram posts export in ZIP format. The archive is processed
		on this device. Only selected text and media, the original post date, and
		minimal duplicate-tracking metadata leave your device. Each post keeps its
		original Instagram date and appears on your profile and in search.
	</p>
	<p class="helper">
		Select Posts and Reels, JSON format, and All time when requesting your
		Instagram export. Keep the downloaded ZIP intact.
	</p>
	<label
		>Choose your Instagram posts ZIP export<input
			bind:this={fileInput}
			type="file"
			accept="application/zip,.zip"
			disabled={busy || checking}
			onchange={readFile}
		/></label
	>
	{#if error}<p class="notice" role="alert">
			{error}
			<button disabled={busy || checking} onclick={refresh}
				>Check imports again</button
			>
		</p>{/if}
	{#if busy}<p role="status">Preparing your post…</p>{:else if checking}<p
			role="status"
		>
			Checking previous imports…
		</p>{/if}
	{#if archive}
		<div class="actions">
			<p class="helper">
				{fileName} · {archive.posts.length} posts{archive.skipped
					? ` · ${archive.skipped} unsupported or invalid entries skipped`
					: ''}
			</p>
			<button disabled={busy || checking} onclick={clear}>Clear export</button>
		</div>
		<p class="helper">
			Your export stays available while you edit posts. After reloading this
			page, select the ZIP again; previous imports will still be recognised.
			Edits you save as drafts stay in this browser.
		</p>
		{#each shown as post (post.sourceKey)}
			{@const imported = posted[post.sourceKey]}
			{@const destination = imported
				? importedPostPath(imported.uri)
				: undefined}
			<article class="post-import-row">
				<div class="post-overview">
					{#if previews[post.sourceKey]}<img
							class="post-thumbnail"
							src={previews[post.sourceKey]}
							alt=""
						/>{:else}<div class="post-thumbnail placeholder" aria-hidden="true">
							{post.videoCount && !post.imageCount ? '▶' : '▧'}
						</div>{/if}
					<div class="post-copy">
						<h3>{dateTime(post.createdAt)}</h3>
						<p class="caption">{post.caption || 'No caption'}</p>
						<p class="helper">
							{post.imageCount}
							{post.imageCount === 1 ? 'photo' : 'photos'} · {post.videoCount}
							{post.videoCount === 1 ? 'video' : 'videos'} · {textLength(
								post.caption
							)} characters
						</p>
					</div>
				</div>
				{#each post.warnings as warning}<p class="helper warning">
						{warning}
					</p>{/each}
				{#if textLength(post.caption) > 2000}<p class="helper warning">
						This caption exceeds 2,000 characters. Shorten it in the composer
						before publishing.
					</p>{/if}
				{#if imported}<p class="notice">
						{imported.kind === 'project'
							? 'Project created'
							: 'Posted'}{#if destination}
							· <a href={destination}
								>View {imported.kind === 'project' ? 'project' : 'post'}</a
							>{/if}
					</p>
				{:else}<div class="actions">
						<button
							disabled={busy || checking}
							onclick={() => choose(post, 'standard')}>Create post</button
						><button
							disabled={busy || checking}
							onclick={() => choose(post, 'project')}>Create project</button
						>
					</div>{/if}
				{#if selected?.sourceKey === post.sourceKey && !imported && ((post.imageCount && post.videoCount) || post.imageCount > 8 || post.videoCount > 1)}
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
									onclick={() => (mediaKind = 'images')}>Photos</button
								><button
									aria-pressed={mediaKind === 'video'}
									disabled={busy}
									onclick={() => (mediaKind = 'video')}>One video</button
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
													imageIndices.length >= 8}
												onchange={(event) =>
													toggleImage(index, event.currentTarget.checked)}
											/>Photo {index + 1}</label
										>{/each}
								</div>
							</fieldset>
						{:else}<label
								>Video<select bind:value={videoIndex} disabled={busy}
									>{#each Array(post.videoCount) as _, index}<option
											value={index}>Video {index + 1}</option
										>{/each}</select
								></label
							>{/if}
						<button
							class="primary"
							disabled={busy ||
								checking ||
								(mediaKind === 'images' && !imageIndices.length)}
							onclick={() => openComposer(post, selectedKind)}
							>Continue to {selectedKind === 'project' ? 'project' : 'post'} composer</button
						>
					</div>
				{/if}
			</article>
		{:else}<p class="empty">
				No supported posts were found. Choose an Instagram JSON export that
				includes your posts and their media.
			</p>{/each}
		{#if visible < archive.posts.length}<button
				disabled={busy || checking}
				onclick={more}>Show more posts</button
			>{/if}
	{/if}
</section>

<style>
	.post-import-row {
		padding: 1.2rem 0;
		border-top: 1px solid var(--line, #ccc);
	}
	.post-overview {
		display: flex;
		gap: 1rem;
		align-items: flex-start;
	}
	.post-thumbnail {
		width: 88px;
		height: 88px;
		object-fit: cover;
		border-radius: 8px;
		flex-shrink: 0;
		background: var(--paper);
	}
	.placeholder {
		display: grid;
		place-items: center;
		font-size: 2rem;
	}
	.post-copy {
		min-width: 0;
		flex: 1;
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
	@media (max-width: 480px) {
		.post-thumbnail {
			width: 64px;
			height: 64px;
		}
		.post-overview {
			gap: 0.75rem;
		}
	}
</style>
