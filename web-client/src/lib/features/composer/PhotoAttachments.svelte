<script lang="ts">
	import { onDestroy, tick } from 'svelte';
	import { pushState } from '$app/navigation';
	import { page } from '$app/state';
	import { mountComposerPage } from './page-layout';
	import { contextMenu } from '../social/context-menu';
	import { reorderRow } from '../social/reorder-row';
	import Icon from '../shell/Icon.svelte';
	import MediaEditor from './MediaEditor.svelte';
	import PhotoThumbnail from './PhotoThumbnail.svelte';
	import type { DraftMedia } from '../drafts/store';
	let {
		images,
		required = false,
		disabled = false,
		canAddVideo = false,
		onadd,
		onchange,
		onBusyChange,
		onEditingChange,
		onOpenChange = () => {}
	}: {
		images: DraftMedia[];
		required?: boolean;
		disabled?: boolean;
		canAddVideo?: boolean;
		onadd: (files: FileList | null) => Promise<void>;
		onchange: (images: DraftMedia[]) => void;
		onBusyChange: (id: string, busy: boolean) => void;
		onEditingChange: (id: string, editing: boolean) => void;
		onOpenChange?: (open: boolean) => void;
	} = $props();
	const uid = $props.id();
	let open = $state(false);
	let disposed = false;
	let photoInput = $state<HTMLInputElement>();
	let videoInput = $state<HTMLInputElement>();
	let previous: HTMLElement | undefined;
	let editing = $state<string[]>([]);
	const described = $derived(images.filter((image) => image.alt.trim()).length);
	async function show() {
		previous =
			document.activeElement instanceof HTMLElement
				? document.activeElement
				: undefined;
		await pushState('', { ...page.state, photoDetails: uid });
	}
	function close() {
		if (disabled || editing.length) return;
		history.back();
	}
	$effect(() => {
		const next = page.state.photoDetails === uid;
		if (open && !next && (disabled || editing.length)) {
			void pushState('', { ...page.state, photoDetails: uid });
			return;
		}
		if (open === next) return;
		open = next;
		onOpenChange(next);
		if (!next) void tick().then(() => previous?.focus({ preventScroll: true }));
	});
	async function add(files: FileList | null) {
		await onadd(files);
		await tick();
		if (!disposed && images.length && !open) show();
	}
	function fileDrop(node: HTMLElement) {
		let depth = 0;
		const isFiles = (event: DragEvent) =>
			event.dataTransfer?.types.includes('Files');
		function enter(event: DragEvent) {
			if (!isFiles(event)) return;
			event.preventDefault();
			depth++;
			node.classList.toggle('drag-over', !disabled && !editing.length);
		}
		function over(event: DragEvent) {
			if (!isFiles(event)) return;
			event.preventDefault();
			if (event.dataTransfer)
				event.dataTransfer.dropEffect =
					disabled || editing.length ? 'none' : 'copy';
		}
		function leave(event: DragEvent) {
			if (!isFiles(event)) return;
			if (--depth <= 0) {
				depth = 0;
				node.classList.remove('drag-over');
			}
		}
		function drop(event: DragEvent) {
			if (!isFiles(event)) return;
			event.preventDefault();
			depth = 0;
			node.classList.remove('drag-over');
			if (!disabled && !editing.length) {
				// Keep files available after the browser protects the original drop data.
				const selection = new DataTransfer();
				for (const file of Array.from(event.dataTransfer!.files))
					selection.items.add(file);
				void add(selection.files);
			}
		}
		node.addEventListener('dragenter', enter);
		node.addEventListener('dragover', over);
		node.addEventListener('dragleave', leave);
		node.addEventListener('drop', drop);
		return {
			destroy() {
				node.removeEventListener('dragenter', enter);
				node.removeEventListener('dragover', over);
				node.removeEventListener('dragleave', leave);
				node.removeEventListener('drop', drop);
			}
		};
	}
	function move(from: number, to: number) {
		if (disabled || from < 0 || to < 0 || to >= images.length) return;
		const next = [...images];
		next.splice(to, 0, next.splice(from, 1)[0]);
		onchange(next);
	}
	onDestroy(() => {
		disposed = true;
	});
</script>

