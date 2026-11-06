import { describe, expect, it, vi } from 'vitest';
import { attachmentSelection, uploadVideo } from './media.ts';
import type { CraftskyAPI } from '#lib/api/client.ts';

describe('native-style attachment selection', () => {
	it('dispatches photos and a single video without allowing mixed or multiple-video embeds', () => {
		expect(attachmentSelection([])).toBeUndefined();
		expect(
			attachmentSelection([
				{ type: 'image/png', name: 'a.png' },
				{ type: 'image/jpeg', name: 'b.jpg' }
			])
		).toBe('images');
		expect(attachmentSelection([{ type: 'video/mp4', name: 'a.mp4' }])).toBe(
			'video'
		);
		expect(attachmentSelection([{ type: '', name: 'A.MP4' }])).toBe('video');
		expect(() =>
			attachmentSelection([
				{ type: 'video/mp4', name: 'a.mp4' },
				{ type: 'image/png', name: 'a.png' }
			])
		).toThrow('Choose photos or one video');
		expect(() =>
			attachmentSelection([
				{ type: 'video/mp4', name: 'a.mp4' },
				{ type: 'video/mp4', name: 'b.mp4' }
			])
		).toThrow('Choose photos or one video');
	});
});

describe('approved video capability transport', () => {
	it('only adds a harmless MP4 free box for an explicit missing-blob recovery', async () => {
		const api = {
			media: {
				videoLimits: async () => ({ canUpload: true }),
				videoAuthorization: async () => ({
					token: 'capability-canary',
					expiresAt: new Date(Date.now() + 60000).toISOString()
				})
			}
		} as unknown as CraftskyAPI;
		const blob = {
			$type: 'blob',
			ref: { $link: 'bafy-video' },
			mimeType: 'video/mp4',
			size: 100
		};
		const fetcher = vi.fn<typeof fetch>(
			async () =>
				new Response(
					JSON.stringify({ jobId: 'job', state: 'JOB_STATE_COMPLETED', blob })
				)
		);
		await uploadVideo(
			api,
			'did:plc:maker',
			{
				id: 'v',
				name: 'v.mp4',
				mime: 'video/mp4',
				blob: new Blob([new Uint8Array([1, 2, 3])]),
				alt: ''
			},
			new AbortController().signal,
			() => {},
			fetcher,
			true
		);
		const bytes = new Uint8Array(
			await (fetcher.mock.calls[0][1]?.body as Blob).arrayBuffer()
		);
		expect([...bytes.slice(0, 11)]).toEqual([
			1, 2, 3, 0, 0, 0, 16, 102, 114, 101, 101
		]);
		expect(bytes.length).toBe(19);
	});
	it('uses an in-memory capability only for the approved upload destination and never for polling', async () => {
		vi.useFakeTimers();
		const api = {
			media: {
				videoLimits: async () => ({ canUpload: true }),
				videoAuthorization: async () => ({
					token: 'capability-canary',
					expiresAt: new Date(Date.now() + 60000).toISOString()
				})
			}
		} as unknown as CraftskyAPI;
		const blob = {
			$type: 'blob',
			ref: { $link: 'bafy-video' },
			mimeType: 'video/mp4',
			size: 100
		};
		const fetcher = vi
			.fn<typeof fetch>()
			.mockResolvedValueOnce(
				new Response(
					JSON.stringify({ jobId: 'job', state: 'JOB_STATE_PROCESSING' })
				)
			)
			.mockResolvedValueOnce(
				new Response(
					JSON.stringify({
						jobStatus: { jobId: 'job', state: 'JOB_STATE_COMPLETED', blob }
					})
				)
			);
		try {
			const promise = uploadVideo(
				api,
				'did:plc:maker',
				{
					id: 'v',
					name: 'v.mp4',
					mime: 'video/mp4',
					blob: new Blob(['video']),
					alt: 'A maker at work'
				},
				new AbortController().signal,
				() => {},
				fetcher
			);
			await vi.advanceTimersByTimeAsync(2100);
			const result = await promise;
			const [upload, options] = fetcher.mock.calls[0];
			expect(new URL(String(upload)).origin).toBe('https://video.bsky.app');
			expect(new URL(String(upload)).pathname).toBe(
				'/xrpc/app.bsky.video.uploadVideo'
			);
			expect(options).toMatchObject({
				credentials: 'omit',
				redirect: 'error',
				referrerPolicy: 'no-referrer'
			});
			expect(new Headers(options?.headers).get('Authorization')).toBe(
				'Bearer capability-canary'
			);
			expect(
				new Headers(fetcher.mock.calls[1][1]?.headers).has('Authorization')
			).toBe(false);
			expect(result).toEqual({
				video: { jobId: 'job', blob, alt: 'A maker at work' }
			});
			expect(JSON.stringify(result)).not.toContain('capability-canary');
		} finally {
			vi.useRealTimers();
		}
	});
	it('rejects a capability redirect rather than forwarding it to a second endpoint', async () => {
		const api = {
			media: {
				videoLimits: async () => ({ canUpload: true }),
				videoAuthorization: async () => ({
					token: 'capability-canary',
					expiresAt: new Date(Date.now() + 60000).toISOString()
				})
			}
		} as unknown as CraftskyAPI;
		const fetcher = vi.fn<typeof fetch>(async () => {
			throw new TypeError('redirect blocked');
		});
		await expect(
			uploadVideo(
				api,
				'did:plc:maker',
				{
					id: 'v',
					name: 'v.mp4',
					mime: 'video/mp4',
					blob: new Blob(['video']),
					alt: ''
				},
				new AbortController().signal,
				() => {},
				fetcher
			)
		).rejects.toThrow('redirect blocked');
		expect(fetcher).toHaveBeenCalledTimes(1);
		expect(fetcher.mock.calls[0][1]?.redirect).toBe('error');
	});
});

it.each(['upload', 'poll', 'body'])(
	'bounds a stalled video %s and aborts its request',
	async (stage) => {
		vi.useFakeTimers();
		try {
			const api = {
				media: {
					videoLimits: async () => ({ canUpload: true }),
					videoAuthorization: async () => ({
						token: 'PRIVATE_CAPABILITY',
						expiresAt: new Date(Date.now() + 60000).toISOString()
					})
				}
			} as unknown as CraftskyAPI;
			const signals: AbortSignal[] = [];
			const fetcher = vi.fn<typeof fetch>(async (_url, init) => {
				signals.push(init!.signal!);
				if (stage === 'poll' && signals.length === 1)
					return new Response(
						JSON.stringify({ jobId: 'job', state: 'JOB_STATE_PROCESSING' })
					);
				if (stage === 'body') return new Response(new ReadableStream());
				return new Promise(() => {});
			});
			const task = uploadVideo(
				api,
				'did:plc:a',
				{
					id: 'video',
					name: 'v.mp4',
					mime: 'video/mp4',
					alt: '',
					blob: new Blob(['video'])
				},
				new AbortController().signal,
				() => {},
				fetcher
			);
			const result = expect(task).rejects.toMatchObject({
				name: 'TimeoutError'
			});
			await vi.advanceTimersByTimeAsync(stage === 'poll' ? 32001 : 300001);
			await result;
			expect(signals.at(-1)?.aborted).toBe(true);
		} finally {
			vi.useRealTimers();
		}
	}
);
