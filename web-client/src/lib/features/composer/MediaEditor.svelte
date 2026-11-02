<script lang="ts">
	import { onDestroy } from 'svelte';
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
	$effect(() => {
		onBusyChange(busy);
	});
	$effect(() => onEditingChange(editing));
	onDestroy(() => {
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
		if (!media.blob) return;
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
			onchange({ ...next, id: media.id, alt: media.alt });
			editing = false;
			rotation = 0;
			x = 0;
			y = 0;
			width = 100;
			height = 100;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Could not edit this image.';
		} finally {
			busy = false;
		}
	}
	async function openEditor() {
		if (media.blob) {
			if (editing) cancelEdit();
			else editing = true;
			return;
		}
		if (!media.remoteUrl) return;
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
				cache: 'no-store'
			});
			if (!response.ok) throw new Error('Could not load the published image.');
			const blob = await response.blob();
			if (blob.size > 40_000_000)
				throw new Error('This image is too large to edit.');
			onchange({ ...media, blob });
			editing = true;
		} catch (e) {
			error =
				e instanceof Error ? e.message : 'Could not edit this published image.';
		} finally {
			busy = false;
		}
	}
	function cancelEdit() {
		editing = false;
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
	{#if editing}<fieldset disabled={disabled || busy}>
			<legend>Edit image</legend><button
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
			<button type="button" class="primary" onclick={apply}>Apply edit</button>
			<button type="button" onclick={cancelEdit}>Cancel edit</button>
		</fieldset>{/if}
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
	fieldset {
		display: grid;
		gap: 0.7rem;
	}
</style>
