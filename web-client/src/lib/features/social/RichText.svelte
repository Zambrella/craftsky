<script lang="ts">
	import { confirmExternalLink } from './prompt';
	import { facetParts } from './model';
	import { plainTextParts } from './plain-text';
	import type { Facet } from '#lib/api/types.ts';
	let {
		text,
		facets,
		detectPlainText = false
	}: {
		text: string;
		facets?: Facet[] | null;
		detectPlainText?: boolean;
	} = $props();
	const parts = $derived(
		detectPlainText ? plainTextParts(text) : facetParts(text, facets)
	);
</script>

{#each parts as part}{#if part.href}<a
			href={part.href}
			rel={part.external ? 'ugc nofollow noreferrer' : undefined}
			onclick={(event) => {
				if (part.external) confirmExternalLink(event);
			}}>{part.text}</a
		>{:else}{part.text}{/if}{/each}

<style>
	a {
		text-decoration: none;
	}

	a:hover {
		text-decoration: underline;
	}
</style>
