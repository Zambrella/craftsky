import { describe, expect, it } from 'vitest';
import { collapsedBody } from './body';

describe('native feed body collapsing', () => {
	it('keeps short bodies whole and counts joined emoji as single graphemes', () => {
		const emoji = '👨‍👩‍👧‍👦';
		expect(collapsedBody(emoji.repeat(300))).toEqual({
			text: emoji.repeat(300),
			collapsible: false
		});
		expect(collapsedBody(emoji.repeat(301))).toEqual({
			text: emoji.repeat(300),
			collapsible: true
		});
	});
	it('moves a crossing UTF-8 facet out whole and retains links ending at the boundary', () => {
		const prefix = 'é'.repeat(295),
			link = 'https://example.com/pattern';
		const facets = [
			{
				index: {
					byteStart: new TextEncoder().encode(prefix).length,
					byteEnd: new TextEncoder().encode(prefix + link).length
				},
				features: [{ $type: 'app.bsky.richtext.facet#link', uri: link }]
			}
		];
		expect(collapsedBody(prefix + link + ' more', facets).text).toBe(prefix);
		expect(
			collapsedBody(prefix + link + ' more', facets, 295 + link.length).text
		).toBe(prefix + link);
	});
	it('ignores invalid facet offsets and never splits a combining character', () => {
		const text = 'é'.repeat(301);
		expect(
			collapsedBody(text, [
				{
					index: { byteStart: 2, byteEnd: 5 },
					features: [
						{
							$type: 'app.bsky.richtext.facet#link',
							uri: 'javascript:alert(1)'
						}
					]
				}
			]).text
		).toBe('é'.repeat(300));
	});
});
