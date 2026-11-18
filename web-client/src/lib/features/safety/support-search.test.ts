import { describe, it, expect } from 'vitest';
import { shouldOfferSearchSupport } from './support-search';
import guide from './guide-data.json';
describe('optional support signposting', () => {
	it('matches explicit terms and separator variants', () => {
		for (const q of [
			'Suicidal feelings',
			'self-harm',
			'#self_harm',
			'self–injury',
			'eating disorders',
			'ANOREXIA',
			'help with bulimia'
		])
			expect(shouldOfferSearchSupport(q), q).toBe(true);
	});
	it('leaves neutral terms, acronyms and word fragments alone', () => {
		for (const q of [
			'',
			'ED',
			'size',
			'fit',
			'diet',
			'knitting',
			'suicideberry',
			'anorexias',
			'self harmony'
		])
			expect(shouldOfferSearchSupport(q), q).toBe(false);
	});
	it('only publishes fixed public-body HTTPS destinations without private routing', () => {
		for (const resource of guide.resources) {
			const url = new URL(resource.url);
			expect(url.protocol).toBe('https:');
			expect(url.search + url.hash + url.username + url.password).toBe('');
			expect(url.hostname).toMatch(
				/(?:\.gov\.uk|\.nhs\.uk|^www\.nhsinform\.scot)$/
			);
		}
	});
});
