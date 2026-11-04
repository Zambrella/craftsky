<script lang="ts">
	import Snackbar from '../social/Snackbar.svelte';
	import { onDestroy, tick, untrack } from 'svelte';
	import Icon from '../shell/Icon.svelte';
	import CropSurface from './CropSurface.svelte';
	import type { DraftMedia } from '../drafts/store.ts';
	import { prepareImage } from './media.ts';
	let {
		media,
		index,
		count,
		onchange,
		onremove,
		onmove,
		disabled = false,
		describe = true,
		allowReorder = true,
		nativePhoto = false,
		ondrag = () => {},
		onBusyChange = () => {},
		onEditingChange = () => {}
	}: {
		media: DraftMedia;
		index: number;
		count: number;
		onchange: (media: DraftMedia) => void;
		onremove: () => void;
		onmove: (direction: number) => void;
		disabled?: boolean;
		describe?: boolean;
		allowReorder?: boolean;
		nativePhoto?: boolean;
		ondrag?: (dragging: boolean) => void;
		onBusyChange?: (busy: boolean) => void;
		onEditingChange?: (editing: boolean) => void;
	} = $props();
	let url = $state('');
	let error = $state('');
	let busy = $state(false);
	let editing = $state(false);
	let dialog: HTMLDialogElement;
	let trigger = $state<HTMLButtonElement>();
	let disposed = false;
	let operation = 0;
	const remote = new AbortController();
	$effect(() => {
		media.id;
		media.remoteUrl;
		untrack(() => {
			operation++;
			cancelEdit();
			busy = false;
			error = '';
		});
	});
	$effect(() => {
		if (!dialog) return;
		if (editing && !dialog.open) dialog.showModal();
		else if (!editing && dialog.open) dialog.close();
	});
	$effect(() => {
		const value = busy;
		untrack(() => onBusyChange(value));
	});
	$effect(() => {
		const value = editing;
		untrack(() => onEditingChange(value));
	});
	onDestroy(() => {
		disposed = true;
		operation++;
		remote.abort();
		dialog?.close();
		onBusyChange(false);
		onEditingChange(false);
	});
	let x = $state(0),
		y = $state(0),
		width = $state(100),
		height = $state(100),
		rotation = $state(0);
	$effect(() => {
		const blob = media.blob;
		const next = blob ? URL.createObjectURL(blob) : media.remoteUrl || '';
		url = next;
		return () => {
			if (blob) URL.revokeObjectURL(next);
		};
	});
	async function apply() {
		if (!media.blob || busy || disabled) return;
		const seq = ++operation,
			source = media;
		busy = true;
		error = '';
		try {
			const next = await prepareImage(media.blob, media.name, {
				rotation,
				crop: {
					x: x / 100,
					y: y / 100,
					width: width / 100,
					height: height / 100
				}
			});
			if (
				disposed ||
				operation !== seq ||
				source.id !== media.id ||
				source.blob !== media.blob
			)
				return;
			onchange({ ...next, id: media.id, alt: media.alt });
			cancelEdit();
			rotation = 0;
			x = 0;
			y = 0;
			width = 100;
			height = 100;
		} catch (e) {
			if (!disposed && operation === seq)
				error = e instanceof Error ? e.message : 'Could not edit this image.';
		} finally {
			if (!disposed && operation === seq) busy = false;
		}
	}
	async function openEditor() {
		if (busy || disabled || disposed) return;
		if (media.blob) {
			if (editing) cancelEdit();
			else editing = true;
			return;
		}
		if (!media.remoteUrl) return;
		const seq = ++operation,
			source = media;
		busy = true;
		error = '';
		try {
			const destination = new URL(media.remoteUrl, location.origin);
			if (
				destination.protocol !== 'https:' &&
				!(
					destination.protocol === 'http:' &&
					['localhost', '127.0.0.1', '[::1]'].includes(destination.hostname)
				)
			)
				throw new Error('This published image cannot be edited.');
			const response = await fetch(destination, {
				credentials: 'omit',
				redirect: 'error',
				referrerPolicy: 'no-referrer',
				cache: 'no-store',
				signal: remote.signal
			});
			if (!response.ok) throw new Error('Could not load the published image.');
			const blob = await response.blob();
			if (blob.size > 40_000_000)
				throw new Error('This image is too large to edit.');
			if (
				disposed ||
				operation !== seq ||
				source.id !== media.id ||
				source.remoteUrl !== media.remoteUrl
			)
				return;
			onchange({ ...media, blob });
			editing = true;
		} catch (e) {
			if (!disposed && operation === seq)
				error =
					e instanceof Error
						? e.message
						: 'Could not edit this published image.';
		} finally {
			if (!disposed && operation === seq) busy = false;
		}
	}
	function cancelEdit() {
		dialog?.close();
		editing = false;
		void tick().then(() => {
			if (!disposed && trigger?.isConnected && !trigger.disabled)
				trigger.focus({ preventScroll: true });
		});
		rotation = 0;
		x = 0;
		y = 0;
		width = 100;
		height = 100;
	}
