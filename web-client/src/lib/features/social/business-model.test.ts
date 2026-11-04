import { describe, it, expect } from 'vitest';
import { businessAction, eventDisplay } from './business-model';
describe('published business actions', () => {
	it('formats native all-day ranges with an inclusive final day and handles invalid timestamps', () => {
		const start = new Date(2026, 10, 1),
			end = new Date(2026, 10, 3);
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
