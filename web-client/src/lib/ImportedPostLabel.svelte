<script lang="ts">
	let { source }: { source?: string } = $props();
</script>

{#if source === 'instagram'}
	<small class="import-label">Imported from Instagram</small>
{/if}

<style>
	.import-label {
		display: block;
		margin-block: 8px;
		color: var(--ink-3);
		font-size: 12px;
		font-weight: 600;
		line-height: 1.2;
		letter-spacing: 1.68px;
	}
</style>
