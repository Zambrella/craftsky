<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { CraftskyAPI, Video } from '#lib/api/client.ts';
	import { safeLink } from './model';
	let {
		video,
		api,
		did,
		rkey
	}: { video: Video; api: CraftskyAPI; did: string; rkey: string } = $props();
	let player: HTMLVideoElement;
	let native = $state(false),
		ready = $state(false),
		loading = $state(false),
		playing = $state(false),
		warning = $state('');
	let captions = $state<{ url: string; lang: string }[]>([]);
	let hls: { destroy: () => void } | undefined;
	let dead = false,
		captionsStarted = false;
	const abort = new AbortController();
	onMount(() => {
		player.muted = false;
		player.volume = 1;
		native = !!player.canPlayType('application/vnd.apple.mpegurl');
	});
	async function loadCaptions() {
		if (captionsStarted) return;
		captionsStarted = true;
		for (const caption of video.captions || []) {
			try {
				const text = await api.media.caption(
					did,
					rkey,
					caption.cid,
					abort.signal
				);
				if (dead) return;
				captions = [
					...captions,
					{
						url: URL.createObjectURL(new Blob([text], { type: 'text/vtt' })),
						lang: caption.lang
					}
				];
			} catch {
				if (!dead) warning = 'Some captions could not be loaded.';
			}
		}
	}
	async function togglePlayback() {
		if (!native && !ready) {
			await start();
			return;
		}
		if (!player.paused) player.pause();
		else {
			warning = '';
			try {
				await player.play();
			} catch {
				if (!dead) {
					loading = false;
					warning = 'Video playback could not start. Please try again.';
				}
			}
		}
	}
	async function start() {
		if (loading) return;
		loading = true;
		warning = '';
		hls?.destroy();
		try {
			const { default: Hls } = await import('hls.js');
			if (dead) return;
			if (!Hls.isSupported()) throw new Error();
			const instance = new Hls({ enableWorker: false, autoStartLoad: false });
			hls = instance;
			instance.on(Hls.Events.MANIFEST_PARSED, () => {
				if (dead || hls !== instance) return;
				ready = true;
				instance.startLoad();
				void player.play().catch(() => {
					if (!dead) {
						loading = false;
						warning = 'Press play in the video controls to start.';
					}
				});
			});
			instance.on(Hls.Events.ERROR, (_event, data) => {
				if (!dead && hls === instance && data.fatal) {
					ready = false;
					loading = false;
					warning = 'Video could not be loaded. Please try again.';
					instance.destroy();
				}
			});
			const url = safeLink(video.playlist);
			if (!url) throw new Error();
			instance.loadSource(url);
			instance.attachMedia(player);
		} catch {
			if (!dead) {
				loading = false;
				warning =
					'Video playback could not start. Please try again or open the video link.';
			}
		}
	}
	onDestroy(() => {
		dead = true;
		abort.abort();
		hls?.destroy();
		for (const caption of captions) URL.revokeObjectURL(caption.url);
	});
</script>

<div class="video-player">
	<video
		bind:this={player}
		controls
		preload="none"
		poster={safeLink(video.thumbnail || '')}
		aria-label={video.alt || 'Post video'}
		onplay={() => {
			playing = true;
			void loadCaptions();
		}}
		onplaying={() => {
			playing = true;
			loading = false;
		}}
		onpause={() => {
			playing = false;
			loading = false;
		}}
		onended={() => {
			playing = false;
			loading = false;
		}}
		onwaiting={() => {
			if (!player.paused) loading = true;
		}}
		oncanplay={() => {
			loading = false;
		}}
		onloadstart={() => {
			if (!player.paused) loading = true;
		}}
		onerror={() => {
			if (native) {
				loading = false;
				warning = 'Video could not be loaded. Please try again.';
			}
		}}
		>{#if native}<source
				src={safeLink(video.playlist)}
				type="application/vnd.apple.mpegurl"
			/>{/if}{#each captions as caption}<track
				kind="captions"
				src={caption.url}
				srclang={caption.lang}
				label={caption.lang}
			/>{/each}</video
	>
	<button
		class="video-toggle"
		aria-label={playing ? 'Pause video' : 'Play video'}
		disabled={loading && !native && !ready}
		onclick={togglePlayback}
		>{#if !playing && !loading}<span class="play-label">Play video</span
			>{/if}</button
	>
	{#if loading}<span
			class="video-loader"
			role="status"
			aria-label="Loading video"><span class="spinner"></span></span
		>{/if}
</div>
{#if warning}<p role="status">
		{warning}
		{#if safeLink(video.playlist)}<a
				href={safeLink(video.playlist)}
				rel="noreferrer">Open video</a
			>{/if}
	</p>{/if}

<style>
	.video-player {
		position: relative;
	}
	video {
		display: block;
		width: 100%;
	}
	/* The picture is clickable; the browser's transport controls remain exposed. */
	.video-toggle {
		position: absolute;
		inset: 0 0 3rem;
		width: 100%;
		border: 0;
		border-radius: 0;
		background: transparent;
		display: grid;
		place-items: center;
		cursor: pointer;
		padding: 0;
	}
	.video-toggle:hover,
	.video-toggle:disabled {
		background: transparent;
		opacity: 1;
	}
	.video-toggle:focus-visible {
		outline: 3px solid var(--primary);
		outline-offset: -3px;
	}
	.play-label {
		background: rgb(0 0 0 / 75%);
		color: white;
		padding: 0.65rem 1rem;
		border-radius: 2rem;
	}
	.video-loader {
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
		animation: video-spin 0.8s linear infinite;
	}
	video::-webkit-media-controls-loading-panel {
		display: none;
	}
	@keyframes video-spin {
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
