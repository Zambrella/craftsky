import { it, expect, vi, afterEach } from 'vitest';
import { publicReadResponse } from './public-proxy';
afterEach(() => vi.unstubAllEnvs());

// IT-008 / RULE-001, NFR-004 / AC-006, AC-010.
it('proxies only the bounded anonymous read contract without credentials', async () => {
	vi.stubEnv('APPVIEW_ORIGIN', 'https://appview.example');
	const fetcher = vi
		.fn<typeof fetch>()
		.mockResolvedValue(Response.json({ items: [] }));
	const response = await publicReadResponse(
		fetcher,
		'search/posts',
		new URLSearchParams('q=knitting&limit=20')
	);
	expect(response.status).toBe(200);
	expect(response.headers.get('cache-control')).toBe('no-store');
	expect(String(fetcher.mock.calls[0][0])).toBe(
		'https://appview.example/v1/public/search/posts?q=knitting&limit=20'
	);
	expect(fetcher.mock.calls[0][1]?.credentials).toBe('omit');
	expect(fetcher.mock.calls[0][1]?.headers).toEqual({
		accept: 'application/json'
	});
});
it('rejects private and discovery routes before calling upstream', async () => {
	const fetcher = vi.fn<typeof fetch>();
	for (const path of [
		'profiles/me',
		'profiles/recent',
		'search/suggestions',
		'notifications'
	]) {
		const response = await publicReadResponse(
			fetcher,
			path,
			new URLSearchParams()
		);
		expect(response.status).toBe(400);
	}
	expect(fetcher).not.toHaveBeenCalled();
});
it('maps opt-out without reflecting protected upstream details', async () => {
	vi.stubEnv('APPVIEW_ORIGIN', 'https://appview.example');
	const fetcher = vi
		.fn<typeof fetch>()
		.mockResolvedValue(
			Response.json(
				{ name: 'PRIVATE_NAME', image: 'PRIVATE_MEDIA' },
				{ status: 403 }
			)
		);
	const response = await publicReadResponse(
		fetcher,
		'posts/did:plc:alice/first',
		new URLSearchParams()
	);
	expect(response.status).toBe(403);
	expect(await response.json()).toEqual({
		error: 'authentication_required',
		message: 'Sign in to view this content.'
	});
});
