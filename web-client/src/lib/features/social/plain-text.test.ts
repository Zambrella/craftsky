import { describe, expect, it } from 'vitest';
import { plainTextParts } from './plain-text';

describe('profile bio rich text', () => {
	it('links handles, Unicode hashtags and bare or explicit HTTPS URLs while preserving text', () => {
		const text =
			'🧶 @alice.test makes #編織. Visit example.org or https://example.org/yarn.';
		const parts = plainTextParts(text);
		expect(parts.map((part) => part.text).join('')).toBe(text);
		expect(parts.filter((part) => part.href)).toEqual([
			{ text: '@alice.test', href: '/profiles/alice.test', external: false },
			{
				text: '#編織',
				href: '/search/tags?tag=%E7%B7%A8%E7%B9%94',
				external: false
			},
			{ text: 'example.org', href: 'https://example.org/', external: true },
			{
				text: 'https://example.org/yarn',
				href: 'https://example.org/yarn',
				external: true
			}
		]);
	});
	it('keeps URL fragments inside their link and trims only unmatched closing punctuation', () => {
		const text = '[example.org/yarn_(blue)#colours] #knit';
		expect(plainTextParts(text).filter((part) => part.href)).toEqual([
			{
				text: 'example.org/yarn_(blue)#colours',
				href: 'https://example.org/yarn_(blue)#colours',
				external: true
			},
			{ text: '#knit', href: '/search/tags?tag=knit', external: false }
		]);
	});
	it('leaves email addresses, unsafe schemes, overlong hashtags and plain text alone', () => {
		for (const text of [
			'',
			'Hello\n🧶 there',
			'mail@example.org',
			'javascript:example.org',
			'https://user:secret@example.org/',
			'#' + 'a'.repeat(65)
		]) {
			expect(plainTextParts(text)).toEqual([{ text }]);
		}
	});
});
