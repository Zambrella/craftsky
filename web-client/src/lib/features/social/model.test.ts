import { describe, it, expect } from 'vitest';
import {
	facetParts,
	postRoute,
	profileStyle,
	texture,
	postAddress,
	safeLink
} from './model';
import { recentDestination } from './search-model';
describe('social presentation boundaries', () => {
	it('preserves unannotated Unicode and empty text', () => {
		for (const text of ['', '🧶 é plain text']) {
			for (const facets of [undefined, null, []]) {
				expect(facetParts(text, facets)).toEqual([{ text }]);
			}
		}
	});
	it('resolves UTF8 facets only on valid text boundaries and rejects overlapping ranges', () => {
		const text = '🧶 Hi Alice!';
		expect(
			facetParts(text, [
				{
					index: { byteStart: 8, byteEnd: 13 },
					features: [
						{ $type: 'app.bsky.richtext.facet#mention', did: 'did:plc:alice' }
					]
				}
			])
		).toEqual([
			{ text: '🧶 Hi ' },
			{
				text: 'Alice',
				href: '/profiles/did%3Aplc%3Aalice',
				external: false
			},
			{ text: '!' }
		]);
		expect(
			facetParts('🧶', [
				{
					index: { byteStart: 1, byteEnd: 3 },
					features: [{ $type: 'x#link', uri: 'https://example.org' }]
				}
			])
		).toEqual([{ text: '🧶' }]);
		expect(
			facetParts('hello', [
				{
					index: { byteStart: 0, byteEnd: 5 },
					features: [{ $type: 'x#link', uri: 'javascript:alert(1)' }]
				}
			])
		).toEqual([{ text: 'hello' }]);
	});
	it('constrains externally supplied URLs and profile style values', () => {
		for (const url of [
			'javascript:alert(1)',
			'data:text/html,<script>',
			'https://user:password@example.org/',
			'http://example.org/'
		])
			expect(safeLink(url)).toBeUndefined();
		expect(
			profileStyle({
				colour: 'red; background:url(https://evil.test)',
				profileBackground: '../../etc/passwd'
			})
		).not.toContain('evil');
		expect(
			texture({ colour: 'cobalt', profileBackground: '../evil' })
		).toBeUndefined();
		expect(
			postAddress('at://did:plc:alice/social.craftsky.feed.post/rkey')
		).toEqual({ did: 'did:plc:alice', rkey: 'rkey' });
		expect(postAddress('https://evil.test/')).toBeUndefined();
	});
	it('restores typed recent searches with queries and filters', () => {
		expect(
			recentDestination({
				id: 'one',
				type: 'profile',
				displayLabel: 'Alice',
				payload: { did: 'did:plc:alice' }
			})
		).toBe('/profiles/did%3Aplc%3Aalice');
		const result = recentDestination({
			id: 'two',
			type: 'project',
			displayLabel: 'knitting',
			payload: {
				q: 'scarf',
				sort: 'popular',
				filters: { craftType: ['knitting', 'crochet'], selfDrafted: true }
			}
		});
		expect(result).toContain('craftType=knitting&craftType=crochet');
		expect(result).toContain('selfDrafted=true');
		expect(result).toContain('tab=projects');
	});
});

it('links compact quote views by canonical AT URI without a fabricated record key', () => {
	expect(
		postRoute({
			uri: 'at://did:plc:alice/social.craftsky.feed.post/quoted',
			author: { did: 'did:plc:alice', handle: 'alice.test' }
		})
	).toBe('/posts/did%3Aplc%3Aalice/quoted');
});

it('keeps published text readable when annotation metadata is malformed', () => {
	const bad = JSON.parse(
		'[null,{"features":[]},{"index":{"byteStart":0,"byteEnd":2},"features":[null]}]'
	);
	expect(facetParts('Hello', bad)).toEqual([{ text: 'Hello' }]);
	expect(
		profileStyle({ colour: '__proto__', profileBackground: 'none' })
	).toContain('#1535D6');
});
