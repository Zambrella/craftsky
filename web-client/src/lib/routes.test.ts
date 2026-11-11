import { expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { authenticatedRoute, guestRoute } from './routes';
import { returnPath } from './server/auth/session';

it('uses the mobile route locations for shared destinations', () => {
	const mobile = readFileSync('../app/lib/router/route_locations.dart', 'utf8');
	for (const name of [
		'feed',
		'projects',
		'search',
		'notifications',
		'messages',
		'profile',
		'settings',
		'drafts'
	]) {
		const path = new RegExp(`static const ${name} = '([^']+)'`).exec(
			mobile
		)![1];
		expect(authenticatedRoute(path), name).toBe(true);
		expect(returnPath(path), name).toBe(path);
	}
	for (const path of [
		'/search/tags?tag=sewing',
		'/notifications/settings',
		'/messages/conversations/one-two',
		'/profiles/@alice.test',
		'/posts/did%3Aplc%3Aalice/first/likes',
		'/saved/folder?folder=123'
	])
		expect(returnPath(path)).toBe(path);
});

it('rejects obsolete routes and non-app auth return destinations', () => {
	for (const path of [
		'/app/home',
		'/home',
		'/activity',
		'/tags/sewing',
		'/messages/123',
		'/sign-in',
		'/api/auth/logout',
		'/auth/browser/confirm',
		'//evil.test/feed',
		'/feed/../../api/auth/logout',
		'/feed/%2e%2e/evil'
	])
		expect(returnPath(path)).toBe('/feed');
	for (const path of [
		'/app/home',
		'/unknown',
		'/messages/123',
		'/home',
		'/activity'
	])
		expect(authenticatedRoute(path)).toBe(false);
});

it('uses SvelteKit matchers to leave OAuth, APIs and unknown paths outside the app catch-all', async () => {
	const { params } = await import('../params');
	for (const path of [
		'feed',
		'notifications/settings',
		'profiles/@alice.test',
		'messages/conversations/example',
		'settings/messaging/'
	])
		expect(await params.authenticated['~standard'].validate(path)).toEqual({
			value: path
		});
	for (const path of [
		'auth/browser/confirm',
		'api/auth/logout',
		'unknown',
		'app/home'
	])
		expect(
			(await params.authenticated['~standard'].validate(path)).issues
		).toBeDefined();
});

it('profile follower lists are member-only app destinations', () => {
	for (const kind of ['followers', 'following']) {
		const path = '/profiles/did%3Aplc%3Abob/' + kind;
		expect(authenticatedRoute(path)).toBe(true);
		expect(guestRoute(path)).toBe(false);
		expect(returnPath(path)).toBe(path);
	}
});
