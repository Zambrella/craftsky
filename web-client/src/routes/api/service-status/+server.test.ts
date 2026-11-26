import { afterEach, expect, it, vi } from 'vitest';
import { GET } from './+server';

afterEach(() => vi.unstubAllGlobals());

it('serves fresh status through the web origin even without upstream CORS headers', async () => {
	const first = { schemaVersion: 2, revision: 'first' };
	const second = { schemaVersion: 2, revision: 'second' };
	const upstream = vi
		.fn()
		.mockResolvedValueOnce(
			Response.json(first, { headers: { 'set-cookie': 'upstream=canary' } })
		)
		.mockResolvedValueOnce(Response.json(second));
	vi.stubGlobal('fetch', upstream);

	const response = await GET();
	expect(response.status).toBe(200);
	expect(response.headers.get('cache-control')).toBe('no-store');
	expect(response.headers.get('set-cookie')).toBeNull();
	expect(await response.json()).toEqual(first);
	expect(await (await GET()).json()).toEqual(second);

	expect(upstream).toHaveBeenCalledTimes(2);
	expect(upstream).toHaveBeenCalledWith(
		'https://status.craftsky.social/app.json',
		{
			credentials: 'omit',
			redirect: 'error',
			cache: 'no-store',
			headers: { Accept: 'application/json' },
			signal: expect.any(AbortSignal)
		}
	);
});

it('returns an unavailable response for failed, invalid or oversized status reads', async () => {
	const upstream = vi
		.fn()
		.mockRejectedValueOnce(new TypeError('fetch failed'))
		.mockResolvedValueOnce(new Response('unavailable', { status: 502 }))
		.mockResolvedValueOnce(new Response('<html>upstream error</html>'))
		.mockResolvedValueOnce(
			Response.json({
				schemaVersion: 2,
				revision: 'large',
				extra: 'x'.repeat(16384)
			})
		);
	vi.stubGlobal('fetch', upstream);

	for (let i = 0; i < 4; i++) {
		const response = await GET();
		expect(response.status).toBe(503);
		expect(response.headers.get('cache-control')).toBe('no-store');
		expect(await response.text()).toBe('');
	}
});
