<script lang="ts">
	import type { DraftMedia } from '../drafts/store';
	let { media }: { media: DraftMedia } = $props();
	let url = $state('');
	const imageBlob = $derived(media.blob);
	const remoteUrl = $derived(media.remoteUrl);
	$effect(() => {
		const blob = imageBlob;
		const current = blob ? URL.createObjectURL(blob) : remoteUrl || '';
		url = current;
		return () => {
			if (blob) URL.revokeObjectURL(current);
		};
	});
</script>

<img src={url} alt="" />

<style>
	img {
		width: 64px;
		height: 64px;
		object-fit: cover;
		border: 1px solid var(--ink);
	}
</style>
