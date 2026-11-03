<script lang="ts">
	import { onDestroy, tick } from 'svelte';
	import { beforeNavigate } from '$app/navigation';
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
	let dialog: HTMLDialogElement;
	let open = $state(false);
	let disposed = false;
	let previous: HTMLElement | undefined;
	let dragged: string | undefined;
	let editing = $state<string[]>([]);
	const described = $derived(images.filter((image) => image.alt.trim()).length);
	function show() {
		previous =
			document.activeElement instanceof HTMLElement
				? document.activeElement
				: undefined;
		open = true;
		onOpenChange(true);
		dialog.showModal();
	}
	function close() {
		if (disabled || editing.length) return;
		dialog.close();
		open = false;
		onOpenChange(false);
		previous?.focus({ preventScroll: true });
	}
	async function add(files: FileList | null) {
		await onadd(files);
		await tick();
		if (!disposed && images.length && !open) show();
	}
	function move(from: number, to: number) {
		if (disabled || from < 0 || to < 0 || to >= images.length) return;
		const next = [...images];
		next.splice(to, 0, next.splice(from, 1)[0]);
		onchange(next);
	}
	beforeNavigate((navigation) => {
		if (open && !navigation.willUnload) {
			navigation.cancel();
			close();
		}
	});
	onDestroy(() => {
		disposed = true;
		dialog?.close();
	});
</script>

{#snippet header()}
	<div class="photos-heading">
		<h2>
			Photos{#if required}<span>required</span>{/if}
		</h2>
		<p class="described">
			<span aria-hidden="true">T</span>{images.length
				? `${described} / ${images.length} described`
				: '0 described'}
		</p>
		<p>
			{images.length
				? `${images.length}/8 · drag to reorder · first is the cover`
				: 'Up to 8 photos'}
		</p>
	</div>
{/snippet}
{#snippet picker(video: boolean)}
	{#if images.length < 8}<label class="photo-picker"
			><span class="add-icon"><Icon name="plus" size={34} /></span><span
				><strong
					>{video
						? 'Add photos or a video'
						: images.length
							? 'Add another photo'
							: 'Add a photo'}</strong
				><small
					>{video
						? 'Up to 8 photos or one video'
						: images.length
							? `Up to ${8 - images.length} more`
							: 'Up to 8 photos'}</small
				></span
			>
			<input
				type="file"
				aria-label={video ? 'Add photos or a video' : 'Add images'}
				accept={video
					? 'image/jpeg,image/png,image/webp,video/mp4'
					: 'image/jpeg,image/png,image/webp'}
				multiple
				{disabled}
				onchange={(event) => {
					void add(event.currentTarget.files);
					event.currentTarget.value = '';
				}}
			/>
		</label>{/if}
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
<dialog
	bind:this={dialog}
	class="photo-details-page"
	aria-label="Photo details"
	oncancel={(event) => {
		event.preventDefault();
		close();
	}}
>
	{#if open}<header class="photo-page-header">
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
					role="group"
					aria-label={`Photo ${index + 1}`}
					ondragover={(event) => {
						if (dragged) event.preventDefault();
					}}
					ondrop={(event) => {
						event.preventDefault();
						move(
							images.findIndex((image) => image.id === dragged),
							index
						);
						dragged = undefined;
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
						ondrag={(value) => (dragged = value ? media.id : undefined)}
					/>
				</div>{/each}
			{@render picker(false)}
		</div>{/if}
</dialog>

<style>
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
		inset: 0;
		width: min(680px, 100vw);
		height: 100dvh;
		max-width: none;
		max-height: none;
		margin: auto;
		padding: 0;
		border: 0;
		border-radius: 0;
		background: var(--paper);
		overflow: hidden;
	}
	.photo-details-page[open] {
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
