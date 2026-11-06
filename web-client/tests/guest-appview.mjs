import { createServer } from 'node:http';
/** @type {string[]} */
const requests = [];
let refreshed = false;
const author = {
	did: 'did:plc:alice',
	handle: 'alice.test',
	displayName: 'Alice',
	accountType: 'regular',
	business: false,
	customisation: { colour: 'cobalt', profileBackground: 'none' }
};
const post = {
	uri: 'at://did:plc:alice/social.craftsky.feed.post/first',
	cid: 'post-cid',
	rkey: 'first',
	text: 'Guest knitting project',
	author,
	createdAt: '2026-11-06T10:00:00Z',
	images: [],
	tags: ['knitting'],
	langs: ['en'],
	sponsored: false,
	likeCount: 2,
	repostCount: 0,
	quoteCount: 0,
	replyCount: 1,
	project: {
		common: {
			craftType: 'social.craftsky.feed.defs#knitting',
			title: 'My knitted scarf'
		}
	}
};
const reply = {
	...post,
	uri: 'at://did:plc:bob/social.craftsky.feed.post/reply',
	rkey: 'reply',
	project: undefined,
	author: {
		...author,
		did: 'did:plc:bob',
		handle: 'bob.test',
		displayName: 'Bob'
	},
	text: 'A public comment',
	reply: {
		root: {
			uri: 'at://did:plc:optout/social.craftsky.feed.post/hidden',
			cid: 'hidden-cid'
		},
		parent: {
			uri: 'at://did:plc:optout/social.craftsky.feed.post/hidden',
			cid: 'hidden-cid'
		}
	},
	replyCount: 0
};
createServer((req, res) => {
	const url = new URL(req.url || '/', 'http://localhost');
	/** @param {number} status @param {unknown} value */
	const send = (status, value) => {
		res.writeHead(status, {
			'content-type': 'application/json',
			'cache-control': 'no-store'
		});
		res.end(JSON.stringify(value));
	};
	if (url.pathname === '/_profile-refresh/reset') {
		refreshed = false;
		return send(200, {});
	}
	if (url.pathname === '/_profile-refresh/advance') {
		refreshed = true;
		return send(200, {});
	}
	if (url.pathname === '/_requests') return send(200, requests);
	requests.push(url.pathname + url.search);
	if (req.headers.authorization || req.headers.cookie)
		return send(500, { error: 'credential_forwarded' });
	if (url.pathname.includes('optout'))
		return send(403, {
			error: 'authentication_required',
			message: 'PRIVATE_CANARY'
		});
	if (url.pathname.includes('unknown'))
		return send(404, { error: 'not_found', message: 'PRIVATE_CANARY' });
	if (!url.pathname.startsWith('/v1/public/'))
		return send(401, { error: 'authentication_required' });
	const path = decodeURIComponent(
		url.pathname.slice('/v1/public/'.length)
	).replace(/^profiles\/@/, 'profiles/');
	if (/^profiles\/(refresh.test|did:plc:refresh)(?:\/posts)?$/.test(path)) {
		const identity = {
			...author,
			did: 'did:plc:refresh',
			handle: 'refresh.test',
			displayName: refreshed ? 'Updated reader' : 'Original reader'
		};
		if (!path.endsWith('/posts'))
			return send(200, {
				...identity,
				crafts: [],
				links: [],
				postCount: 1,
				projectCount: 0
			});
		if (url.searchParams.get('cursor')) return send(200, { items: [] });
		return send(200, {
			items: [
				{
					...post,
					author: identity,
					project: undefined,
					uri: 'at://did:plc:refresh/social.craftsky.feed.post/fresh',
					rkey: 'fresh',
					text: refreshed ? 'Updated safe body' : 'Original public body',
					quoteView: refreshed
						? { state: 'unavailable' }
						: {
								state: 'visible',
								post: {
									...reply,
									text: 'EXCLUDED_AFTER_REFRESH',
									reply: undefined
								}
							}
				}
			],
			cursor: refreshed ? 'fresh-cursor' : 'old-cursor'
		});
	}
	if (
		path === 'profiles/did:plc:alice/reposts' ||
		path === 'profiles/alice.test/reposts' ||
		/^profiles\/(did:plc:bob|bob.test)\/(posts|projects|reposts)$/.test(path)
	)
		return send(200, {
			items: [reply],
			cursor:
				path.endsWith('/reposts') && !url.searchParams.get('cursor')
					? 'profile-reposts-next'
					: undefined
		});
	if (
		/^profiles\/(did:plc:alice|alice.test)\/posts$/.test(path) &&
		url.searchParams.get('cursor') === 'profile-page-two'
	)
		return send(200, {
			items: [
				{
					...post,
					text: 'Second public profile page',
					rkey: 'second',
					uri: 'at://did:plc:alice/social.craftsky.feed.post/second'
				}
			]
		});
	if (
		path === 'projects' ||
		/^search\/(posts|projects)$/.test(path) ||
		/^profiles\/[^/]+\/(posts|projects|reposts)$/.test(path)
	)
		return send(200, { items: [post] });
	if (path === 'search/profiles')
		return send(200, {
			items: [
				{
					...author,
					crafts: ['knitting'],
					links: [],
					postCount: 1,
					projectCount: 1
				}
			]
		});
	if (path === 'search/hashtags')
		return send(200, { items: [{ tag: 'knitting', postCount: 1 }] });
	if (/^search\/hashtags\/[^/]+\/posts$/.test(path))
		return send(200, { items: [post] });
	if (/^profiles\/[^/]+\/comments$/.test(path))
		return send(200, { items: [reply] });
	const business = {
		...author,
		did: 'did:plc:business',
		handle: 'business.test',
		business: true,
		accountType: 'business',
		hasUpcomingEvents: true,
		businessProfile: {
			tagline: 'Public shop details',
			businessTypes: [],
			products: [
				{ id: 'kit', title: 'Public wool kit', uri: 'https://example.com/kit' }
			]
		}
	};
	if (path === 'profiles/business.test')
		return send(200, {
			...business,
			crafts: ['knitting'],
			links: [],
			description: 'Public shop details'
		});
	if (path === 'profiles/business.test/events')
		return send(200, {
			items: [
				{
					did: business.did,
					rkey: 'fair',
					uri: 'at://did:plc:business/social.craftsky.business.event/fair',
					cid: 'fair-cid',
					author: business,
					name: 'Public craft fair',
					startsAt: '2027-01-01T10:00:00Z',
					endsAt: '2027-01-01T18:00:00Z',
					roles: [],
					status: { value: 'scheduled' },
					isAllDay: false,
					createdAt: '2026-11-06T00:00:00Z',
					past: false
				}
			]
		});
	if (/^profiles\/[^/]+\/events$/.test(path)) return send(200, { items: [] });
	if (/^profiles\/[^/]+$/.test(path))
		return send(200, {
			...(path === 'profiles/did:plc:bob' || path === 'profiles/bob.test'
				? reply.author
				: author),
			crafts: ['knitting'],
			links: [],
			postCount: 1,
			projectCount: 1,
			followerCount: 2,
			followingCount: 1,
			description: 'Making things in public.'
		});
	if (path === 'posts/did:plc:bob/reply/thread')
		return send(200, {
			post: reply,
			ancestors: [{ state: 'unavailable' }],
			replies: { items: [] }
		});
	if (path === 'posts/did:plc:bob/reply') return send(200, reply);
	if (/^posts\/[^/]+\/[^/]+\/thread$/.test(path))
		return send(200, { post, ancestors: [], replies: { items: [reply] } });
	if (/^posts\/[^/]+\/[^/]+\/(likes|reposts)$/.test(path))
		return send(200, { items: [author] });
	if (/^posts\/[^/]+\/[^/]+\/quotes$/.test(path))
		return send(200, { items: [] });
	if (path === 'posts/did:plc:alice/navigation')
		return send(200, {
			...post,
			rkey: 'navigation',
			uri: 'at://did:plc:alice/social.craftsky.feed.post/navigation',
			quoteView: { state: 'visible', post: reply }
		});
	if (/^posts\/[^/]+\/[^/]+$/.test(path)) return send(200, post);
	return send(404, { error: 'not_found' });
}).listen(19851, '127.0.0.1');
