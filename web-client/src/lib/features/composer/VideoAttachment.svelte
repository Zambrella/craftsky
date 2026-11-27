<script lang="ts">
	import { tick } from 'svelte';
	import { pushState } from '$app/navigation';
	import { page } from '$app/state';
	import type { DraftMedia } from '../drafts/store.ts';
	import { mountComposerPage } from './page-layout';
	import ComposerDetailAction from './ComposerDetailAction.svelte';
	import { prepareImage } from './media';
	import {
		captureVideoThumbnailFrames,
		videoThumbnailPercent,
		videoThumbnailPercents,
		type VideoThumbnailPercent
	} from './video-thumbnail-frames';
	import { userErrorMessage } from '#lib/user-facing-error.ts';
	import Icon from '../shell/Icon.svelte';
	let {
		media,
		disabled = false,
		autoOpen = false,
		customThumbnail = false,
		onthumbnail,
		onframe,
		onalt,
		onremove
	}: {
		media: DraftMedia;
		disabled?: boolean;
		autoOpen?: boolean;
		customThumbnail?: boolean;
		onthumbnail: (image: Blob | undefined) => void;
		onframe: (percent: VideoThumbnailPercent) => void;
		onalt: (value: string) => void;
		onremove: () => void;
	} = $props();
	const uid = $props.id();
	let open = $state(false);
	let url = $state('');
	let thumbnailURL = $state('');
	let frameURLs = $state<Partial<Record<VideoThumbnailPercent, string>>>({});
	const selectedPercent = $derived(
		videoThumbnailPercent(media.thumbnailFramePercent)
	);
	const videoRatio = $derived(
		media.width && media.height ? `${media.width} / ${media.height}` : undefined
	);
	let thumbnailInput = $state<HTMLInputElement>();
	let thumbnailError = $state('');
	let preparingThumbnail = $state(false);
	$effect(() => {
		const source = media.thumbnail;
		const objectURL = source ? URL.createObjectURL(source) : '';
		thumbnailURL = objectURL;
		return () => {
			if (objectURL) URL.revokeObjectURL(objectURL);
		};
	});
	$effect(() => {
		const source = media.blob;
		if (!open || !customThumbnail || !source || media.previewUnavailable)
			return;
		const capture = new AbortController();
		const signal = AbortSignal.any([
			capture.signal,
			AbortSignal.timeout(10_000)
		]);
		const urls: Partial<Record<VideoThumbnailPercent, string>> = {};
		void captureVideoThumbnailFrames(
			source,
			media.durationMs,
			signal,
			(percent, image) => {
				const url = URL.createObjectURL(image);
				urls[percent] = url;
				frameURLs[percent] = url;
			}
		).catch(() => {});
		return () => {
			capture.abort();
			for (const url of Object.values(urls)) URL.revokeObjectURL(url);
			frameURLs = {};
		};
	});
	async function selectThumbnail(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		input.value = '';
		if (!file || preparingThumbnail || disabled) return;
		const id = media.id;
		preparingThumbnail = true;
		thumbnailError = '';
		try {
			const image = await prepareImage(
				file,
				file.name,
				{},
				{
					maxDimension: 1280,
					maxBytes: 1_000_000
				}
			);
			if (open && media.id === id && !disabled) onthumbnail(image.blob);
		} catch (error) {
			thumbnailError = userErrorMessage(error);
		} finally {
			preparingThumbnail = false;
		}
	}
	let button: HTMLButtonElement;
	let previous: HTMLElement | undefined;
	async function show() {
		if (disabled) return;
		previous =
			document.activeElement instanceof HTMLElement
				? document.activeElement
				: undefined;
		await pushState('', { ...page.state, videoDetails: uid });
	}
	function close() {
		if (!disabled) history.back();
	}
	$effect(() => {
		const next = page.state.videoDetails === uid;
		if (open === next) return;
		open = next;
		if (!next)
			void tick().then(() => {
				(previous?.isConnected ? previous : button)?.focus({
					preventScroll: true
				});
			});
	});
	$effect(() => {
		const blob = media.blob;
		const source = blob ? URL.createObjectURL(blob) : media.remoteUrl || '';
		url = source;
		return () => {
			if (blob) URL.revokeObjectURL(source);
		};
	});
	let requestedFor = '';
	$effect(() => {
		if (autoOpen && !disabled && requestedFor !== media.id) {
			requestedFor = media.id;
			void tick().then(show);
		}
	});
	function playback(node: HTMLVideoElement) {
		node.muted = false;
		node.volume = 1;
		return {
			destroy() {
				node.pause();
				node.removeAttribute('src');
				node.load();
			}
		};
	}
