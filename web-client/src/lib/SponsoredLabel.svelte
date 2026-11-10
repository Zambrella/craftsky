<script lang="ts">
	import Icon from './features/shell/Icon.svelte';
	const explanation =
		'The creator marked this post as sponsored because they received money, products, or another benefit.';
	const descriptionId = $props.id();
</script>

<span
	class="sponsored-label"
	title={explanation}
	aria-describedby={descriptionId}
>
	<Icon name="info" size={14} />Sponsored
</span>
<span id={descriptionId} class="visually-hidden">{explanation}</span>

<style>
	.sponsored-label {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		min-height: 28px;
		color: var(--ink-3);
		font-size: 12px;
		font-weight: 600;
		line-height: 1.4;
	}
</style>
