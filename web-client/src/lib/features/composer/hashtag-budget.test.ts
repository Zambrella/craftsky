import { expect, it } from 'vitest';
import { acceptedHashtags, facetSegments, hashtagsInText } from './facet-input';
import { buildFacets, projectFacets } from './model';
import type { JsonObject } from '#lib/api/client.ts';

const firstNine = Array.from({ length: 9 }, (_, i) => `tag${i}`);
const caption = firstNine.map((tag) => `#${tag}`).join(' ');
const resolve = async () => 'did:plc:alice';
const tags = (facets: unknown): string[] =>
	(facets as { features: { $type: string; tag: string }[] }[]).flatMap(
		(facet) =>
			facet.features
				.filter((feature) => feature.$type.endsWith('#tag'))
				.map((feature) => feature.tag)
	);

it('paints and publishes only ten distinct tags, preserving mentions and links', async () => {
	const text = `${caption} #Tenth #overflow #TAG0 @alice.test https://example.com`;
	const facets = await buildFacets(text, resolve);
	expect(tags(facets)).toEqual([...firstNine, 'Tenth', 'TAG0']);
	expect(facets).toHaveLength(13);
	expect(
		facetSegments(text).find((part) => part.text === '#overflow')?.facet
	).toBe(false);
	expect(facetSegments(text).find((part) => part.text === '#TAG0')?.facet).toBe(
		true
	);
	expect(
		acceptedHashtags([...firstNine, ' TAG0 ', 'tenth', 'overflow']).size
	).toBe(10);
	expect(
		facetSegments('#tenth #overflow', firstNine).map((part) => part.facet)
	).toEqual([true, false, false]);
});

it('shares the budget across structured tags, pattern fields and materials and refreshes unchanged leaves', async () => {
	const project: JsonObject = {
		common: {
			craftType: 'social.craftsky.feed.defs#sewing',
			tags: ['TAG0', 'structured'],
			pattern: { name: '#pattern' },
			materials: [{ text: '#material @alice.test' }]
		}
	};
	const result = await projectFacets(
		project,
		undefined,
		resolve,
		hashtagsInText(caption)
	);
	const common = result.common as JsonObject;
	expect(common.tags).toEqual(['TAG0', 'structured']);
	expect((common.pattern as JsonObject).nameFacets).toBeUndefined();
	expect(tags((common.materials as JsonObject[])[0].facets)).toEqual([]);
	const updated = await projectFacets(
		result,
		result,
		async () => {
			throw new Error('unchanged mentions must not resolve');
		},
		[]
	);
	const next = updated.common as JsonObject;
	expect(tags((next.pattern as JsonObject).nameFacets)).toEqual(['pattern']);
	expect(tags((next.materials as JsonObject[])[0].facets)).toEqual([
		'material'
	]);
	expect(JSON.stringify(project)).toContain('#pattern');
});