</script>

<article class="media-card" class:native-photo={nativePhoto}>
	{#if nativePhoto}<div class="photo-controls">
			<span class="photo-number" class:first={index === 0}>{index + 1}</span
			><span class="control-spacer"></span><button
				type="button"
				disabled={disabled || busy || index === 0}
				onclick={() => onmove(-1)}
				aria-label={`Move image ${index + 1} earlier`}
				><Icon name="moveUp" bold /></button
			><button
				type="button"
				disabled={disabled || busy || index === count - 1}
				onclick={() => onmove(1)}
				aria-label={`Move image ${index + 1} later`}
				><span class="down-arrow"><Icon name="moveUp" bold /></span></button
			><button
				type="button"
				class="remove-photo"
				disabled={disabled || busy}
				onclick={onremove}
				aria-label={`Remove image ${index + 1}`}
				><Icon name="trash" bold /></button
			><button
				type="button"
				class="photo-drag"
				aria-label={`Drag to reorder image ${index + 1}`}
				draggable={!disabled && !busy}
				ondragstart={(event) => {
					ondrag(true);
					event.dataTransfer?.setData('text/plain', media.id);
				}}
				ondragend={() => ondrag(false)}
				><span aria-hidden="true">⠿</span></button
			>
		</div>{/if}
	<div class="photo-preview">
		<img
			src={url}
			alt={media.alt || `Image ${index + 1} preview`}
			style={`transform:rotate(${editing ? rotation * 90 : 0}deg);clip-path:inset(${editing ? y : 0}% ${editing ? Math.max(0, 100 - x - width) : 0}% ${editing ? Math.max(0, 100 - y - height) : 0}% ${editing ? x : 0}%)`}
		/>
		{#if nativePhoto && (media.blob || media.remoteUrl)}<button
				bind:this={trigger}
				type="button"
				class="edit-photo"
				aria-label="Edit photo"
				disabled={disabled || busy}
				onclick={openEditor}><Icon name="edit" /></button
			>{/if}
	</div>
	{#if describe}<label
			><span class="alt-heading"
				>{#if nativePhoto}<span
						><span class="alt-text-icon" aria-hidden="true">T</span>ALT TEXT</span
					><small class:described={!!media.alt.trim()}
						>{media.alt.trim() ? 'Described' : 'Help screen readers'}</small
					>{:else}Image description{/if}</span
			><textarea
				aria-label="Image description"
				rows="2"
				maxlength="1000"
				value={media.alt}
				disabled={disabled || busy}
				oninput={(e) => onchange({ ...media, alt: e.currentTarget.value })}
				placeholder={nativePhoto
					? 'Describe the image for someone who cannot see it, including the craft, materials, colors, and important details.'
					: 'Describe this image for people who cannot see it'}
			></textarea>{#if nativePhoto}<span class="alt-count"
					>{Array.from(media.alt).length}/1000</span
				>{/if}</label
		>{/if}
	{#if !nativePhoto}<div class="toolbar">
			{#if allowReorder}<button
					type="button"
					disabled={disabled || busy || index === 0}
					onclick={() => onmove(-1)}
					aria-label={`Move image ${index + 1} earlier`}>←</button
				><button
					type="button"
					disabled={disabled || busy || index === count - 1}
					onclick={() => onmove(1)}
					aria-label={`Move image ${index + 1} later`}>→</button
				>{/if}{#if media.blob || media.remoteUrl}<button
					bind:this={trigger}
					type="button"
					disabled={disabled || busy}
					onclick={openEditor}>Crop / rotate</button
				>{/if}<button
				type="button"
				class="danger"
				disabled={disabled || busy}
				onclick={onremove}>Remove</button
			>
		</div>{/if}
	<dialog
		bind:this={dialog}
		class="image-edit-dialog"
		aria-label="Edit photo"
		oncancel={(event) => {
			event.preventDefault();
			if (!busy) cancelEdit();
		}}
		onclose={() => {
			if (editing && !disposed) cancelEdit();
		}}
	>
		{#if editing}<header class="image-edit-heading">
				<button
					type="button"
					aria-label="Cancel edit"
					disabled={busy}
					onclick={cancelEdit}><Icon name="close" /></button
				>
				<h2>Edit photo</h2>
				<button
					type="button"
					aria-label="Apply edit"
					title="Done"
					disabled={disabled || busy}
					onclick={apply}><Icon name="check" /></button
				>
			</header>
			<CropSurface
				{url}
				{rotation}
				{x}
				{y}
				{width}
				{height}
				disabled={disabled || busy}
				onchange={(crop) => {
					x = crop.x;
					y = crop.y;
					width = crop.width;
					height = crop.height;
				}}
			/>
			<div class="image-edit-tools">
				<button
					type="button"
					disabled={disabled || busy}
					onclick={() => (rotation = (rotation + 1) % 4)}
					aria-label="Rotate photo"
					><Icon name="refresh" /><span>Rotate</span></button
				>
			</div>
			{#if error}<Snackbar
					message={error}
					warning
					onDismiss={() => (error = '')}
				/>{/if}{#if busy}<p role="status">Preparing image…</p>{/if}{/if}
	</dialog>
	{#if error}<Snackbar
			message={error}
			warning
			onDismiss={() => (error = '')}
		/>{/if}
</article>

<style>
	.photo-preview {
		position: relative;
	}
	.media-card.native-photo {
		background: var(--surface);
		border: 1.5px solid var(--ink);
		border-radius: 14px;
		box-shadow: 3px 3px var(--shadow);
		margin: 0 3px 27px 0;
		padding: 16px;
		gap: 12px;
	}
	.native-photo .photo-preview > img {
		height: auto;
		max-height: 420px;
		border: 1px solid var(--ink);
		border-radius: 0;
		display: block;
	}
	.photo-controls {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.control-spacer {
		flex: 1;
	}
	.photo-number {
		display: grid;
		place-items: center;
		width: 50px;
		height: 50px;
		flex: none;
		border: 2px solid var(--ink);
		border-radius: 50%;
		background: var(--paper-2);
		font-size: 20px;
		font-weight: 800;
	}
	.photo-number.first {
		background: var(--primary);
		color: white;
	}
	.photo-controls button {
		display: grid;
		place-items: center;
		width: 52px;
		height: 52px;
		padding: 12px;
		flex: none;
		border: 2px solid var(--ink);
		border-radius: 50%;
		box-shadow: none;
		background: var(--surface);
	}
	.photo-controls button:disabled {
		color: var(--ink-4);
		border-color: var(--ink-4);
		opacity: 1;
	}
	.photo-controls .remove-photo {
		color: #f03a2e;
	}
	.photo-controls .photo-drag {
		width: 34px;
		padding: 0;
		border: 0;
		color: var(--ink-3);
		font-size: 34px;
		cursor: grab;
	}
	.down-arrow {
		display: flex;
		transform: rotate(180deg);
	}
	.edit-photo {
		position: absolute;
		bottom: 4px;
		right: 4px;
		width: 48px;
		height: 48px;
		padding: 12px;
		border: 0;
		border-radius: 50%;
		background: var(--primary);
		color: white;
		box-shadow: none;
	}
	.alt-heading {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
	}
	.native-photo .alt-heading > span {
		display: flex;
		align-items: center;
		gap: 8px;
		color: var(--ink-2);
		font-size: 12px;
		font-weight: 800;
		letter-spacing: 1.5px;
	}
	.alt-text-icon {
		font-family: serif;
		font-size: 24px;
		font-weight: 400;
	}
	.alt-heading small {
		font-size: 12px;
		font-weight: 700;
		color: var(--red-ink);
	}
	.alt-heading small.described {
		color: light-dark(#4e6e1f, #a8c779);
	}
	.native-photo label {
		margin-top: 4px;
	}
	.native-photo textarea {
		min-height: 128px;
		resize: vertical;
	}
	.alt-count {
		text-align: right;
		font-size: 14px;
		font-weight: 400;
	}

	.media-card {
		padding: 1rem;
		border: 1px solid var(--hairline);
		border-radius: 12px;
		display: grid;
		gap: 0.7rem;
		min-width: 0;
	}
	.media-card img {
		width: 100%;
		height: 220px;
		object-fit: contain;
		border-radius: 8px;
		background: var(--paper-2);
	}
	.image-edit-dialog {
		inset: 0;
		margin: auto;
		padding: 0;
		width: min(800px, 100vw);
		height: 100dvh;
		max-width: none;
		max-height: none;
		border: 0;
		border-radius: 0;
		background: #191715;
		color: #fffaf2;
		overflow: auto;
	}
	.image-edit-dialog[open] {
		display: flex;
		flex-direction: column;
	}
	.image-edit-dialog::backdrop {
		background: #000b;
	}
	.image-edit-heading {
		display: flex;
		align-items: center;
		gap: 16px;
		height: 56px;
		min-height: 56px;
		padding: 8px 16px;
	}
	.image-edit-heading h2 {
		flex: 1;
		margin: 0;
		color: inherit;
		font-family: inherit;
		font-size: 20px;
		font-weight: 700;
		line-height: 1.3;
	}
	.image-edit-heading button {
		display: grid;
		place-items: center;
		width: 40px;
		min-height: 40px;
		padding: 8px;
		border: 0;
		border-radius: 50%;
		background: transparent;
		color: inherit;
		box-shadow: none;
	}
	.image-edit-tools {
		display: flex;
		justify-content: center;
		padding: 12px 24px calc(20px + env(safe-area-inset-bottom));
	}
	.image-edit-tools button {
		display: flex;
		flex-direction: column;
		gap: 8px;
		background: transparent;
		color: #f5efe4;
		border: 0;
		box-shadow: none;
		font-size: 12px;
	}

	.image-edit-dialog p {
		padding: 8px 24px;
	}
</style>
