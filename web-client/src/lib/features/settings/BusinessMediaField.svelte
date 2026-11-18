<script lang="ts">
	import Icon from '../shell/Icon.svelte';
	import { safeURL } from '#lib/public.ts';
	import type { BusinessImage } from './model';
	let {
		image,
		required = false,
		disabled = false,
		onupload,
		onedit,
		onremove
	}: {
		image?: BusinessImage;
		required?: boolean;
		disabled?: boolean;
		onupload: (event: Event) => void;
		onedit: () => void;
		onremove?: () => void;
	} = $props();
</script>

<div class="business-media">
	<p class="media-heading">
		Media{#if required}<small>required</small>{/if}
	</p>
	{#if safeURL(image?.thumb)}<img
			class="media-preview"
			src={safeURL(image?.thumb)}
			alt={image?.alt || ''}
		/>{/if}
	<label class="photo-picker" class:has-image={!!image}>
		<input
			type="file"
			accept="image/jpeg,image/png,image/webp"
			aria-label={image
				? 'Replace image'
				: required
					? 'Product image'
					: 'Event image · optional'}
			{disabled}
			onchange={onupload}
		/>
		<span class="photo-plus"><Icon name="plus" size={32} /></span>
		<span>{image ? 'Replace image' : 'Add a photo'}</span>
	</label>
	{#if image}<div class="actions">
			<button type="button" {disabled} onclick={onedit}>Edit image</button>
			{#if onremove}<button type="button" {disabled} onclick={onremove}
					>Remove image</button
				>{/if}
		</div>{/if}
</div>

<style>
	.business-media {
		display: grid;
		gap: 12px;
	}
	.media-heading {
		display: flex;
		gap: 10px;
		align-items: baseline;
		font-weight: 700;
	}
	small {
		font-size: 14px;
		font-weight: 400;
		color: var(--ink-2);
	}
	.media-preview {
		width: 100%;
		max-height: 280px;
		object-fit: contain;
		border-radius: 12px;
		background: var(--paper-2);
	}
	:global(.settings-workspace) .photo-picker {
		position: relative;
		display: flex;
		align-items: center;
		gap: 16px;
		padding: 16px;
		min-height: 88px;
		border: 2px dashed var(--ink);
		border-radius: 22px;
		cursor: pointer;
	}
	.photo-picker:focus-within {
		outline: 3px solid var(--primary);
		outline-offset: 3px;
	}
	.photo-picker:has(input:disabled) {
		opacity: 0.5;
		cursor: default;
	}
	.photo-picker input {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		opacity: 0;
		cursor: inherit;
	}
	.photo-plus {
		border: 2px solid var(--ink);
		border-radius: 50%;
		background: var(--highlight, #f7d868);
		width: 52px;
		height: 52px;
		display: grid;
		place-items: center;
		flex-shrink: 0;
	}
</style>
