import { createServer } from 'node:http';
const author = {
	did: 'did:plc:alice',
	handle: 'alice.test',
	displayName: 'Alice <script>unsafe</script>',
	avatar: 'https://cdn.example/avatar.jpg'
};
const post = {
	rkey: 'first',
	text: 'A linen dress <script>alert("unsafe")</script>',
	author,
	createdAt: '2026-10-01T12:00:00Z',
	images: [
		{
			fullsize: 'https://cdn.example/dress.jpg',
			alt: 'A handmade blue linen dress',
			aspectRatio: { width: 800, height: 1000 }
		}
	],
	sponsored: true,
	project: {
		common: {
			title: 'My linen dress',
			craftType: 'social.craftsky.feed.defs#sewing',
			status: 'social.craftsky.feed.defs#finished',
			materials: [{ text: 'Blue linen' }],
			pattern: { name: 'Summer dress', url: 'javascript:alert(1)' }
		},
		details: { projectType: 'social.craftsky.project.defs#garment' }
	}
};
createServer((req, res) => {
	const url = new URL(req.url || '/', 'http://localhost');
	/** @param {number} status @param {unknown} value */
	const send = (status, value) => {
		res.writeHead(status, { 'content-type': 'application/json' });
		res.end(JSON.stringify(value));
	};
	if (req.headers.authorization || req.headers.cookie)
		return send(500, { message: 'credentials forwarded' });
	if (url.pathname.includes('missing'))
		return send(404, { message: 'PRIVATE_CANARY' });
	if (url.pathname.includes('failed'))
		return send(500, { message: 'PRIVATE_CANARY' });
	if (url.pathname.endsWith('/posts') || url.pathname.endsWith('/projects'))
		return send(200, {
			items: url.searchParams.get('cursor')
				? [{ ...post, rkey: 'older', text: 'An older project' }]
				: [post],
			cursor: url.searchParams.get('cursor') ? undefined : 'next-page'
		});
	if (url.pathname.startsWith('/v1/public/profiles/'))
		return send(200, {
			...author,
			avatar: url.pathname.endsWith('/no-avatar.test')
				? undefined
				: url.pathname.endsWith('/unsafe-avatar.test')
					? 'javascript:alert(1)'
					: author.avatar,
			description: 'Making things, one stitch at a time.',
			crafts: ['sewing'],
			links: [
				{ title: 'My shop', url: 'https://example.com' },
				{ title: 'Bad link', url: 'javascript:alert(1)' }
			],
			business: true
		});
	if (url.pathname.startsWith('/v1/public/posts/')) return send(200, post);
	return send(404, {});
}).listen(19841, '127.0.0.1');
