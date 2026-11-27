import { describe, expect, it } from 'vitest';
import { identityCrafts, fittingIdentityCrafts } from './compact-identity';

describe('native compact craft identity', () => {
	it('renders profiles without crafts from search responses', () => {
		expect(identityCrafts(null)).toEqual([]);
		expect(identityCrafts(undefined)).toEqual([]);
		expect(identityCrafts([])).toEqual([]);
	});
	it('normalizes lexicon IDs without changing profile order or dropping unknown crafts', () => {
		expect(
			identityCrafts([
				'social.craftsky.feed.defs#Knitting',
				' knitting ',
				'',
				'social.craftsky.feed.defs#punchNeedle',
				'punch-needle',
				'pottery',
				'sewing'
			])
		).toEqual([
			{
				id: 'knitting',
				name: 'Knitting',
				icon: "url('/craft_icons/knitting.svg')"
			},
			{
				id: 'punch-needle',
				name: 'Punch Needle',
				icon: "url('/craft_icons/punch-needle.svg')"
			},
			{ id: 'pottery', name: 'Pottery', icon: undefined },
			{
				id: 'sewing',
				name: 'Sewing',
				icon: "url('/craft_icons/sewing.svg')"
			}
		]);
	});
	it('fits all known crafts when there is space', () => {
		expect(
			fittingIdentityCrafts(
				240,
				80,
				identityCrafts(['knitting', 'sewing']),
				() => 14
			)
		).toBe(2);
	});
	it('prioritizes a long display name and keeps the overflow count', () => {
		expect(
			fittingIdentityCrafts(
				120,
				300,
				identityCrafts(['knitting', 'sewing']),
				() => 14
			)
		).toBe(0);
	});
	it('reserves the unknown craft count when choosing the visible glyphs', () => {
		expect(
			fittingIdentityCrafts(
				130,
				80,
				identityCrafts(['knitting', 'sewing', 'pottery']),
				() => 14
			)
		).toBe(1);
	});
});
