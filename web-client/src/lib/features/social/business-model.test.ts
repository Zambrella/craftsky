import { describe, it, expect } from 'vitest';
import { businessAction } from './business-model';
describe('published business actions', () => {
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
