import { describe, it, expect } from 'vitest';
import { safeURL, description, label, postPath } from './public';
describe('public content boundaries', () => {
	it.each([
		'javascript:alert(1)',
		'data:text/html,test',
		'http://example.com',
		'https://name:secret@example.com',
		'/relative',
		'not a URL'
	])('rejects unsafe URL %s', (url) => expect(safeURL(url)).toBeUndefined());
	it('accepts HTTPS links', () =>
		expect(safeURL('https://example.com/pattern')).toBe(
			'https://example.com/pattern'
		));
	it('bounds metadata by code points and collapses whitespace', () => {
		expect(description('hello\n  world')).toBe('hello world');
		expect(Array.from(description('🧶'.repeat(250)))).toHaveLength(200);
	});
	it('formats protocol tokens for people', () => {
		expect(label('social.craftsky.feed.defs#wip')).toBe('Work in progress');
		expect(label('homeGoods')).toBe('Home goods');
	});
	it('encodes path segments rather than interpreting them', () =>
		expect(
			postPath({ author: { did: 'did:plc:a' }, rkey: 'a/b' } as never)
		).toBe('/posts/did%3Aplc%3Aa/a%2Fb'));
});

it('renders structured gauge without exposing unfamiliar fields', async () => {
	const { gaugeText } = await import('./public');
	expect(
		gaugeText({
			stitches: 20,
			rows: 28,
			measurement: 10,
			unit: 'cm',
			internal: 'ignored'
		})
	).toBe('20 stitches × 28 rows over 10 cm');
	expect(
		gaugeText({ stitches: 'bad', measurement: 10, unit: 'cm' })
	).toBeUndefined();
});

it('matches Flutter compact dates including the seven-day boundary', async () => {
	const { compactDate } = await import('./public');
	const when = '2026-10-01T12:00:00Z',
		start = Date.parse(when);
	for (const [delta, label] of [
		[-1000, 'now'],
		[59000, 'now'],
		[3 * 60000, '3m'],
		[4 * 3600000, '4h'],
		[2 * 86400000, '2d'],
		[7 * 86400000, '7d'],
		[7 * 86400000 + 1, '1 Oct 26']
	] as const) {
		expect(compactDate(when, start + delta)).toBe(label);
	}
});
