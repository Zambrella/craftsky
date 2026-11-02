<script lang="ts">
	import { facetParts } from './model';
	import type { Facet } from '#lib/api/types.ts';
	let { text, facets }: { text: string; facets?: Facet[] | null } = $props();
</script>

{#each facetParts(text, facets) as part}{#if part.href}<a
			href={part.href}
			rel={part.external ? 'ugc nofollow noreferrer' : undefined}
			onclick={(event) => {
				if (part.external && !confirm('Open this external website?'))
					event.preventDefault();
			}}>{part.text}</a
		>{:else}{part.text}{/if}{/each}
