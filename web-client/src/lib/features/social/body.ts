import { graphemeBoundary } from '#lib/text.ts';
import type { Facet } from '#lib/api/types.ts';
import { facetParts } from './model';

/** Flutter collapses feed bodies after 300 graphemes, moving a crossing link out whole. */
export function collapsedBody(
	text: string,
	facets?: Facet[] | null,
	limit = 300
) {
	let cutoff = graphemeBoundary(text, limit);
	if (cutoff === text.length) return { text, collapsible: false };
	let offset = 0;
	for (const part of facetParts(text, facets)) {
		if (part.href && offset < cutoff && offset + part.text.length > cutoff)
			cutoff = offset;
		offset += part.text.length;
	}
	return { text: text.slice(0, cutoff), collapsible: true };
}