{#snippet header()}
	<div class="photos-heading">
		<h2>
			{images.length ? 'Photos' : 'Media'}{#if required}<span>required</span
				>{/if}
		</h2>
		{#if images.length}
			<p class="described">
				<span aria-hidden="true">T</span>{described} / {images.length} described
			</p>
			<p>
				{images.length}/8 · drag to reorder · first is the cover
			</p>
		{/if}
	</div>
{/snippet}
{#snippet picker(video: boolean)}
	{#if images.length < 8}
		{#if video}
			<details class="media-picker" use:contextMenu>
				<summary
					class="photo-picker"
					use:fileDrop
					aria-disabled={disabled}
					onclick={(event) => {
						if (disabled) {
							event.preventDefault();
							event.stopPropagation();
						}
					}}
					><span class="add-icon"><Icon name="plus" size={34} /></span><span
						><strong>Add photos or a video</strong><small
							>Choose up to 8 photos or a video, or drag and drop.</small
						></span
					></summary
				>
				<div class="menu-items">
					<button type="button" {disabled} onclick={() => photoInput?.click()}
						><Icon name="image" /><span
							>Choose photos<small
								>Up to {8 - images.length} photos · 40 MB each</small
							></span
						></button
					>
					<button type="button" {disabled} onclick={() => videoInput?.click()}
						><Icon name="video" /><span
							>Choose a video<small
								>MP4 or MOV · up to 10 minutes · 300 MB</small
							></span
						></button
					>
				</div>
			</details>
			<input
				hidden
				bind:this={photoInput}
				type="file"
				aria-label="Add images"
				accept="image/jpeg,image/png,image/webp"
				multiple
				{disabled}
				onchange={(event) => {
					void add(event.currentTarget.files);
					event.currentTarget.value = '';
				}}
			/>
			<input
				hidden
				bind:this={videoInput}
				type="file"
				aria-label="Choose a video"
				accept="video/mp4,video/quicktime,.mp4,.mov"
				{disabled}
				onchange={(event) => {
					void add(event.currentTarget.files);
					event.currentTarget.value = '';
				}}
			/>
		{:else}<label class="photo-picker" use:fileDrop>
				<span class="add-icon"><Icon name="plus" size={34} /></span>
				<span
					><strong>{images.length ? 'Add another photo' : 'Add a photo'}</strong
					>{#if images.length}<small>Up to {8 - images.length} more</small
						>{/if}<span class="drop-hint">Or drag and drop photos here</span
					></span
				>
				<input
					type="file"
					aria-label="Add images"
					accept="image/jpeg,image/png,image/webp"
					multiple
					{disabled}
					onchange={(event) => {
						void add(event.currentTarget.files);
						event.currentTarget.value = '';
					}}
				/>
			</label>{/if}{/if}
{/snippet}
{#if images.length}
	<button type="button" class="photo-summary" {disabled} onclick={show}>
		<span class="summary-title">Photos<Icon name="edit" /></span>
		<span class="thumbnails"
			>{#each images as media (media.id)}<PhotoThumbnail {media} />{/each}</span
		>
		<span>Reorder photos, remove them or add alt text</span><small
			>{described} / {images.length} described</small
		>
	</button>
	<div class="clear-photos">
		<button type="button" {disabled} onclick={() => onchange([])}
			><Icon name="trash" />Clear photos</button
		>
	</div>
{:else}
	{@render header()}{@render picker(canAddVideo)}
{/if}
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
		class="photo-details-page"
		aria-label="Photo details"
		use:mountComposerPage
	>
		<header class="photo-page-header">
			<button
				type="button"
				aria-label="Back to post"
				disabled={disabled || editing.length > 0}
				onclick={close}><Icon name="back" /></button
			>
			<h2>Photo details</h2>
		</header>
		<div class="photo-page-scroll">
			{@render header()}
			{#each images as media, index (media.id)}<div
					class="photo-row"
					role="group"
					aria-label={`Photo ${index + 1}`}
					use:reorderRow={{
						group: `photos-${uid}`,
						index,
						enabled: !disabled && editing.length === 0 && images.length > 1,
						onmove: move
					}}
				>
					<MediaEditor
						{media}
						{index}
						count={images.length}
						{disabled}
						nativePhoto
						onBusyChange={(value) => onBusyChange(media.id, value)}
						onEditingChange={(value) => {
							editing = value
								? [...editing, media.id]
								: editing.filter((id) => id !== media.id);
							onEditingChange(media.id, value);
						}}
						onchange={(value) =>
							onchange(
								images.map((image) => (image.id === media.id ? value : image))
							)}
						onremove={() =>
							onchange(images.filter((image) => image.id !== media.id))}
						onmove={(direction) => move(index, index + direction)}
					/>
				</div>{/each}
			{@render picker(false)}
		</div>
	</section>{/if}

<style>
	.photo-row {
		margin: 0 3px 27px 0;
	}
	.photo-row :global(.media-card.native-photo) {
		margin: 0;
	}
	.photos-heading h2 {
		display: block;
		font-size: 16px;
		line-height: 1.3;
		margin: 0 0 8px;
	}
	.photos-heading h2 span {
		margin-left: 8px;
		font-size: 12px;
		letter-spacing: 1.5px;
	}
	.photos-heading p {
		font-size: 14px;
		font-weight: 600;
		color: var(--ink-3);
		margin: 0 0 12px;
	}
	.described {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.described > span {
		font-family: serif;
		font-size: 24px;
		line-height: 24px;
		font-weight: 400;
	}
	:global(.composer) .media-picker,
	:global(.composer) .media-picker[open] {
		border: 0;
		padding: 0;
	}
	:global(.composer) .media-picker > summary {
		margin: 0;
		list-style: none;
	}
	:global(.composer) .media-picker > summary::after {
		content: none;
	}
	.media-picker summary::-webkit-details-marker {
		display: none;
	}
	.media-picker .menu-items small {
		display: block;
		color: var(--ink-2);
		font-size: 13px;
		font-weight: 400;
		margin-top: 4px;
	}
	.photo-picker {
		position: relative;
		display: flex;
		flex-direction: row;
		align-items: center;
		gap: 16px;
		width: 100%;
		padding: 16px;
		min-height: 88px;
		border: 2px dashed var(--ink);
		border-radius: 24px;
		cursor: pointer;
	}
	.photo-picker:focus-visible,
	.photo-picker:has(input:focus-visible) {
		outline: 3px solid var(--primary);
		outline-offset: 3px;
	}
	.photo-picker:global(.drag-over) {
		border-color: var(--primary);
		background: color-mix(in srgb, var(--primary) 8%, var(--paper));
	}
	.drop-hint {
		display: block;
		font-size: 13px;
		color: var(--ink-3);
		margin-top: 4px;
	}
	.photo-picker input {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		opacity: 0;
		cursor: pointer;
	}
	.photo-picker strong,
	.photo-picker small {
		display: block;
	}
	.photo-picker strong {
		font-size: 16px;
		font-weight: 700;
	}
	.photo-picker small {
		font-size: 14px;
		color: var(--ink-3);
		margin-top: 4px;
	}
	.add-icon {
		display: grid;
		place-items: center;
		width: 56px;
		height: 56px;
		flex: none;
		border-radius: 50%;
		border: 2px solid var(--ink);
		background: var(--butter);
	}
	.photo-summary {
		width: 100%;
		display: flex;
		flex-direction: column;
		align-items: stretch;
		gap: 8px;
		padding: 16px;
		border: 1.5px solid var(--ink);
		border-radius: 14px;
		background: var(--surface);
		box-shadow: none;
		text-align: left;
		font-size: 14px;
		font-weight: 400;
	}
	.summary-title {
		display: flex;
		align-items: center;
		justify-content: space-between;
		font-size: 16px;
		font-weight: 600;
	}
	.thumbnails {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin: 4px 0 0;
	}
	.photo-summary small {
		font-size: 13px;
	}
	.clear-photos {
		display: flex;
		justify-content: flex-end;
	}
	.clear-photos button {
		border: 0;
		box-shadow: none;
		background: transparent;
		color: var(--primary);
		gap: 8px;
	}
	.photo-details-page {
		position: fixed;
		top: 0;
		left: var(--app-content-left, 0px);
		width: var(--app-content-width, 100vw);
		z-index: 25;
		height: 100dvh;
		max-width: none;
		max-height: none;
		margin: 0;
		padding: 0;
		border: 0;
		border-radius: 0;
		background: var(--paper);
		overflow: hidden;
	}
	.photo-details-page {
		display: flex;
		flex-direction: column;
	}
	.photo-page-header {
		display: flex;
		align-items: center;
		gap: 16px;
		flex: none;
		height: 56px;
		border-bottom: 1.5px solid var(--ink);
		padding: 0 8px;
	}
	.photo-page-header button {
		width: 48px;
		height: 48px;
		padding: 12px;
		border: 0;
		background: transparent;
		box-shadow: none;
		border-radius: 50%;
	}
	.photo-page-header h2 {
		font-size: 20px;
		font-weight: 600;
		margin: 0;
	}
	.photo-page-scroll {
		overflow: auto;
		padding: 16px 16px calc(16px + env(safe-area-inset-bottom));
	}
</style>
