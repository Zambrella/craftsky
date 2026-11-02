import { beforeEach, afterEach, it, expect, vi } from 'vitest';
import { publicAPI } from './api';
beforeEach(() => vi.stubEnv('APPVIEW_ORIGIN', 'https://appview.example'));
afterEach(() => vi.unstubAllEnvs());
it('fetches only anonymous data without following redirects', async () => {
	const fake = vi
		.fn<typeof fetch>()
		.mockResolvedValue(Response.json({ text: 'public' }));
	expect(await publicAPI(fake, 'profiles/did%3Aplc%3Aa')).toEqual({
		text: 'public'
	});
	const [url, options] = fake.mock.calls[0];
	expect(String(url)).toBe(
		'https://appview.example/v1/public/profiles/did%3Aplc%3Aa'
	);
	expect(options?.redirect).toBe('manual');
	expect(options?.credentials).toBe('omit');
	expect(options?.headers).toEqual({ accept: 'application/json' });
});
it.each([400, 404, 429, 500, 503])(
	'maps upstream %s without reflecting error payloads',
	async (status) => {
		const fake = vi
			.fn<typeof fetch>()
			.mockResolvedValue(new Response('PRIVATE_CANARY', { status }));
		await expect(publicAPI(fake, 'posts/a/b')).rejects.toMatchObject({
			status: status === 500 ? 503 : status
		});
	}
);
it('bounds streamed response size', async () => {
	const fake = vi
		.fn<typeof fetch>()
		.mockResolvedValue(new Response('x'.repeat(2 * 1024 * 1024 + 1)));
	await expect(publicAPI(fake, 'posts/a/b')).rejects.toMatchObject({
		status: 503
	});
});
it('does not reflect malformed JSON or network errors', async () => {
	for (const fake of [
		vi.fn<typeof fetch>().mockResolvedValue(new Response('PRIVATE_CANARY')),
		vi.fn<typeof fetch>().mockRejectedValue(new Error('PRIVATE_CANARY'))
	]) {
		await expect(publicAPI(fake, 'posts/a/b')).rejects.toMatchObject({
			status: 503
		});
	}
});

it('rejects redirects without requesting the Location target', async () => {
	const fake = vi
		.fn<typeof fetch>()
		.mockResolvedValue(
			new Response(null, {
				status: 302,
				headers: { location: 'https://private.example' }
			})
		);
	await expect(publicAPI(fake, 'posts/a/b')).rejects.toMatchObject({
		status: 503
	});
	expect(fake).toHaveBeenCalledTimes(1);
});
