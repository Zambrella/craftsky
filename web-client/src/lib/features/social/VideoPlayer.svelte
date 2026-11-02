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
		warning = $state('');
	let captions = $state<{ url: string; lang: string }[]>([]);
	let hls: { destroy: () => void } | undefined;
	let dead = false,
		captionsStarted = false;
	const abort = new AbortController();
	onMount(() => {
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
				loading = false;
				instance.startLoad();
				void player.play().catch(() => {
					if (!dead) warning = 'Press play in the video controls to start.';
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

<video
	bind:this={player}
	controls
	preload="none"
	poster={safeLink(video.thumbnail || '')}
	aria-label={video.alt || 'Post video'}
	onplay={() => void loadCaptions()}
	>{#if native}<source
			src={safeLink(video.playlist)}
			type="application/vnd.apple.mpegurl"
		/>{/if}{#each captions as caption}<track
			kind="captions"
			src={caption.url}
			srclang={caption.lang}
			label={caption.lang}
		/>{/each}</video
>{#if !native && !ready}<button disabled={loading} onclick={start}
		>{loading ? 'Loading video…' : 'Play video'}</button
	>{/if}{#if warning}<p role="status">
		{warning}
		{#if safeLink(video.playlist)}<a
				href={safeLink(video.playlist)}
				rel="noreferrer">Open video</a
			>{/if}
	</p>{/if}
