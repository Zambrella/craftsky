import { describe, expect, it, vi } from 'vitest';
import { profileShareAvatar } from './profile-share-avatar';
const source =
	'https://cdn.bsky.app/img/avatar/plain/did:plc:abcdefghijklmnopqrstuvwx/bafkreitest@jpeg';

describe('public share-card avatar boundary', () => {
	it('fetches only avatar bytes without caller credentials and prevents content sniffing', async () => {
		const fetcher = vi.fn<typeof fetch>().mockResolvedValue(
			new Response(new Uint8Array([1, 2, 3]), {
				headers: { 'content-type': 'image/jpeg' }
			})
		);
		const result = await profileShareAvatar(source, fetcher);
		expect(result.status).toBe(200);
		expect([...new Uint8Array(await result.arrayBuffer())]).toEqual([1, 2, 3]);
		expect(result.headers.get('x-content-type-options')).toBe('nosniff');
		expect(result.headers.get('cache-control')).toBe('no-store');
		expect(fetcher).toHaveBeenCalledWith(
			source,
			expect.objectContaining({
				credentials: 'omit',
				redirect: 'manual',
				headers: { accept: 'image/jpeg,image/png,image/webp,image/gif' }
			})
		);
	});
	it.each([
		null,
		'http://localhost/private',
		'https://example.com/avatar',
		source.replace('cdn.bsky.app', 'cdn.bsky.app.example.org'),
		source.replace('https://', 'https://name:password@'),
		source + '?token=private',
		source + '#private',
		source.replace('/avatar/', '/feed_fullsize/'),
		source.replace('/plain/', '/plain/../'),
		source.replace('@jpeg', '@svg')
	])('rejects unsupported targets without fetching', async (value) => {
		const fetcher = vi.fn<typeof fetch>();
		expect((await profileShareAvatar(value, fetcher)).status).toBe(400);
		expect(fetcher).not.toHaveBeenCalled();
	});
	it.each([
		new Response(null, {
			status: 302,
			headers: { location: 'https://example.com' }
		}),
		new Response('private', { headers: { 'content-type': 'text/html' } })
	])(
		'does not follow redirects or return non-image bodies',
		async (response) => {
			const fetcher = vi.fn<typeof fetch>().mockResolvedValue(response);
			expect((await profileShareAvatar(source, fetcher)).status).toBe(502);
			expect(fetcher).toHaveBeenCalledTimes(1);
		}
	);
	it('bounds declared and streamed response bodies', async () => {
		for (const response of [
			new Response(null, {
				headers: { 'content-type': 'image/png', 'content-length': '2097153' }
			}),
			new Response(new Uint8Array(2097153), {
				headers: { 'content-type': 'image/png' }
			})
		]) {
			expect(
				(
					await profileShareAvatar(
						source,
						vi.fn<typeof fetch>().mockResolvedValue(response)
					)
				).status
			).toBe(413);
		}
	});
	it('handles unavailable images without leaking upstream payloads', async () => {
		const fetcher = vi
			.fn<typeof fetch>()
			.mockRejectedValue(new Error('PRIVATE_CANARY'));
		const response = await profileShareAvatar(source, fetcher);
		expect(response.status).toBe(502);
		expect(await response.text()).toBe('');
	});
});
