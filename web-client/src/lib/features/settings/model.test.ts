import { describe, it, expect } from 'vitest';
import {
	accountScope,
	growthSegments,
	parseFollowing,
	languagePreferences,
	productBody,
	zonedTimestamp,
	wallTime,
	type Product
} from './model.ts';

describe('Instagram following imports', () => {
	it('normalizes usernames without silently treating links as accounts', () => {
		expect(
			parseFollowing(
				' @Maker.One\nmaker.one\nMAKER_TWO\nhttps://instagram.com/unsafe\ninvalid name'
			)
		).toEqual({
			entries: [{ username: 'maker.one' }, { username: 'maker_two' }],
			duplicates: 1,
			ignored: 2
		});
	});
	it('supports the legacy and current Instagram following exports', () => {
		const data = {
			relationships_following: [
				{ string_list_data: [{ value: 'Maker.One' }] },
				{
					title: 'MakerTwo',
					string_list_data: [{ href: 'https://www.instagram.com/_u/MakerTwo' }]
				}
			]
		};
		expect(parseFollowing(JSON.stringify(data), true).entries).toEqual([
			{ username: 'maker.one' },
			{ username: 'makertwo' }
		]);
	});
	it('requires an exact, matching Instagram profile URL for the current export', () => {
		const entries = [
			'https://www.instagram.com/_u/maker?tracking=1',
			'https://www.instagram.com.evil.test/_u/maker',
			'https://user@www.instagram.com/_u/maker',
			'https://www.instagram.com/_u/other'
		].map((href) => ({ title: 'maker', string_list_data: [{ href }] }));
		expect(
			parseFollowing(JSON.stringify({ relationships_following: entries }), true)
		).toEqual({ entries: [], ignored: 4, duplicates: 0 });
	});
	it('rejects unrelated exports and enforces the server entry bound', () => {
		expect(() => parseFollowing('[{"username":"maker"}]', true)).toThrow(
			'following.json'
		);
		expect(() =>
			parseFollowing(
				Array.from({ length: 10001 }, (_, i) => `maker${i}`).join('\n')
			)
		).toThrow('10,000');
	});
});
describe('follower growth', () => {
	it('leaves missing daily snapshots as gaps rather than connecting or zero-filling them', () => {
		const segments = growthSegments([
			{ date: '1', count: 8 },
			{ date: '2', count: 9 },
			{ date: '3', count: null },
			{ date: '4', count: 7 }
		]);
		expect(segments).toHaveLength(2);
		expect(segments[0]).toContain(' L');
		expect(segments[1]).not.toContain(' L');
	});
	it('handles absent history and a constant count without invalid SVG coordinates', () => {
		expect(growthSegments([{ date: '1', count: null }])).toEqual([]);
		expect(
			growthSegments([
				{ date: '1', count: 0 },
				{ date: '2', count: 0 }
			]).join('')
		).not.toMatch(/NaN|Infinity/);
	});
});
describe('account-bound settings state', () => {
	it('invalidates completions after the account or route changes', () => {
		const scope = accountScope(),
			token = scope.token();
		expect(scope.current(token)).toBe(true);
		scope.invalidate();
		expect(scope.current(token)).toBe(false);
	});
});
describe('language settings', () => {
	it('matches the AppView supported base-language catalogue and deduplicates selection', () =>
		expect(languagePreferences('PT', 'en, cy, pt, en')).toEqual({
			primaryLanguage: 'pt',
			contentLanguages: ['en', 'cy', 'pt']
		}));
	it('rejects unsupported regional tags before replacing server state', () =>
		expect(() => languagePreferences('en', 'en,pt-BR')).toThrow(
			'supported languages'
		));
});
describe('business products', () => {
	const product: Product = {
		title: 'Hand dyed wool',
		uri: 'https://shop.example/wool',
		amount: '12.50',
		currency: 'gbp',
		image: {
			cid: 'bafyimage',
			mime: 'image/jpeg',
			size: 100,
			alt: 'Blue wool',
			aspectRatio: { width: 100, height: 200 }
		}
	};
	it('converts image views back to blob references and canonical decimal prices', () => {
		expect(productBody([product])).toEqual([
			{
				title: 'Hand dyed wool',
				uri: 'https://shop.example/wool',
				image: {
					image: {
						$type: 'blob',
						ref: { $link: 'bafyimage' },
						mimeType: 'image/jpeg',
						size: 100
					},
					alt: 'Blue wool',
					aspectRatio: { width: 100, height: 200 }
				},
				price: { amount: '12.5', currency: 'GBP' }
			}
		]);
	});
	it('rejects unsafe and duplicate destinations and incomplete prices', () => {
		expect(() =>
			productBody([{ ...product, uri: 'javascript:alert(1)' }])
		).toThrow('HTTPS');
		expect(() => productBody([product, product])).toThrow(
			'different destination'
		);
		expect(() => productBody([{ ...product, currency: '' }])).toThrow('both');
		expect(() => productBody([{ ...product, image: undefined }])).toThrow(
			'image'
		);
	});
});
describe('business event time zones', () => {
	it('resolves an explicit IANA timezone independently of the browser timezone', () => {
		expect(zonedTimestamp('2026-07-02T12:30', 'Europe/London')).toBe(
			'2026-07-02T11:30:00.000Z'
		);
		expect(wallTime('2026-07-02T11:30:00Z', 'Europe/London')).toBe(
			'2026-07-02T12:30'
		);
	});
	it('rejects a nonexistent local time during the spring DST transition', () =>
		expect(() => zonedTimestamp('2026-03-29T01:30', 'Europe/London')).toThrow(
			'does not exist'
		));
	it('rejects calendar overflow instead of silently moving the event date', () =>
		expect(() => zonedTimestamp('2026-02-31T12:00', 'UTC')).toThrow(
			'valid date'
		));
});
