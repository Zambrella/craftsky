<script lang="ts">
	import Icon from './Icon.svelte';
	import PlusTierIcon from './PlusTierIcon.svelte';
	let { name, locked = false }: { name: string; locked?: boolean } = $props();
</script>

<span class="plus-action-icon" aria-hidden="true">
	<Icon {name} />
	{#if locked}<span class="plus-action-mark" data-plus-badge
			><PlusTierIcon size={12} /></span
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
	.plus-action-mark {
		position: absolute;
		inset-inline-end: -4px;
		bottom: -4px;
		display: flex;
		color: var(--primary);
		width: 12px;
		height: 12px;
	}
	/* Composer icons have a shared size; the tier mark stays a small overlay. */
	.plus-action-mark :global(svg) {
		width: 100%;
		height: 100%;
	}
</style>
