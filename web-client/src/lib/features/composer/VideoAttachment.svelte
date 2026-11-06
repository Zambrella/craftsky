<script lang="ts">
	import { tick } from 'svelte';
	import { pushState } from '$app/navigation';
	import { page } from '$app/state';
	import type { DraftMedia } from '../drafts/store.ts';
	import { mountComposerPage } from './page-layout';
	import Icon from '../shell/Icon.svelte';
	let {
		media,
		disabled = false,
		autoOpen = false,
		onalt,
		onremove
	}: {
		media: DraftMedia;
		disabled?: boolean;
		autoOpen?: boolean;
		onalt: (value: string) => void;
		onremove: () => void;
	} = $props();
	const uid = $props.id();
	let open = $state(false);
	let url = $state('');
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
		<header>
			<button type="button" aria-label="Back to post" {disabled} onclick={close}
				><Icon name="back" /></button
			>
			<h2>Video details</h2>
		</header>
		<div class="video-details-scroll">
			{#if media.previewUnavailable}<p class="muted">
					Preview unavailable in this browser. You can still upload this video.
				</p>{:else}<video
					use:playback
					controls
					playsinline
					src={url}
					aria-label="Local video preview"><track kind="captions" /></video
				>{/if}
			<label
				>Video description<textarea
					name="draft.video.alt"
					maxlength="1000"
					rows="3"
					value={media.alt}
					{disabled}
					placeholder="Describe what happens in this video"
					oninput={(event) => onalt(event.currentTarget.value)}
				></textarea></label
			>
			<p class="muted">MP4 or MOV · up to ten minutes · up to 300 MB</p>
		</div>
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
		height: 100dvh;
		display: flex;
		flex-direction: column;
		background: var(--paper);
		overflow: hidden;
	}
	header {
		display: flex;
		align-items: center;
		gap: 16px;
		height: 56px;
		flex: none;
		border-bottom: 1.5px solid var(--ink);
		padding: 0 8px;
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
	h2 {
		font-size: 20px;
		font-weight: 600;
		margin: 0;
	}
	.video-details-scroll {
		overflow: auto;
		padding: 16px 16px calc(16px + env(safe-area-inset-bottom));
	}
	video {
		display: block;
		width: 100%;
		max-height: 60dvh;
		margin-bottom: 16px;
		background: black;
	}
	label {
		display: grid;
		gap: 8px;
	}
	textarea {
		width: 100%;
	}
</style>
