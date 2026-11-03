<script lang="ts">
	import { textLength } from './model';
	let { value, maximum }: { value: string; maximum: number } = $props();
	const count = $derived(textLength(value));
</script>

<span class="field-counter" class:over={count > maximum} aria-hidden="true"
	>{count}/{maximum}</span
>

<style>
	.field-counter {
		display: block;
		text-align: right;
		font-size: 14px;
		line-height: 1.45;
		font-weight: 400;
		color: var(--ink-2);
		padding-top: 4px;
	}
	.over {
		color: var(--red-ink);
	}
</style>
