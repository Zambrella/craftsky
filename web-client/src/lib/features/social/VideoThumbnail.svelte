<script lang="ts">
	import type { Video } from '#lib/api/types.ts';
	import Icon from '../shell/Icon.svelte';
	import { safeLink } from './model';
	let { video }: { video: Video } = $props();
	let failed = $state('');
	const thumbnail = $derived(safeLink(video.thumbnail || ''));
</script>

<div
	class="video-thumbnail"
	role="img"
	aria-label={video.alt ? `Video: ${video.alt}` : 'Video'}
>
	{#if thumbnail && failed !== thumbnail}<img
			src={thumbnail}
			alt=""
			loading="lazy"
			onerror={() => {
				failed = thumbnail;
			}}
		/>{/if}
	<span class="video-badge"><Icon name="video" size={28} /></span>
</div>

<style>
	.video-thumbnail {
		position: relative;
		width: 100%;
		aspect-ratio: 16 / 9;
		margin-top: 8px;
		overflow: hidden;
		border: 1px solid var(--hairline);
		border-radius: 8px;
		background: var(--paper-2);
	}
	img {
		position: absolute;
		inset: 0;
		display: block;
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.video-badge {
		position: absolute;
		inset: 0;
		display: grid;
		place-content: center;
	}
	.video-badge :global(svg) {
		box-sizing: content-box;
		background: rgb(0 0 0 / 75%);
		color: white;
		padding: 12px;
		border-radius: 50%;
	}
</style>
