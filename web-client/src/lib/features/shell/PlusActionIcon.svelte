<script lang="ts">
	import Icon from './Icon.svelte';
	import PlusTierIcon from './PlusTierIcon.svelte';
	let { name, locked = false }: { name: string; locked?: boolean } = $props();
</script>

<span class="plus-action-icon" aria-hidden="true">
	<Icon {name} />
	{#if locked}<span class="badge" data-plus-badge
			><PlusTierIcon size={14} /></span
		>{/if}
</span>

<style>
	.plus-action-icon {
		position: relative;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		flex-shrink: 0;
	}
	.badge {
		position: absolute;
		inset-inline-end: -4px;
		top: -4px;
		display: flex;
		color: var(--primary);
		background: var(--surface);
		border-radius: 50%;
		padding: 1px;
	}
</style>
