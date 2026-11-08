<script lang="ts">
	import { onMount } from 'svelte';
	import { loadYouTubeAPI, type YouTubeExternal } from './youtube';
	let {
		video,
		title,
		onerror
	}: { video: YouTubeExternal; title: string; onerror: () => void } = $props();
	let frame: HTMLIFrameElement;
	let src = $state('');
	onMount(() => {
		const url = new URL(
			'https://www.youtube-nocookie.com/embed/' + video.videoId
		);
		url.search = new URLSearchParams({
			autoplay: '1',
			playsinline: '1',
			rel: '0',
			enablejsapi: '1',
			origin: location.origin,
			start: String(video.startSeconds)
		}).toString();
		src = url.href;
		let disposed = false;
		let player: { destroy(): void } | undefined;
		void loadYouTubeAPI()
			.then((api) => {
				if (!disposed)
					player = new api.Player(frame, { events: { onError: onerror } });
			})
			.catch(() => {
				if (!disposed) onerror();
			});
		return () => {
			disposed = true;
			player?.destroy();
		};
	});
</script>

<iframe
	bind:this={frame}
	{src}
	title={'YouTube video player: ' + title}
	style:aspect-ratio={video.isShort ? '9 / 16' : '16 / 9'}
	allow="autoplay; encrypted-media; picture-in-picture; fullscreen"
	allowfullscreen
	referrerpolicy="strict-origin-when-cross-origin"
	{onerror}
></iframe>

<style>
	iframe {
		display: block;
		width: 100%;
		border: 0;
		background: black;
	}
</style>
