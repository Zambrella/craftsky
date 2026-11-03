import { describe, expect, it, vi } from 'vitest';
import {
	APIError,
	CraftskyAPI,
	type LanguagePreferences
} from '#lib/api/client.ts';
import { deriveInitialLanguages, loadLanguagePreferences } from './languages';

describe('initial account languages', () => {
	it.each([
		{
			locales: ['fr-CA', 'cy-GB', 'fr-FR'],
			primary: 'fr',
			content: ['fr', 'cy', 'en']
		},
		{ locales: ['fr-FR', 'en-GB'], primary: 'en', content: ['fr', 'en'] },
		{
			locales: ['zh-Hant-TW', 'pt-BR'],
			primary: 'zh',
			content: ['zh', 'pt', 'en']
		},
		{
			locales: ['', 'not_a_locale', 'zz-ZZ', 'und'],
			primary: 'en',
			content: ['en']
		},
		{ locales: [], primary: 'en', content: ['en'] }
	])(
		'matches the mobile defaults for $locales',
		({ locales, primary, content }) => {
			expect(deriveInitialLanguages(locales)).toEqual({
				primaryLanguage: primary,
				contentLanguages: content
			});
		}
	);

	function setup(responses: Response[]) {
		const fetcher = vi.fn<typeof fetch>(async () => responses.shift()!);
		const api = new CraftskyAPI(fetcher, '/api/appview', {
			headers: { 'X-Craftsky-Account-Id': 'owner-account' }
		});
		return { fetcher, api };
	}
	const response = (body: unknown, status = 200) =>
		Response.json(body, { status });
	const missing = () =>
		response(
			{ error: 'language_preferences_not_found', message: 'Not found' },
			404
		);

	it('uses existing preferences, including an intentionally unfiltered content selection', async () => {
		const saved = { primaryLanguage: 'cy', contentLanguages: [] };
		const { api, fetcher } = setup([response(saved)]);
		expect(await loadLanguagePreferences(api, ['en-GB'])).toEqual(saved);
		expect(fetcher).toHaveBeenCalledTimes(1);
		expect(fetcher.mock.calls[0][1]?.method).toBe('GET');
	});
	it('initializes only a missing row through the same account-bound API and trusts the authoritative result', async () => {
		// Another tab/device can initialize first; its stored choice must win.
		const authoritative: LanguagePreferences = {
			primaryLanguage: 'es',
			contentLanguages: ['es']
		};
		const { api, fetcher } = setup([missing(), response(authoritative)]);
		expect(await loadLanguagePreferences(api, ['fr-CA'])).toEqual(
			authoritative
		);
		expect(fetcher).toHaveBeenCalledTimes(2);
		const [url, request] = fetcher.mock.calls[1];
		expect(url).toBe('/api/appview/v1/languages/preferences/initialize');
		expect(request?.method).toBe('POST');
		expect(JSON.parse(String(request?.body))).toEqual({
			primaryLanguage: 'fr',
			contentLanguages: ['fr', 'en']
		});
		for (const [, init] of fetcher.mock.calls)
			expect(new Headers(init?.headers).get('X-Craftsky-Account-Id')).toBe(
				'owner-account'
			);
	});
	it.each([
		[404, 'not_found'],
		[503, 'unavailable'],
		[401, 'session_expired'],
		[500, 'language_preferences_not_found']
	])(
		'does not initialize after an unrelated read failure (%s, %s)',
		async (status, error) => {
			const { api, fetcher } = setup([
				response({ error, message: 'Request failed' }, status)
			]);
			await expect(loadLanguagePreferences(api, ['en'])).rejects.toBeInstanceOf(
				APIError
			);
			expect(fetcher).toHaveBeenCalledTimes(1);
		}
	);
	it('propagates initialization failure so the account gate can retry rather than loading feeds', async () => {
		const { api } = setup([
			missing(),
			response({ error: 'unavailable', message: 'Try again' }, 503)
		]);
		await expect(loadLanguagePreferences(api, ['en'])).rejects.toMatchObject({
			status: 503,
			code: 'unavailable'
		});
	});
});
