<script lang="ts">
	import type { Draft, DraftMedia } from '../drafts/store';
	import Icon from '../shell/Icon.svelte';
	import { safeURL } from '#lib/public.ts';
	let {
		external,
		image,
		thumbnailUrl,
		loading = false,
		index = 1,
		total = 1,
		disabled = false,
		onremove,
		onprevious = () => {},
		onnext = () => {}
	}: {
		external: NonNullable<Draft['external']>;
		image?: DraftMedia;
		thumbnailUrl?: string;
		loading?: boolean;
		index?: number;
		total?: number;
		disabled?: boolean;
		onremove: () => void;
		onprevious?: () => void;
		onnext?: () => void;
	} = $props();
	let url = $state('');
	$effect(() => {
		const blob = image?.blob;
		const next = blob ? URL.createObjectURL(blob) : safeURL(thumbnailUrl) || '';
		url = next;
		return () => {
			if (blob) URL.revokeObjectURL(next);
		};
	});
	const host = $derived.by(() => {
		try {
			return new URL(external.uri).hostname;
		} catch {
			return external.uri;
		}
	});
</script>

<section class="composer-link-card" aria-label="Link preview">
	{#if url}<img
			src={url}
			alt={`Link preview thumbnail: ${external.title}`}
			referrerpolicy="no-referrer"
		/>{/if}
	<div class="preview-title">
		<h2>{external.title}</h2>
		<button
			type="button"
			aria-label="Remove preview"
			title="Remove preview"
			{disabled}
			onclick={onremove}><Icon name="close" /></button
		>
	</div>
	<div class="preview-copy">
		{#if external.description}<p>{external.description}</p>{/if}<small
			>{host}</small
		>
	</div>
	<div class="preview-navigation">
		<button
			type="button"
			aria-label="Previous link preview"
			disabled={disabled || total < 2}
			onclick={onprevious}><Icon name="back" /></button
		><span aria-live="polite">Link preview {index} of {total}</span><button
			type="button"
			aria-label="Next link preview"
			disabled={disabled || total < 2}
			onclick={onnext}><Icon name="chevron" /></button
		>
	</div>
	{#if loading}<span role="status" class="preview-loading"
			>Loading link preview</span
		>{/if}
</section>

<style>
	.composer-link-card {
		border: 1px solid var(--hairline);
		border-radius: 12px;
		overflow: hidden;
		background: var(--paper-2);
		min-width: 0;
	}
	.composer-link-card > img {
		display: block;
		width: 100%;
		height: 180px;
		object-fit: cover;
	}
	.preview-title {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 8px 4px 0 12px;
	}
	.preview-title h2 {
		flex: 1;
		min-width: 0;
		font-size: 16px;
		line-height: 1.4;
		font-weight: 600;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
	.preview-title button,
	.preview-navigation button {
		border: 0;
		background: transparent;
		box-shadow: none;
		min-width: 44px;
		min-height: 44px;
		padding: 8px;
		color: var(--ink);
	}
	.preview-copy {
		padding: 0 12px;
		display: grid;
		gap: 4px;
	}
	.preview-copy p {
		font-size: 14px;
		line-height: 1.5;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
		overflow-wrap: anywhere;
	}
	.preview-copy small {
		font-size: 13px;
		color: var(--ink-3);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.preview-navigation {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		font-size: 13px;
	}
	.preview-loading {
		display: block;
		font-size: 12px;
		padding: 0 12px 12px;
		color: var(--ink-3);
	}
</style>
