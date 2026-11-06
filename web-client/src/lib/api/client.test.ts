import { describe, expect, it, vi } from 'vitest';
import {
	CraftskyAPI,
	APIError,
	MutationOutcomeUnknownError,
	APITransport,
	queryString
} from './client';

const key = '62cdf11e-c422-4cc2-a0ed-5c86aa58436c';
function setup(value: unknown = {}, status = 200, headers?: HeadersInit) {
	const fetcher = vi.fn<typeof fetch>(
		async () =>
			new Response(status === 204 ? null : JSON.stringify(value), {
				status,
				headers
			})
	);
	const api = new CraftskyAPI(fetcher);
	const request = () => {
		const [url, options] = fetcher.mock.calls.at(-1)!;
		return {
			url: String(url),
			...options,
			headers: new Headers(options?.headers),
			json: options?.body ? JSON.parse(String(options.body)) : undefined
		};
	};
	return { api, fetcher, request };
}

describe('AppView wire contracts', () => {
	it('polls Bunny video completion through the authenticated AppView POST', async () => {
		const jobId = 'aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa';
		const controller = new AbortController();
		const { api, request, fetcher } = setup({
			jobStatus: { jobId, state: 'JOB_STATE_PROCESSING' }
		});
		expect(await api.media.videoStatus(jobId, controller.signal)).toMatchObject(
			{ jobStatus: { jobId } }
		);
		expect(request().method).toBe('POST');
		expect(request().url).toBe(`/api/appview/v1/blobs/videos/${jobId}/status`);
		expect(request().signal).toBeInstanceOf(AbortSignal);
		expect(request().signal?.aborted).toBe(false);
		expect(request().body).toBeUndefined();
		expect(() => api.media.videoStatus('../authorization')).toThrow(
			'Invalid video job'
		);
		expect(fetcher).toHaveBeenCalledTimes(1);
	});
	it('uses the response request ID when an error envelope is missing it and retains parse causes', async () => {
		const requestId = '12345678-1234-4123-8123-123456789012';
		const { api } = setup(
			{ error: 'internal_error', message: 'private' },
			500,
			{ 'X-Request-ID': requestId }
		);
		await expect(api.account.whoami()).rejects.toMatchObject({
			status: 500,
			requestId
		});
		const broken = new APITransport(
			async () =>
				new Response('private unreadable body', {
					status: 502,
					headers: { 'X-Request-ID': requestId }
				})
		);
		await expect(broken.get('/v1/whoami')).rejects.toMatchObject({
			status: 502,
			code: 'invalid_response',
			requestId,
			cause: expect.any(SyntaxError)
		});
	});
	it('omits false unfiled flags because saved-post queries only accept true when present', async () => {
		const { api, request } = setup({ items: [] });
		await api.saved.list({ unfiled: false, sort: 'newest' });
		expect(request().url).toBe('/api/appview/v1/saved-posts?sort=newest');
		await api.saved.list({ unfiled: true });
		expect(request().url).toBe('/api/appview/v1/saved-posts?unfiled=true');
	});

	it('keeps Instagram import IDs and creation counts in their actual response envelope', async () => {
		const summary = {
			importId: 'import-1',
			state: 'active',
			sourceType: 'manual',
			followingCount: 1,
			createdAt: '2026-11-01T12:00:00Z'
		};
		const { api, request } = setup(
			{ import: summary, counts: { followingCount: 1 } },
			201
		);
		const result = await api.instagram.createImport({
			sourceType: 'manual',
			entries: [{ username: 'stitches' }]
		});
		expect(result.import.importId).toBe('import-1');
		expect(result.counts.followingCount).toBe(1);
		expect(request().json).toEqual({
			sourceType: 'manual',
			entries: [{ username: 'stitches' }]
		});
	});
	it('uses same-origin cookie transport without taking ownership of credentials', async () => {
		const { api, request } = setup();
		await api.profiles.me();
		expect(request()).toMatchObject({
			url: '/api/appview/v1/profiles/me',
			credentials: 'same-origin',
			redirect: 'manual',
			cache: 'no-store'
		});
		expect(request().headers.has('Authorization')).toBe(false);
	});
	it('encodes profile and post identifiers as single URL segments', async () => {
		const { api, request } = setup();
		await api.profiles.get('maker.example');
		expect(request().url).toBe('/api/appview/v1/profiles/%40maker.example');
		await api.posts.get('did:plc:abc', 'key/with?symbols');
		expect(request().url).toBe(
			'/api/appview/v1/posts/did%3Aplc%3Aabc/key%2Fwith%3Fsymbols'
		);
	});
	it('passes opaque cursors unchanged and repeats multi-valued craft filters', async () => {
		const { api, request } = setup({ items: [], cursor: 'next' });
		const cursor = 'a+/==:&?';
		await api.search.browseProjects({
			cursor,
			limit: 20,
			sort: 'chronological',
			craftType: ['knitting', 'sewing'],
			filters: { status: ['finished', 'in-progress'] }
		});
		const url = new URL(request().url, 'https://craftsky.test');
		expect(url.searchParams.get('cursor')).toBe(cursor);
		expect(url.searchParams.getAll('craftType')).toEqual([
			'knitting',
			'sewing'
		]);
		expect(url.searchParams.getAll('status')).toEqual([
			'finished',
			'in-progress'
		]);
		expect(url.searchParams.get('sort')).toBe('chronological');
	});
	it('preserves zero and false query values while omitting absent values', () => {
		expect(queryString({ a: false, b: 0, c: null, d: undefined })).toBe(
			'?a=false&b=0'
		);
	});
	it('keeps project text search separate from craft browsing filters and sort', async () => {
		const { api, request } = setup({ items: [] });
		const browserOptions = {
			q: 'linen',
			cursor: 'opaque',
			limit: 12,
			sort: 'popular' as const,
			filters: { status: ['finished'] }
		};
		await api.search.projects(browserOptions);
		expect(request().url).toBe(
			'/api/appview/v1/search/projects?q=linen&cursor=opaque&limit=12'
		);
	});
	it('preserves the video completion proof in create requests', async () => {
		const { api, request } = setup(
			{
				uri: 'at://created',
				cid: 'created-cid',
				rkey: 'post',
				text: 'A project',
				author: { did: 'did:plc:maker', handle: 'maker.example' }
			},
			201
		);
		const blob = {
			ref: { $link: 'bafy-video' },
			mimeType: 'video/mp4',
			size: 200
		};
		await api.posts.create(
			{
				text: 'A project',
				langs: ['en'],
				sponsored: false,
				embed: {
					video: { jobId: 'verified-job', blob, alt: 'Working at the loom' }
				}
			},
			key
		);
		expect(request().json).toEqual({
			text: 'A project',
			langs: ['en'],
			sponsored: false,
			embed: {
				video: { jobId: 'verified-job', blob, alt: 'Working at the loom' }
			}
		});
		expect(request().headers.get('Idempotency-Key')).toBe(key);
	});
	it('uses optimistic concurrency and a changes envelope for edits', async () => {
		const { api, request } = setup({
			acceptedCid: 'new-cid',
			changed: true,
			editedAt: '2026-01-01T00:00:00Z'
		});
		await api.posts.edit(
			'did:plc:a',
			'post',
			{ text: 'Edited', facets: null },
			'original-cid',
			key
		);
		expect(request().method).toBe('PATCH');
		expect(request().headers.get('If-Match')).toBe('original-cid');
		expect(request().json).toEqual({
			changes: { text: 'Edited', facets: null }
		});
	});
	it('sends private notification patches and profile customisation using exact camelCase fields', async () => {
		const { api, request } = setup();
		const patch = { preferences: { likes: { pushEnabled: false } } };
		await api.notifications.patchPreferences(patch);
		expect(request().json).toEqual(patch);
		await api.profiles.customisation({
			colour: 'cobalt',
			profileBackground: 'none'
		});
		expect(request().json).toEqual({
			colour: 'cobalt',
			profileBackground: 'none'
		});
	});
	it('passes focused comments and reports without translating their field names', async () => {
		const { api, request } = setup();
		await api.posts.comments('did:plc:a', 'post', {
			focus: 'at://did:plc:b/social.craftsky.feed.post/reply',
			sort: 'follows'
		});
		expect(
			new URL(request().url, 'https://test').searchParams.get('focus')
		).toBe('at://did:plc:b/social.craftsky.feed.post/reply');
		await api.posts.report('did:plc:a', 'post', {
			reasonType: 'spam',
			details: 'Repeated promotion'
		});
		expect(request().json).toEqual({
			reasonType: 'spam',
			details: 'Repeated promotion'
		});
	});
	it('separates local chat operation receipts from PDS write keys', async () => {
		const { api, request } = setup({
			operationId: key,
			kind: 'send',
			state: 'confirmed',
			messageId: 'msg'
		});
		const result = await api.chat.send('convo/1', key, 'Hello');
		expect(request().url).toBe(
			'/api/appview/v1/chat/conversations/convo%2F1/messages'
		);
		expect(request().json).toEqual({ operationId: key, text: 'Hello' });
		expect(request().headers.has('Idempotency-Key')).toBe(false);
		expect(result.messageId).toBe('msg');
	});
	it('uses conditional creation for business declarations', async () => {
		const { api, request } = setup({ cid: 'new-cid' });
		await api.business.saveProfile({ businessTypes: ['dyer'] }, null, key);
		expect(request().headers.get('If-Match')).toBe('*');
		expect(request().json).toEqual({ businessTypes: ['dyer'] });
	});
	it('uploads scheduled media bytes without JSON encoding them', async () => {
		const { api, fetcher } = setup({ id: 'media' });
		const bytes = new Uint8Array([1, 2, 3]);
		await api.schedules.stageMedia('media', bytes, 'image/jpeg');
		const [, options] = fetcher.mock.calls[0];
		expect(options?.body).toBe(bytes);
		expect(new Headers(options?.headers).get('Content-Type')).toBe(
			'image/jpeg'
		);
	});
	it('allows the background publication response without pretending the post already exists', async () => {
		const { api } = setup({ status: 'publishing' }, 202);
		expect(await api.schedules.publish('schedule', { text: 'Later' })).toEqual({
			status: 'publishing'
		});
	});
	it('uses the mention resolver route and canonical follower growth periods', async () => {
		const { api, request } = setup({ did: 'did:plc:maker' });
		await api.search.resolveMention('maker.example');
		expect(request().url).toBe(
			'/api/appview/v1/facets/mentions/resolve?handle=maker.example'
		);
		await api.profiles.followerGrowth('30d');
		expect(request().url).toBe(
			'/api/appview/v1/profiles/me/follower-growth?period=30d'
		);
	});
});

