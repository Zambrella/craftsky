import { afterEach, describe, expect, it, vi } from 'vitest';
import type { CraftskyAPI, JsonObject } from '#lib/api/client.ts';
import type { DraftMedia } from '../drafts/store.ts';
import { uploadBunnyVideo } from './bunny-video.ts';
import { uploadVideo, videoSourceMime } from './media.ts';

const jobId = 'aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa';
const token = 'c'.repeat(64);
const authorization = (): JsonObject => ({
	provider: 'bunny',
	jobId,
	videoId: 'bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb',
	libraryId: '123',
	token,
	expiresAt: new Date(Date.now() + 1800000).toISOString(),
	title: '2026-11-05 12:00 UTC · aaaaaaaa',
	collectionId: 'dddddddd-dddd-4ddd-dddd-dddddddddddd'
});
const bytes = new Uint8Array([
	0,
	0,
	0,
	20,
	...new TextEncoder().encode('ftypqt  ')
]);
const media = (mime = 'video/quicktime'): DraftMedia => ({
	id: 'video',
	name: 'private-filename.mov',
	mime,
	blob: new Blob([bytes]),
	alt: 'A knitted scarf',
	width: 1080,
	height: 1920
});
const completed = () => ({
	jobId,
	state: 'JOB_STATE_COMPLETED',
	blob: {
		$type: 'blob',
		ref: { $link: 'bafyprocessed' },
		mimeType: 'video/mp4',
		size: 100
	}
});
const tus = () =>
	vi.fn<typeof fetch>(async (_url, options) => {
		if (options?.method === 'POST')
			return new Response(null, {
				status: 201,
				headers: { Location: '/tusupload/test-upload' }
			});
		return new Response(null, {
			status: 204,
			headers: {
				'Upload-Offset': String(
					Number(new Headers(options?.headers).get('Upload-Offset')) +
						(options?.body as Blob).size
				)
			}
		});
	});
const signal = () => new AbortController().signal;
const metadata = (options?: RequestInit) =>
	Object.fromEntries(
		new Headers(options?.headers)
			.get('Upload-Metadata')!
			.split(',')
			.map((field) => {
				const [key, value] = field.split(' ');
				return [
					key,
					new TextDecoder().decode(
						Uint8Array.from(atob(value), (c) => c.charCodeAt(0))
					)
				];
			})
	);
afterEach(() => vi.useRealTimers());

