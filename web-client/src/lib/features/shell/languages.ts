import {
	APIError,
	type CraftskyAPI,
	type LanguagePreferences
} from '#lib/api/client.ts';
import catalogue from '../settings/languages.json';

/** Match Flutter's device-locale defaults using the shared supported base tags. */
export function deriveInitialLanguages(
	locales: readonly string[]
): LanguagePreferences {
	const content: string[] = [];
	for (const locale of locales) {
		try {
			const tag = new Intl.Locale(locale).language.toLowerCase();
			if (Object.hasOwn(catalogue, tag) && !content.includes(tag))
				content.push(tag);
		} catch {
			// Invalid or unsupported browser locales cannot become stored tags.
		}
	}
	const primaryLanguage =
		content.includes('en') || !content.length ? 'en' : content[0];
	if (!content.includes('en')) content.push('en');
	return { primaryLanguage, contentLanguages: content };
}

/** Missing preferences are initialized once; existing and concurrently saved choices win. */
export async function loadLanguagePreferences(
	api: Pick<CraftskyAPI, 'languages'>,
	locales: readonly string[]
): Promise<LanguagePreferences> {
	try {
		return await api.languages.get();
	} catch (error) {
		if (
			!(error instanceof APIError) ||
			error.status !== 404 ||
			error.code !== 'language_preferences_not_found'
		)
			throw error;
		return api.languages.initialize(deriveInitialLanguages(locales));
	}
}
