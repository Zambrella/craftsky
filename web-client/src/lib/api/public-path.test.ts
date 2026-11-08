import { describe, it, expect } from 'vitest';
import { publicReadPath } from './public-path';

// IT-008 / RULE-001, FR-006 / AC-006: browser proxy and client share a
// closed read allowlist; arbitrary member routes never reach the upstream.
describe('public read route and query allowlist', () => {
	it.each([
		'projects',
		'search/profiles',
		'search/posts',
		'search/projects',
		'search/hashtags',
		'search/hashtags/needle/posts',
		'profiles/did:plc:alice',
		'profiles/alice.test/comments',
		'profiles/alice.test/reposts',
		'profiles/alice.test/events',
		'posts/did:plc:alice/first',
		'posts/did:plc:alice/first/thread',
		'posts/did:plc:alice/first/likes',
		'events/did:plc:alice/first'
	])('permits %s', (path) => {
		const query = new URLSearchParams(
			/^search\/(profiles|posts|projects|hashtags)$/.test(path)
				? 'q=knitting&limit=20'
				: ''
		);
		expect(publicReadPath(path, query)).toBeTruthy();
	});
	it.each([
		'profiles/recent',
		'profiles/me',
		'search/suggestions',
		'search/recent',
		'search/hashtags/top',
		'notifications',
		'saved',
		'whoami',
		'auth/logout',
		'profiles/alice.test/mutual-followers',
		'../whoami',
		'profiles/%2e%2e',
		'profiles/%2fme',
		'https://private.example',
		'posts/did:plc:alice/../edit'
	])('rejects %s', (path) =>
		expect(() => publicReadPath(path, new URLSearchParams())).toThrow()
	);
	it.each([
		'q=',
		'q=knit&sort=popular',
		'q=knit&craftType=knitting',
		'q=knit&viewerDid=did:plc:owner',
		'q=knit&limit=21',
		'q=knit&limit=0',
		'q=knit&limit=1&limit=2',
		'q=knit&cursor=' + 'x'.repeat(2049)
	])('rejects unsupported search query %s', (query) => {
		expect(() =>
			publicReadPath('search/projects', new URLSearchParams(query))
		).toThrow();
	});
	it('retains existing craft-feed filters and sort', () => {
		const path = publicReadPath(
			'projects',
			new URLSearchParams(
				'sort=popular&craftType=knitting&status=wip&status=finished'
			)
		);
		expect(path).toContain('sort=popular');
		expect(path).toContain('status=wip');
		expect(path).toContain('status=finished');
	});
});
