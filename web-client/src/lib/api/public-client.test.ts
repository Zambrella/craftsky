import { it, expect, vi } from 'vitest';
import { PublicCraftskyAPI } from './public-client';
const wire = {
	uri: 'at://did:plc:alice/social.craftsky.feed.post/first',
	cid: 'post-cid',
	rkey: 'first',
	text: 'Public text',
	author: { did: 'did:plc:alice', handle: 'alice.test' },
	createdAt: '2026-11-06T10:00:00Z',
	images: [],
	tags: [],
	langs: [],
	sponsored: false,
	likeCount: 1,
	replyCount: 1,
	repostCount: 0,
	quoteCount: 0
};

// UT-002, IT-008 / RULE-001, FR-006 / AC-006.
it('reads the explicit public contract with no credentials or private defaults on the wire', async () => {
	const fetcher = vi
		.fn<typeof fetch>()
		.mockResolvedValue(Response.json({ items: [wire] }));
	const api = new PublicCraftskyAPI(fetcher);
	const page = await api.search.posts({ q: 'knitting', limit: 20 });
	expect(api.audience).toBe('guest');
	expect(String(fetcher.mock.calls[0][0])).toBe(
		'/api/public/search/posts?q=knitting&limit=20'
	);
	expect(fetcher.mock.calls[0][1]?.credentials).toBe('omit');
	expect(fetcher.mock.calls[0][1]?.headers).toEqual({
		accept: 'application/json'
	});
	expect(page.items[0]).toMatchObject({
		text: 'Public text',
		viewerHasLiked: false,
		viewerHasSaved: false
	});
});
it('submits only post and profile reports without credentials', async () => {
	const fetcher = vi.fn<typeof fetch>(async () =>
		Response.json(
			{ reportId: 'guest-report', status: 'accepted' },
			{ status: 201 }
		)
	);
	const api = new PublicCraftskyAPI(fetcher);
	const submission = {
		reasonType: 'privacy',
		details: 'Private details',
		contactEmail: 'visitor@example.test'
	};
	await expect(
		api.posts.report('did:plc:alice', 'first', submission)
	).resolves.toMatchObject({ status: 'accepted' });
	await expect(
		api.profiles.report('did:plc:alice', submission)
	).resolves.toMatchObject({ status: 'accepted' });
	expect(fetcher.mock.calls.map(([path]) => String(path))).toEqual([
		'/api/public/posts/did%3Aplc%3Aalice/first/reports',
		'/api/public/profiles/%40did%3Aplc%3Aalice/reports'
	]);
	for (const [, options] of fetcher.mock.calls) {
		expect(options).toMatchObject({
			method: 'POST',
			credentials: 'omit',
			redirect: 'manual'
		});
		expect(new Headers(options?.headers).has('authorization')).toBe(false);
		expect(JSON.parse(String(options?.body))).toEqual(submission);
	}
});
it('rejects all mutations, private reads and discovery before fetch', async () => {
	const fetcher = vi.fn<typeof fetch>();
	const api = new PublicCraftskyAPI(fetcher);
	for (const call of [
		() =>
			api.posts.like(
				'did:plc:alice',
				'first',
				'018f4d5c-7a61-7d40-a1a2-777777777777'
			),
		() => api.profiles.me(),
		() => api.profiles.recent(),
		() => api.search.suggestions('knit'),
		() => api.search.topHashtags(),
		() => api.search.recent()
	]) {
		await expect(call()).rejects.toMatchObject({
			code: 'authentication_required'
		});
	}
	expect(fetcher).not.toHaveBeenCalled();
});
it('adapts public thread sections for the shared reading UI', async () => {
	const fetcher = vi.fn<typeof fetch>().mockResolvedValue(
		Response.json({
			post: wire,
			ancestors: [{ state: 'unavailable' }],
			replies: {
				items: [{ ...wire, rkey: 'reply', uri: wire.uri + 'reply' }],
				cursor: 'next'
			}
		})
	);
	const result = await new PublicCraftskyAPI(fetcher).posts.comments(
		'did:plc:alice',
		'first',
		{ sort: 'oldest', limit: 20 }
	);
	expect(String(fetcher.mock.calls[0][0])).toBe(
		'/api/public/posts/did%3Aplc%3Aalice/first/thread?limit=20'
	);
	expect(result.comments.items[0].post).toMatchObject({ rkey: 'reply' });
	expect(result.comments.cursor).toBe('next');
});

it('AT-005 reads a guarded public WebVTT caption without JSON adaptation', async () => {
	const seen: string[] = [];
	const api = new PublicCraftskyAPI(async (input) => {
		seen.push(String(input));
		return new Response('WEBVTT\n\n00:00.000 --> 00:01.000\nPublic caption\n', {
			headers: { 'content-type': 'text/vtt' }
		});
	});
	const caption = await api.media.caption(
		'did:plc:alice',
		'post',
		'caption-cid'
	);
	expect(caption).toContain('WEBVTT');
	expect(seen).toEqual([
		'/api/public/posts/did%3Aplc%3Aalice/post/video-captions/caption-cid'
	]);
});