</script>

<div class="video-attachment">
	<div class="video-summary">
		<span class="video-icon"><Icon name="video" size={28} /></span>
		<span class="video-title"
			><strong>Selected video</strong><small>{media.name}</small></span
		>
	</div>
	<button
		bind:this={button}
		class="video-details-button"
		type="button"
		{disabled}
		onclick={show}
		aria-label="Inspect selected video"
		><Icon name="video" />Video details</button
	>
	<div class="video-actions">
		<button type="button" class="danger" {disabled} onclick={onremove}
			><Icon name="trash" />Remove video</button
		>
	</div>
</div>
<svelte:window
	onkeydown={(event) => {
		if (
			open &&
			event.key === 'Escape' &&
			!event.defaultPrevented &&
			!document.querySelector('dialog[open]')
		) {
			event.preventDefault();
			close();
		}
	}}
/>
{#if open}<section
		class="video-details-page"
		aria-label="Video details"
		use:mountComposerPage
	>
		<header class="page-appbar">
			<button type="button" aria-label="Back to post" {disabled} onclick={close}
				><Icon name="back" /></button
			>
			<h2>Video details</h2>
		</header>
		<div class="video-details-scroll">
			<h3 class="detail-heading video-preview-heading">Video</h3>
			{#if media.previewUnavailable}<p class="muted">
					Preview unavailable in this browser. You can still upload this video.
				</p>{:else}<video
					use:playback
					controls
					playsinline
					src={url}
					aria-label="Local video preview"><track kind="captions" /></video
				>{/if}
			{#if customThumbnail || thumbnailURL}
				<div class="thumbnail-controls">
					<h3 class="detail-heading">Thumbnail</h3>
					{#if thumbnailURL}
						<img
							class="thumbnail-preview"
							src={thumbnailURL}
							alt="Video thumbnail preview"
							style:aspect-ratio={videoRatio}
							style:width={media.width && media.height
								? `min(100%, ${Math.round((240 * media.width) / media.height)}px)`
								: undefined}
						/>
					{:else if media.blob && customThumbnail && ((media.durationMs ?? 0) > 0 || frameURLs[0])}
						<div
							class="thumbnail-grid"
							role="group"
							aria-label="Automatic thumbnail frames"
						>
							{#each videoThumbnailPercents as percent (percent)}
								<button
									type="button"
									aria-label={`Frame at ${percent}%`}
									aria-pressed={selectedPercent === percent}
									{disabled}
									onclick={() => onframe(percent)}
								>
									<span class="thumbnail-frame" style:aspect-ratio={videoRatio}>
										{#if frameURLs[percent]}
											<img src={frameURLs[percent]} alt="" />
										{:else}
											<Icon name="video" />
										{/if}
									</span>
									{#if selectedPercent === percent}
										<span class="thumbnail-selected"
											><Icon name="check" size={16} /></span
										>
									{/if}
								</button>
							{/each}
						</div>
					{/if}
					{#if thumbnailURL}
						<button
							type="button"
							class="thumbnail-remove"
							{disabled}
							onclick={() => {
								thumbnailError = '';
								onthumbnail(undefined);
							}}>Remove custom thumbnail</button
						>
					{:else if customThumbnail}
						<input
							hidden
							type="file"
							accept="image/*"
							bind:this={thumbnailInput}
							onchange={selectThumbnail}
						/>
						<button
							type="button"
							disabled={disabled || preparingThumbnail}
							class="thumbnail-image-action"
							onclick={() => thumbnailInput?.click()}
						>
							<Icon name="image" size={20} />
							Add custom thumbnail
						</button>
					{/if}
					{#if thumbnailError}<p role="alert">{thumbnailError}</p>{/if}
				</div>
			{/if}
			<label>
				<span class="detail-heading">Video description</span>
				<textarea
					name="draft.video.alt"
					maxlength="1000"
					rows="3"
					value={media.alt}
					{disabled}
					placeholder="Describe what happens in this video"
					oninput={(event) => onalt(event.currentTarget.value)}></textarea>
			</label>
		</div>
		<ComposerDetailAction ondone={close} {disabled} />
	</section>{/if}

<style>
	.video-attachment {
		display: grid;
		gap: 24px;
		padding: 24px;
		background: var(--surface);
		border: 2px solid var(--ink);
		border-radius: 24px;
		box-shadow: 4px 4px 0 var(--ink);
	}
	.video-summary {
		display: flex;
		align-items: center;
		gap: 16px;
		min-width: 0;
	}
	.video-icon {
		display: grid;
		place-items: center;
		width: 64px;
		height: 64px;
		flex: none;
		border: 2px solid var(--ink);
		border-radius: 50%;
		background: var(--butter);
	}
	.video-title {
		min-width: 0;
		display: grid;
		gap: 4px;
	}
	.video-title strong {
		font-size: 20px;
	}
	.video-title small {
		color: var(--ink-2);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-size: 16px;
	}
	.video-details-button,
	.video-actions button {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		border: 1.5px solid var(--hairline);
		background: transparent;
		box-shadow: none;
		color: var(--primary);
	}
	.video-details-button {
		width: 100%;
	}
	.video-actions {
		display: flex;
		justify-content: flex-end;
	}
	.video-actions .danger {
		color: var(--red-ink);
	}
	.video-details-page {
		position: fixed;
		top: 0;
		left: var(--app-content-left, 0px);
		width: var(--app-content-width, 100vw);
		z-index: 25;
		height: var(--composer-detail-height, 100dvh);
		display: flex;
		flex-direction: column;
		background: var(--paper);
		overflow: hidden;
	}
	header {
		justify-content: flex-start;
	}
	header button {
		width: 48px;
		height: 48px;
		padding: 12px;
		border: 0;
		background: transparent;
		box-shadow: none;
		border-radius: 50%;
	}
	.video-details-scroll {
		overflow: auto;
		padding: 16px 16px calc(84px + env(safe-area-inset-bottom));
	}
	.detail-heading {
		margin: 0;
		font-size: 16px;
		font-weight: 700;
	}
	.video-details-scroll .video-preview-heading {
		margin-bottom: 8px;
	}
	video {
		display: block;
		width: 100%;
		max-height: 60dvh;
		margin-bottom: 16px;
		background: black;
	}
	.thumbnail-controls {
		display: grid;
		gap: 12px;
		margin-bottom: 16px;
	}
	.thumbnail-preview {
		display: block;
		max-width: min(100%, 420px);
		max-height: 240px;
		object-fit: cover;
	}
	.thumbnail-grid {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 8px;
		width: min(100%, 420px);
		justify-self: center;
	}
	video,
	.thumbnail-preview,
	.thumbnail-grid button {
		border: 1px solid var(--hairline);
		border-radius: 8px;
	}
	.thumbnail-grid button {
		position: relative;
		display: block;
		padding: 0;
		box-shadow: none;
		overflow: hidden;
		background: var(--surface);
	}
	.thumbnail-grid button[aria-pressed='true'] {
		border-color: var(--primary);
	}
	.thumbnail-grid button[aria-pressed='true']::after {
		content: '';
		position: absolute;
		inset: -1px;
		border: 4px solid var(--primary);
		border-radius: inherit;
		pointer-events: none;
	}
	.thumbnail-frame {
		display: grid;
		place-items: center;
		width: 100%;
		aspect-ratio: 16 / 9;
		overflow: hidden;
	}
	.thumbnail-selected {
		position: absolute;
		inset-block-start: 6px;
		inset-inline-end: 6px;
		display: grid;
		place-items: center;
		width: 24px;
		height: 24px;
		border-radius: 50%;
		background: var(--primary);
		color: var(--on-primary);
	}
	.thumbnail-frame img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.thumbnail-remove {
		justify-self: start;
		border: 0;
		background: transparent;
		box-shadow: none;
		color: var(--red-ink);
	}
	.thumbnail-remove:hover {
		background: color-mix(in srgb, var(--red-ink) 8%, transparent);
	}
	.thumbnail-image-action {
		width: 100%;
		border-color: var(--hairline);
		background: transparent;
		color: var(--primary);
	}
	label {
		display: grid;
		gap: 8px;
	}
	textarea {
		width: 100%;
	}
</style>
