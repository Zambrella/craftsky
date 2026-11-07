<script lang="ts">
	import type { CraftskyAPI, Video } from '#lib/api/client.ts';
	import HlsVideoPlayer from './HlsVideoPlayer.svelte';
	import BunnyVideoPlayer from './BunnyVideoPlayer.svelte';
	import { bunnyEmbedURL } from './bunny-player.ts';
	let {
		video,
		api,
		did,
		rkey
	}: { video: Video; api: CraftskyAPI; did: string; rkey: string } = $props();
	let fallbackFor = $state('');
	const embed = $derived(bunnyEmbedURL(video));
</script>

<div class="post-video">
	{#if embed && fallbackFor !== embed}
		{#key embed}<BunnyVideoPlayer
				{video}
				{embed}
				onfallback={() => {
					fallbackFor = embed;
				}}
			/>{/key}
	{:else}<HlsVideoPlayer {video} {api} {did} {rkey} />{/if}
</div>

<style>
	.post-video {
		margin-top: 8px;
	}
</style>
