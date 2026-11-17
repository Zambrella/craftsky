import { describe, it, expect } from 'vitest';
import { businessAction, eventDisplay } from './business-model';
describe('published business actions', () => {
	it('formats native all-day ranges with an inclusive final day and handles invalid timestamps', () => {
		const start = new Date('2026-11-01T00:00:00Z'),
			end = new Date('2026-11-03T00:00:00Z');
		const display = eventDisplay(
			{
				startsAt: start.toISOString(),
				endsAt: end.toISOString(),
				isAllDay: true
			},
			'en-GB'
		);
		expect(display).toEqual({
			date: '1 Nov 2026 – 2 Nov 2026',
			time: 'All day'
		});
		expect(
			eventDisplay({
				startsAt: 'invalid',
				endsAt: end.toISOString(),
				isAllDay: false
			})
		).toEqual({ date: 'Date unavailable', time: '' });
	});
	it('uses the event timezone rather than the viewer timezone for timed and overnight events', () => {
		expect(
			eventDisplay(
				{
					startsAt: '2026-11-02T04:30:00Z',
					endsAt: '2026-11-02T06:30:00Z',
					isAllDay: false,
					timeZone: 'America/New_York'
				},
				'en-GB'
			)
		).toEqual({
			date: '1 Nov 2026 – 2 Nov 2026',
			time: '23:30 – 1:30'
		});
	});
	it('keeps authored all-day dates through timezone offsets and daylight-saving transitions', () => {
		for (const [startsAt, endsAt, timeZone, date] of [
			[
				'2026-03-08T05:00:00Z',
				'2026-03-09T04:00:00Z',
				'America/New_York',
				'8 Mar 2026'
			],
			[
				'2026-10-31T15:00:00Z',
				'2026-11-02T15:00:00Z',
				'Asia/Tokyo',
				'1 Nov 2026 – 2 Nov 2026'
			],
			[
				'2026-11-01T04:00:00Z',
				'2026-11-02T05:00:00Z',
				'America/New_York',
				'1 Nov 2026'
			]
		]) {
			expect(
				eventDisplay({ startsAt, endsAt, isAllDay: true, timeZone }, 'en-GB')
			).toEqual({ date, time: 'All day' });
		}
	});
	it('uses UTC for missing or unavailable timezone identifiers', () => {
		for (const timeZone of [undefined, 'Unavailable/Zone'])
			expect(
				eventDisplay(
					{
						startsAt: '2026-11-01T10:00:00Z',
						endsAt: '2026-11-01T11:00:00Z',
						isAllDay: false,
						timeZone
					},
					'en-GB'
				)
			).toEqual({ date: '1 Nov 2026', time: '10:00 – 11:00' });
	});
	it('allows labelled HTTPS actions and a single header-free mail address', () => {
		expect(
			businessAction({
				type: 'shop',
				destination: 'https://maker.example/shop'
			})
		).toEqual({ href: 'https://maker.example/shop', label: 'Shop' });
		expect(
			businessAction({
				type: 'email',
				destination: 'mailto:hello@maker.example'
			})
		).toEqual({ href: 'mailto:hello@maker.example', label: 'Email' });
	});
	it('rejects unsupported actions, credential URLs, mail-header injection and extra recipients', () => {
		for (const value of [
			{ type: '__proto__', destination: 'https://example.org' },
			{ type: 'shop', destination: 'javascript:alert(1)' },
			{ type: 'shop', destination: 'https://user:password@example.org/' },
			{ type: 'email', destination: 'mailto:hello@example.org?body=private' },
			{
				type: 'email',
				destination: 'mailto:hello@example.org%0aBcc:other@example.org'
			},
			{ type: 'email', destination: 'mailto:a@example.org,b@example.org' }
		])
			expect(businessAction(value)).toBeUndefined();
	});
});
