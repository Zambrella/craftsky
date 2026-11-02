import { describe, expect, it } from 'vitest';
import { activeFacetToken, replaceFacetToken } from './facet-input.ts';
import { projectFacets } from './model.ts';

describe('inline facet editing', () => {
	it('detects a partial mention at the caret and replaces the whole token without losing suffix text', () => {
		const text = '🧶 Hi @alicesuffix after';
		const caret = text.indexOf('@') + 3;
		const token = activeFacetToken(text, caret)!;
		expect(token).toEqual({
			kind: 'mention',
			start: text.indexOf('@'),
			end: text.indexOf(' after'),
			query: 'al'
		});
		const result = replaceFacetToken(text, token, 'alice.test');
		expect(result.text).toBe('🧶 Hi @alice.test after');
		expect(result.caret).toBe(result.text.indexOf(' after'));
	});
	it('rejects email addresses, URLs, invalid partial handles and noncollapsed selections', () => {
		for (const text of [
			'mail@example',
			'https://example.com/#knit',
			'Hi @bad!',
			'word#tag'
		])
			expect(activeFacetToken(text, text.length)).toBeUndefined();
		expect(activeFacetToken('@alice', 2, 5)).toBeUndefined();
	});
	it('supports Unicode hashtags and adds a separating space only at the end', () => {
		const text = 'Try #編織';
		const token = activeFacetToken(text, text.length)!;
		expect(token.kind).toBe('hashtag');
		expect(token.query).toBe('編織');
		expect(replaceFacetToken(text, token, '編織作品')).toEqual({
			text: 'Try #編織作品 ',
			caret: 'Try #編織作品 '.length
		});
		expect(() => replaceFacetToken(text, token, 'not a token')).toThrow(
			'Invalid suggestion'
		);
	});
	it('generates annotations for project pattern/material fields without changing unedited authored annotations', async () => {
		const project = {
			common: {
				craftType: 'social.craftsky.feed.defs#knitting',
				pattern: { name: '#Socks', designer: '@alice.test' },
				materials: [{ text: 'Yarn from @alice.test' }]
			}
		};
		const result = await projectFacets(
			project,
			undefined,
			async () => 'did:plc:alice'
		);
		const common = result.common as Record<string, unknown>;
		const pattern = common.pattern as Record<string, unknown>;
		expect(pattern.nameFacets).toEqual([
			{
				index: { byteStart: 0, byteEnd: 6 },
				features: [{ $type: 'app.bsky.richtext.facet#tag', tag: 'Socks' }]
			}
		]);
		expect(pattern.designerFacets).toEqual([
			{
				index: { byteStart: 0, byteEnd: 11 },
				features: [
					{ $type: 'app.bsky.richtext.facet#mention', did: 'did:plc:alice' }
				]
			}
		]);
		let resolved = 0;
		expect(
			await projectFacets(result, result, async () => {
				resolved++;
				return 'different-owner';
			})
		).toEqual(result);
		expect(resolved).toBe(0);
	});
});
