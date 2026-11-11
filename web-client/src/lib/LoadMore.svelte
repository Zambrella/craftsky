<script lang="ts">
	import StitchSpinner from './StitchSpinner.svelte';
	let {
		loading = false,
		disabled = false,
		onclick
	}: { loading?: boolean; disabled?: boolean; onclick: () => void } = $props();
</script>

<div class="load-more">
	{#if loading}<StitchSpinner label="Loading more accounts" />{:else}<button
			type="button"
			{disabled}
			{onclick}>Load more</button
		>{/if}
</div>

<style>
	.load-more {
		display: flex;
		justify-content: center;
		align-items: center;
		min-height: 80px;
		padding: 16px;
	}
	.load-more button {
		border: 0;
		background: transparent;
		box-shadow: none;
		color: var(--primary);
		padding: 8px 16px;
		min-height: 48px;
		border-radius: 999px;
		font:
			600 16px/1.5 Outfit,
			sans-serif;
	}
	.load-more button:hover:not(:disabled) {
		background: color-mix(in srgb, var(--primary) 8%, transparent);
	}
</style>