describe('Bunny upload and AppView publication', () => {
	it.each(['video/mp4', 'video/quicktime'])(
		'uploads %s and polls AppView with no vendor capability',
		async (mime) => {
			vi.useFakeTimers();
			const fetcher = tus();
			const videoStatus = vi.fn(async () => ({ jobStatus: completed() }));
			const api = {
				media: {
					videoLimits: async () => ({ canUpload: true }),
					videoAuthorization: async () => authorization(),
					videoStatus
				}
			} as unknown as CraftskyAPI;
			const activeSignal = signal();
			const upload = uploadVideo(
				api,
				'did:plc:maker',
				media(mime),
				activeSignal,
				() => {},
				fetcher,
				true
			);
			const result = expect(upload).resolves.toEqual({
				video: {
					jobId,
					blob: completed().blob,
					alt: 'A knitted scarf',
					aspectRatio: { width: 1080, height: 1920 }
				}
			});
			await vi.runAllTimersAsync();
			await result;
			expect(videoStatus).toHaveBeenCalledWith(jobId, activeSignal);
			expect(metadata(fetcher.mock.calls[0][1])).toEqual({
				filetype: mime,
				title: authorization().title,
				collection: authorization().collectionId
			});
			expect(
				new Uint8Array(
					await (fetcher.mock.calls[1][1]!.body as Blob).arrayBuffer()
				)
			).toEqual(bytes);
			for (const [url, options] of fetcher.mock.calls) {
				expect(new URL(String(url)).origin).toBe('https://video.bunnycdn.com');
				expect(options?.credentials).toBe('omit');
				expect(options?.redirect).toBe('error');
				const h = new Headers(options?.headers);
				expect(h.get('Authorization')).toBeNull();
				expect(h.get('Cookie')).toBeNull();
				expect(h.get('AccessKey')).toBeNull();
				expect(h.get('AuthorizationSignature')).toBe(token);
			}
		}
	);
	it('resumes from the server offset after interrupted bytes', async () => {
		const fetcher = tus();
		let patches = 0;
		fetcher.mockImplementation(async (_url, options) => {
			if (options?.method === 'POST')
				return new Response(null, {
					status: 201,
					headers: { Location: '/tusupload/resume' }
				});
			if (options?.method === 'HEAD')
				return new Response(null, {
					status: 200,
					headers: { 'Upload-Offset': '3' }
				});
			if (++patches === 1) throw new TypeError('connection interrupted');
			return new Response(null, {
				status: 204,
				headers: { 'Upload-Offset': String(bytes.length) }
			});
		});
		expect(
			await uploadBunnyVideo(
				media(),
				authorization(),
				signal(),
				() => {},
				fetcher
			)
		).toBe(jobId);
		expect(fetcher.mock.calls.map((c) => c[1]?.method)).toEqual([
			'POST',
			'PATCH',
			'HEAD',
			'PATCH'
		]);
		expect(
			new Headers(fetcher.mock.calls[3][1]?.headers).get('Upload-Offset')
		).toBe('3');
		expect(
			new Uint8Array(
				await (fetcher.mock.calls[3][1]!.body as Blob).arrayBuffer()
			)
		).toEqual(bytes.slice(3));
	});
	it('sends bounded Blob slices instead of buffering the original', async () => {
		const source = new Blob([new Uint8Array(8 * 1024 * 1024 + 3)]);
		const fullRead = vi.spyOn(source, 'arrayBuffer');
		const fetcher = tus();
		await uploadBunnyVideo(
			{ ...media(), blob: source },
			authorization(),
			signal(),
			() => {},
			fetcher
		);
		expect(fullRead).not.toHaveBeenCalled();
		expect(
			fetcher.mock.calls
				.filter((c) => c[1]?.method === 'PATCH')
				.map((c) => (c[1]?.body as Blob).size)
		).toEqual([8 * 1024 * 1024, 3]);
	});
	it('stops in-flight requests on cancellation without attempting recovery', async () => {
		const controller = new AbortController();
		const fetcher = vi.fn<typeof fetch>(async (_url, options) => {
			if (options?.method === 'POST')
				return new Response(null, {
					status: 201,
					headers: { Location: '/tusupload/cancel' }
				});
			return new Promise((_resolve, reject) => {
				options!.signal!.addEventListener('abort', () =>
					reject(new DOMException('Cancelled', 'AbortError'))
				);
				controller.abort();
			});
		});
		await expect(
			uploadBunnyVideo(
				media(),
				authorization(),
				controller.signal,
				() => {},
				fetcher
			)
		).rejects.toMatchObject({ name: 'AbortError' });
		expect(fetcher.mock.calls.map((c) => c[1]?.method)).toEqual([
			'POST',
			'PATCH'
		]);
	});
	it('bounds recovery attempts on repeated network failure', async () => {
		const fetcher = vi.fn<typeof fetch>(async (_url, options) => {
			if (options?.method === 'POST')
				return new Response(null, {
					status: 201,
					headers: { Location: '/tusupload/retry' }
				});
			if (options?.method === 'HEAD')
				return new Response(null, {
					status: 200,
					headers: { 'Upload-Offset': '0' }
				});
			throw new TypeError('offline');
		});
		await expect(
			uploadBunnyVideo(media(), authorization(), signal(), () => {}, fetcher)
		).rejects.toThrow('Could not finish');
		expect(
			fetcher.mock.calls.filter((c) => c[1]?.method === 'PATCH')
		).toHaveLength(4);
	});
	it.each([
		'https://untrusted.example/tusupload/x',
		'http://video.bunnycdn.com/tusupload/x',
		'https://user:pass@video.bunnycdn.com/tusupload/x',
		'https://video.bunnycdn.com:444/tusupload/x',
		'/tusupload/x?token=private',
		'/tusupload/x#fragment',
		'/not-tus/x',
		'/tusupload/../other'
	])(
		'rejects an unapproved Location %s before sending credentials',
		async (location) => {
			const fetcher = vi.fn<typeof fetch>(
				async () =>
					new Response(null, { status: 201, headers: { Location: location } })
			);
			await expect(
				uploadBunnyVideo(media(), authorization(), signal(), () => {}, fetcher)
			).rejects.toThrow('invalid upload response');
			expect(fetcher).toHaveBeenCalledTimes(1);
		}
	);
	it.each(['-1', '1.5', '999999999999999999999', '13', '', 'NaN'])(
		'rejects invalid upload offset %s',
		async (value) => {
			const fetcher = tus();
			fetcher.mockResolvedValueOnce(
				new Response(null, {
					status: 201,
					headers: { Location: '/tusupload/offset' }
				})
			);
			fetcher.mockResolvedValueOnce(
				new Response(null, { status: 204, headers: { 'Upload-Offset': value } })
			);
			await expect(
				uploadBunnyVideo(media(), authorization(), signal(), () => {}, fetcher)
			).rejects.toThrow('invalid upload response');
		}
	);
	it.each([
		{ token: 'wrong' },
		{ expiresAt: 'invalid' },
		{ expiresAt: '2000-01-01' },
		{ jobId: 'other' },
		{ collectionId: 'bad' },
		{ provider: 'unknown' }
	])('rejects invalid capabilities before any upload %j', async (change) => {
		const fetcher = tus();
		await expect(
			uploadBunnyVideo(
				media(),
				{ ...authorization(), ...change },
				signal(),
				() => {},
				fetcher
			)
		).rejects.toThrow('authorize');
		expect(fetcher).not.toHaveBeenCalled();
	});
	it.each(['JOB_STATE_FAILED', 'wrong-job', 'bad-blob'])(
		'rejects invalid processing result %s',
		async (kind) => {
			vi.useFakeTimers();
			const state = completed();
			if (kind === 'wrong-job')
				state.jobId = 'eeeeeeee-eeee-4eee-eeee-eeeeeeeeeeee';
			else if (kind === 'bad-blob') state.blob.mimeType = 'video/quicktime';
			else state.state = kind;
			const api = {
				media: {
					videoLimits: async () => ({ canUpload: true }),
					videoAuthorization: async () => authorization(),
					videoStatus: async () => ({ jobStatus: state })
				}
			} as unknown as CraftskyAPI;
			const result = expect(
				uploadVideo(api, 'did:plc:maker', media(), signal(), () => {}, tus())
			).rejects.toThrow();
			await vi.runAllTimersAsync();
			await result;
		}
	);
	it('detects MOV/MP4 from the header and rejects renamed or truncated files', () => {
		expect(videoSourceMime(bytes)).toBe('video/quicktime');
		expect(
			videoSourceMime(
				new Uint8Array([0, 0, 0, 24, ...new TextEncoder().encode('ftypisom')])
			)
		).toBe('video/mp4');
		expect(videoSourceMime(bytes.slice(0, 11))).toBeUndefined();
		expect(videoSourceMime(new Uint8Array(12))).toBeUndefined();
	});
});
