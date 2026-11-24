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

it('shares the budget across structured tags, pattern fields and materials while preserving untouched patterns', async () => {
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
	expect(next.pattern).toEqual(common.pattern);
	expect(tags((next.materials as JsonObject[])[0].facets)).toEqual([
		'material'
	]);
	expect(JSON.stringify(project)).toContain('#pattern');
});

it('keeps legacy tagged pattern metadata when a caption edit fills the hashtag budget', async () => {
	const project: JsonObject = {
		common: {
			craftType: 'social.craftsky.feed.defs#sewing',
			pattern: {
				name: '#collar',
				nameFacets: [
					{
						index: { byteStart: 0, byteEnd: 7 },
						features: [{ $type: 'app.bsky.richtext.facet#tag', tag: 'collar' }]
					}
				],
				futureCredit: 'original input'
			}
		}
	};
	expect(
		await projectFacets(project, project, resolve, [...firstNine, 'tenth'])
	).toEqual(project);
});

it('counts eligible credit and later pattern tags in the same ten-tag budget', async () => {
	const result = await projectFacets(
		{
			common: {
				patterns: [
					{ designer: '@alice.test #credit', publisher: '#TAG0' },
					{ name: 'Later #later #overflow', designer: '@alice.test' }
				]
			}
		},
		undefined,
		resolve,
		firstNine.slice(0, 8)
	);
	const [first, later] = (result.common as JsonObject).patterns as JsonObject[];
	expect(tags(first.designerFacets)).toEqual(['credit']);
	expect(tags(first.publisherFacets)).toEqual(['TAG0']);
	expect(tags(later.nameFacets)).toEqual(['later']);
	expect(later.designerFacets).toEqual([
		{
			index: { byteStart: 0, byteEnd: 11 },
			features: [
				{ $type: 'app.bsky.richtext.facet#mention', did: 'did:plc:alice' }
			]
		}
	]);
});
