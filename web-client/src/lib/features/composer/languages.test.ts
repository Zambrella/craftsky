import { describe, expect, it } from 'vitest';
import {
	addLanguage,
	languageChoices,
	languageName,
	removeLanguage
} from './languages';
describe('post language picker', () => {
	it('searches names and codes without offering selected languages', () => {
		expect(languageChoices('  FRENCH ', ['en'])).toEqual([['fr', 'French']]);
		expect(languageChoices('en', ['en']).some(([tag]) => tag === 'en')).toBe(
			false
		);
	});
	it('retains one to three distinct supported languages', () => {
		expect(addLanguage(['en'], 'en')).toEqual(['en']);
		expect(addLanguage(['en'], 'fr')).toEqual(['en', 'fr']);
		expect(addLanguage(['en', 'fr', 'de'], 'es')).toEqual(['en', 'fr', 'de']);
		expect(addLanguage(['en'], '__proto__')).toEqual(['en']);
		expect(removeLanguage(['en'], 'en')).toEqual(['en']);
		expect(removeLanguage(['en', 'fr'], 'en')).toEqual(['fr']);
	});
	it('keeps an existing regional language visible without rewriting its tag', () => {
		expect(languageName('en-GB')).toBe('en-GB');
		expect(addLanguage(['en-GB'], 'fr')).toEqual(['en-GB', 'fr']);
	});
});