describe('transport failure and mutation semantics', () => {
	it('retains an Instagram follow operation key if a nominal success names another suggestion or unconfirmed state', async () => {
		const { api } = setup(
			{ suggestionId: 'different', state: 'accepting' },
			200
		);
		await expect(
			api.instagram.acceptSuggestion('original', key)
		).rejects.toMatchObject({
			name: 'MutationOutcomeUnknownError',
			operationKey: key
		});
	});
	it('retains the write key when a successful create response lacks the accepted post identity', async () => {
		const { api } = setup({}, 201);
		await expect(
			api.posts.create({ text: 'Hello', sponsored: false, langs: ['en'] }, key)
		).rejects.toMatchObject({
			name: 'MutationOutcomeUnknownError',
			operationKey: key
		});
	});
	it('preserves the public error envelope and validation fields', async () => {
		const { api } = setup(
			{
				error: 'validation_failed',
				message: 'Check the form',
				requestId: '12345678-1234-4123-8123-123456789012',
				fields: { text: 'Too long' }
			},
			422
		);
		await expect(api.posts.get('did:plc:a', 'post')).rejects.toMatchObject({
			status: 422,
			code: 'validation_failed',
			message: 'Check the form',
			requestId: '12345678-1234-4123-8123-123456789012',
			fields: { text: 'Too long' },
			retryable: false
		});
	});
	it('does not retry or silently create another operation for an ambiguous PDS response', async () => {
		const { api, fetcher } = setup({ status: 'ambiguous' }, 202, {
			'Retry-After': '2'
		});
		await expect(
			api.posts.like('did:plc:a', 'post', key)
		).rejects.toMatchObject({
			name: 'MutationOutcomeUnknownError',
			operationKey: key,
			retryAfter: 2,
			responsePending: true
		});
		expect(fetcher).toHaveBeenCalledTimes(1);
	});
	it.each([500, 502, 503])(
		'keeps the operation key when a PDS write returns %s',
		async (status) => {
			const { api } = setup(
				{ error: 'upstream_unavailable', message: 'Unavailable' },
				status
			);
			await expect(
				api.posts.like('did:plc:a', 'post', key)
			).rejects.toBeInstanceOf(MutationOutcomeUnknownError);
		}
	);
	it('distinguishes a definite missing video rejection from an unknown write', async () => {
		const { api } = setup(
			{ error: 'video_blob_missing', message: 'Upload again' },
			503
		);
		await expect(
			api.posts.like('did:plc:a', 'post', key)
		).rejects.toMatchObject({ name: 'APIError', code: 'video_blob_missing' });
	});
	it('retains the PDS key after network loss and classifies read network loss separately', async () => {
		const fetcher = vi.fn<typeof fetch>(async () => {
			throw new TypeError('offline');
		});
		const api = new CraftskyAPI(fetcher);
		await expect(
			api.posts.like('did:plc:a', 'post', key)
		).rejects.toMatchObject({ operationKey: key });
		await expect(api.profiles.me()).rejects.toMatchObject({
			code: 'network_unavailable',
			status: 0
		});
	});
	it('rejects malformed successful delete responses because their write outcome is unconfirmed', async () => {
		const { api } = setup({}, 200);
		await expect(
			api.posts.delete('did:plc:a', 'post', 'cid', key)
		).rejects.toBeInstanceOf(MutationOutcomeUnknownError);
	});
	it('accepts an exact empty 204 write response', async () => {
		const { api } = setup(undefined, 204);
		await expect(
			api.posts.delete('did:plc:a', 'post', 'cid', key)
		).resolves.toBeUndefined();
	});
	it('rejects invalid operation IDs before contacting the server', async () => {
		const { api, fetcher } = setup();
		await expect(
			api.posts.like('did:plc:a', 'post', key.toUpperCase())
		).rejects.toBeInstanceOf(TypeError);
		expect(fetcher).not.toHaveBeenCalled();
	});
	it('rejects malformed pagination rather than allowing a broken infinite-scroll state', async () => {
		const { api } = setup({ items: [], cursor: 123 });
		await expect(api.posts.timeline()).rejects.toBeInstanceOf(APIError);
	});
	it('rejects cross-origin transport paths and never follows redirects', () => {
		expect(() => new APITransport(fetch, 'https://other.example')).toThrow(
			TypeError
		);
		expect(() => new APITransport(fetch, '//other.example')).toThrow(TypeError);
	});
	it('retains cancellation for reads and marks interrupted writes as unconfirmed', async () => {
		const fetcher = vi.fn<typeof fetch>(async () => {
			throw new DOMException('Cancelled', 'AbortError');
		});
		const api = new CraftskyAPI(fetcher);
		const controller = new AbortController();
		controller.abort();
		await expect(
			api.posts.timeline({ signal: controller.signal })
		).rejects.toMatchObject({ name: 'AbortError' });
		expect(fetcher).not.toHaveBeenCalled();
		await expect(
			api.posts.like('did:plc:a', 'post', key)
		).rejects.toBeInstanceOf(MutationOutcomeUnknownError);
	});
});

it('keeps video finalisation alive beyond ordinary request deadlines but bounded to seven minutes', async () => {
	vi.useFakeTimers();
	try {
		const { api, fetcher, request } = setup();
		fetcher.mockImplementationOnce(async () => new Promise(() => {}));
		let settled = false;
		const task = api.media.videoStatus('aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa');
		const result = expect(task).rejects.toMatchObject({ name: 'TimeoutError' });
		void task.then(
			() => {
				settled = true;
			},
			() => {
				settled = true;
			}
		);
		await vi.advanceTimersByTimeAsync(120001);
		expect(settled).toBe(false);
		expect(request().signal?.aborted).toBe(false);
		await vi.advanceTimersByTimeAsync(300000);
		await result;
		expect(request().signal?.aborted).toBe(true);
	} finally {
		vi.useRealTimers();
	}
});
