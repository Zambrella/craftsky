<script lang="ts">
	import Icon from '../shell/Icon.svelte';
	let { url }: { url?: string } = $props();
	let failed = $state('');
	const source = $derived(
		url &&
			(/^https:\/\//i.test(url) ||
				/^data:image\/(?:png|jpeg|gif|webp|svg\+xml);base64,/i.test(url))
			? url
			: ''
	);
</script>

{#if source && failed !== source}<img
		src={source}
		width="18"
		height="18"
		alt=""
		onerror={() => (failed = source)}
	/>{:else}<Icon name="globe" size={18} />{/if}

<style>
	img {
		width: 18px;
		height: 18px;
		object-fit: contain;
		flex: none;
	}
</style>
