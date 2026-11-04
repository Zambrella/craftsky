<script lang="ts">
	import type { Facet } from '#lib/api/types.ts';
	import RichText from './RichText.svelte';
	import { collapsedBody } from './body';
	let {
		text,
		facets,
		collapse = true
	}: { text: string; facets?: Facet[] | null; collapse?: boolean } = $props();
	let expanded = $state(false);
	const preview = $derived(
		collapse ? collapsedBody(text, facets) : { text, collapsible: false }
	);
</script>

<span
	><RichText
		text={expanded ? text : preview.text}
		{facets}
	/>{#if preview.collapsible}{expanded ? ' ' : '… '}<button
			type="button"
			aria-expanded={expanded}
			onclick={() => (expanded = !expanded)}
			>{expanded ? 'Show less' : 'Show more'}</button
		>{/if}</span
>

<style>
	button {
		display: inline;
		min-height: 0;
		padding: 0;
		border: 0;
		border-radius: 0;
		background: transparent;
		box-shadow: none;
		font: inherit;
		font-weight: 600;
		color: var(--ink);
		vertical-align: baseline;
	}
	button:hover {
		text-decoration: underline;
	}
</style>
