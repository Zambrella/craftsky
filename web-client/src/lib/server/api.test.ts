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
it('decodes UTF-8 characters split across streamed chunks', async () => {
	const bytes = new TextEncoder().encode(JSON.stringify({ text: '🧶 café' }));
	const fake = vi.fn<typeof fetch>().mockResolvedValue(
		new Response(
			new ReadableStream({
				start(controller) {
					for (const byte of bytes) controller.enqueue(Uint8Array.of(byte));
					controller.close();
				}
			})
		)
	);
	expect(await publicAPI(fake, 'posts/a/b')).toEqual({ text: '🧶 café' });
});
it.each(['json', 'caption'] as const)(
	'enforces the exact streamed byte boundary for %s',
	async (format) => {
		const limit = format === 'caption' ? 20000 : 2 * 1024 * 1024;
		const text =
			format === 'caption'
				? 'WEBVTT\n' + 'x'.repeat(limit - 7)
				: '"' + 'x'.repeat(limit - 2) + '"';
		const bytes = new TextEncoder().encode(text);
		const accepted = vi
			.fn<typeof fetch>()
			.mockResolvedValue(new Response(bytes));
		expect(await publicAPI(accepted, 'posts/a/b', format)).toBe(
			format === 'caption' ? text : JSON.parse(text)
		);
		const cancel = vi.fn();
		const oversized = vi.fn<typeof fetch>().mockResolvedValue(
			new Response(
				new ReadableStream({
					start(controller) {
						controller.enqueue(bytes);
						controller.enqueue(Uint8Array.of(32));
					},
					cancel
				})
			)
		);
		await expect(
			publicAPI(oversized, 'posts/a/b', format)
		).rejects.toMatchObject({ status: 503 });
		expect(cancel).toHaveBeenCalledOnce();
	}
);
it.each([302, 400, 403, 404, 429, 500, 503])(
	'maps upstream %s and cancels its unread body without reflecting it',
	async (status) => {
		const cancel = vi.fn();
		const fake = vi.fn<typeof fetch>().mockResolvedValue(
			new Response(
				new ReadableStream({
					start(controller) {
						controller.enqueue(new TextEncoder().encode('PRIVATE_CANARY'));
					},
					cancel
				}),
				{ status }
			)
		);
		await expect(publicAPI(fake, 'posts/a/b')).rejects.toMatchObject({
			status: status === 302 || status === 500 ? 503 : status,
			body: { message: expect.not.stringContaining('PRIVATE_CANARY') }
		});
		expect(cancel).toHaveBeenCalledOnce();
	}
);
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
	const fake = vi.fn<typeof fetch>().mockResolvedValue(
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

// UT-008 / FR-011 / AC-011: opt-out is an authentication-required result,
// not a transient upstream failure or a source of private metadata.
it('maps a known opt-out to generic authentication-required text', async () => {
	const fake = vi
		.fn<typeof fetch>()
		.mockResolvedValue(
			new Response('PRIVATE_NAME PRIVATE_MEDIA', { status: 403 })
		);
	await expect(publicAPI(fake, 'posts/a/b')).rejects.toMatchObject({
		status: 403,
		body: { message: 'Sign in to view this content.' }
	});
});
