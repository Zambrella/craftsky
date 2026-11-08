<script lang="ts">
	import { confirmExternalLink } from './prompt';
	import { facetParts } from './model';
	import type { Facet } from '#lib/api/types.ts';
	let { text, facets }: { text: string; facets?: Facet[] | null } = $props();
</script>

{#each facetParts(text, facets) as part}{#if part.href}<a
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
