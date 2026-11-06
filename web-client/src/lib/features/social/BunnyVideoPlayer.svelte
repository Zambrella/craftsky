<script lang="ts">
	import { onMount } from 'svelte';
	import type { Video } from '#lib/api/client.ts';
	import { connectBunnyPlayer } from './bunny-player.ts';
	let {
		video,
		embed,
		onfallback
	}: { video: Video; embed: string; onfallback: () => void } = $props();
	let src = $state('');
	let loading = $state(true);
	const aspect = $derived(
		video.aspectRatio &&
			video.aspectRatio.width > 0 &&
			video.aspectRatio.height > 0
			? `${video.aspectRatio.width} / ${video.aspectRatio.height}`
			: '16 / 9'
	);
	onMount(() => {
		const url = new URL(embed);
		for (const [key, value] of Object.entries({
			autoplay: 'false',
			muted: 'false',
			preload: 'false',
			playsinline: 'true',
			rememberSettings: 'false',
			rememberPosition: 'false',
			levelCap: 'true',
			context: crypto.randomUUID()
		}))
			url.searchParams.set(key, value);
		src = url.href;
	});
	function connect(frame: HTMLIFrameElement) {
		return connectBunnyPlayer(frame, {
			ready: () => {
				loading = false;
			},
			failed: onfallback
		});
	}
</script>

<div class="bunny-video" style:aspect-ratio={aspect}>
	{#if src}<iframe
			use:connect
			{src}
			title={video.alt || 'Post video'}
			loading="lazy"
			allow="autoplay; encrypted-media; picture-in-picture; fullscreen"
			allowfullscreen
			referrerpolicy="strict-origin-when-cross-origin"
			onerror={onfallback}
		></iframe>{/if}
	{#if loading}<span class="loader" role="status" aria-label="Loading video"
			><span class="spinner"></span></span
		>{/if}
</div>

<style>
	.bunny-video {
		position: relative;
		width: 100%;
		background: #000;
	}
	iframe {
		display: block;
		width: 100%;
		height: 100%;
		border: 0;
	}
	.loader {
		position: absolute;
		inset: 0;
		display: grid;
		place-items: center;
		pointer-events: none;
	}
	.spinner {
		width: 2rem;
		height: 2rem;
		border: 3px solid transparent;
		border-top-color: var(--primary);
		border-right-color: var(--primary);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.spinner {
			animation: none;
		}
	}
</style>
