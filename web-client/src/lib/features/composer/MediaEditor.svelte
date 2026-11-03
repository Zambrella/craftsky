<script lang="ts">
	import { onDestroy, tick, untrack } from 'svelte';
	import Icon from '../shell/Icon.svelte';
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
		onBusyChange(busy);
	});
	$effect(() => onEditingChange(editing));
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

<article class="media-card">
	<img
		src={url}
		alt={media.alt || `Image ${index + 1} preview`}
		style={`transform:rotate(${editing ? rotation * 90 : 0}deg);clip-path:inset(${editing ? y : 0}% ${editing ? Math.max(0, 100 - x - width) : 0}% ${editing ? Math.max(0, 100 - y - height) : 0}% ${editing ? x : 0}%)`}
	/>
	{#if describe}<label
			>Image description<textarea
				rows="2"
				maxlength="1000"
				value={media.alt}
				disabled={disabled || busy}
				oninput={(e) => onchange({ ...media, alt: e.currentTarget.value })}
				placeholder="Describe this image for people who cannot see it"
			></textarea></label
		>{/if}
	<div class="toolbar">
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
	</div>
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
			<div class="image-edit-stage">
				<img
					src={url}
					alt="Crop preview"
					style={`transform:rotate(${rotation * 90}deg);clip-path:inset(${y}% ${Math.max(0, 100 - x - width)}% ${Math.max(0, 100 - y - height)}% ${x}%)`}
				/>
			</div>
			<fieldset disabled={disabled || busy}>
				<legend class="sr-only">Crop and rotate</legend><button
					type="button"
					onclick={() => (rotation = (rotation + 1) % 4)}>Rotate 90° ↻</button
				>
				<div class="fields">
					{#each [['x', 'Left'], ['y', 'Top'], ['width', 'Width'], ['height', 'Height']] as [key, name]}<label
							>{name} (%)<input
								type="number"
								min={key === 'width' || key === 'height' ? 1 : 0}
								max="100"
								value={key === 'x'
									? x
									: key === 'y'
										? y
										: key === 'width'
											? width
											: height}
								oninput={(e) => {
									const value = Number(e.currentTarget.value);
									if (key === 'x') x = value;
									else if (key === 'y') y = value;
									else if (key === 'width') width = value;
									else height = value;
								}}
							/></label
						>{/each}
				</div>
			</fieldset>
			{#if error}<p role="alert">{error}</p>{/if}{#if busy}<p role="status">
					Preparing image…
				</p>{/if}{/if}
	</dialog>
	{#if error}<p role="alert" class="error">{error}</p>{/if}
</article>

<style>
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
	.image-edit-stage {
		flex: 1;
		min-height: 200px;
		display: grid;
		place-items: center;
		overflow: hidden;
		margin: 16px;
		border: 1px solid #fff8;
	}
	.image-edit-stage img {
		width: 100%;
		height: 100%;
		max-height: 55dvh;
		object-fit: contain;
		border-radius: 0;
		background: transparent;
	}
	.image-edit-dialog fieldset {
		display: grid;
		gap: 16px;
		border: 0;
		margin: 0;
		padding: 16px 24px calc(24px + env(safe-area-inset-bottom));
	}
	.image-edit-dialog .fields {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: 12px;
	}
	.image-edit-dialog label {
		font-size: 13px;
	}
	.image-edit-dialog input {
		background: #2b2824;
		border: 1px solid #fff8;
		color: #fffaf2;
		padding: 12px 8px;
	}
	.image-edit-dialog fieldset > button {
		justify-self: center;
		background: transparent;
		color: inherit;
		border: 0;
		box-shadow: none;
	}
	.image-edit-dialog p {
		padding: 8px 24px;
	}
	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip-path: inset(50%);
	}
</style>
