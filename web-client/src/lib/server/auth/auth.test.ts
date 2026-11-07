import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { isRedirect, type Cookies } from '@sveltejs/kit';
import { Buffer } from 'node:buffer';
import { POST as start } from '../../../routes/api/auth/start/+server';
import { POST as switchAccount } from '../../../routes/api/auth/switch/+server';
import { POST as logout } from '../../../routes/api/auth/logout/+server';
import { GET as complete } from '../../../routes/auth/browser/complete/+server';
import { GET as confirm } from '../../../routes/auth/browser/confirm/+server';
import { GET as session } from '../../../routes/api/session/+server';
import {
	GET as proxyGet,
	POST as proxyPost
} from '../../../routes/api/appview/[...path]/+server';
import { POST as deletionStart } from '../../../routes/api/auth/deletion/start/+server';
import { GET as deletionComplete } from '../../../routes/account-deletion/browser/reauth-complete/+server';
import { POST as deletionConfirm } from '../../../routes/api/auth/deletion/confirm/+server';
import { GET as deletionStatus } from '../../../routes/api/auth/deletion/status/+server';
import {
	store,
	encode,
	decode,
	credentials,
	selected,
	options,
	prefix,
	readPending,
	readIntent,
	handoffResult,
	returnPath,
	device,
	sameState,
	type Credential,
	type Intent,
	type Pending
} from './session';
import { allowedRoute, safeProxyPath } from './routes';
import { upstream, UpstreamError, safeFailure } from './upstream';

const origin = 'https://app.craftsky.social';
const accountA = '10000000-0000-4000-8000-000000000001';
const accountB = '10000000-0000-4000-8000-000000000002';
const receiptID = '10000000-0000-4000-8000-000000000003';
const jobID = '10000000-0000-4000-8000-000000000004';
const token = (byte: number) =>
	Buffer.from(new Uint8Array(32).fill(byte)).toString('base64url');
const A: Credential = {
	id: accountA,
	did: 'did:plc:alice',
	handle: 'alice.example',
	token: token(1)
};
const B: Credential = {
	id: accountB,
	did: 'did:plc:bob',
	handle: 'bob.example',
	token: token(2)
};
const capabilities = token(3);
const response = (body: unknown = {}, status = 200, headers?: HeadersInit) =>
	new Response(status === 204 ? null : JSON.stringify(body), {
		status,
		headers: {
			'Content-Type': 'application/json',
			...Object.fromEntries(new Headers(headers))
		}
	});
class CookieJar {
	values = new Map<string, string>();
	changes: { name: string; value?: string; options: unknown }[] = [];
	cookies = {
		get: (name: string) => this.values.get(name),
		getAll: () => [...this.values].map(([name, value]) => ({ name, value })),
		set: (name: string, value: string, options: unknown) => {
			this.values.set(name, value);
			this.changes.push({ name, value, options });
		},
		delete: (name: string, options: unknown) => {
			this.values.delete(name);
			this.changes.push({ name, options });
		}
	} as unknown as Cookies;
	add(value: Credential) {
		this.values.set('__Host-craftsky_account_' + value.id, encode(value));
		return this;
	}
	active(id: string) {
		this.values.set('__Host-craftsky_active', id);
		return this;
	}
}
interface TestEvent {
	url: URL;
	request: Request;
	cookies: Cookies;
	params: { path?: string };
}
function event(
	jar: CookieJar,
	path: string,
	method = 'GET',
	body?: unknown,
	headers: HeadersInit = {}
): TestEvent {
	const url = new URL(path, origin);
	const request = new Request(url, {
		method,
		headers: {
			...(method !== 'GET'
				? { origin, 'Content-Type': 'application/json' }
				: {}),
			...Object.fromEntries(new Headers(headers))
		},
		...(body !== undefined
			? { body: typeof body === 'string' ? body : JSON.stringify(body) }
			: {})
	});
	return {
		url,
		request,
		cookies: jar.cookies,
		params: {
			path: path.startsWith('/api/appview/')
				? path.slice('/api/appview/'.length).split('?')[0]
				: undefined
		}
	};
}
async function call(handler: unknown, value: TestEvent): Promise<Response> {
	return await (handler as (event: TestEvent) => Promise<Response> | Response)(
		value
	);
}
async function location(handler: unknown, value: TestEvent): Promise<string> {
	try {
		await call(handler, value);
	} catch (error) {
		if (isRedirect(error)) {
			expect(error.status).toBe(303);
			return error.location;
		}
		throw error;
	}
	throw new Error('Expected a redirect');
}
function intent(kind: Intent['kind'] = 'login'): Intent {
	return {
		state: capabilities,
		active: accountA,
		kind,
		returnTo: '/messages',
		expiresAt: Date.now() + 300000,
		...(kind === 'chat' ? { source: accountA } : {})
	};
}
function handoff() {
	return {
		token: token(4),
		did: A.did,
		handle: A.handle,
		receiptId: receiptID,
		confirmBy: new Date(Date.now() + 300000).toISOString()
	};
}
function pending(kind: Intent['kind'] = 'login'): Pending {
	return { ...handoff(), active: accountA, kind, returnTo: '/messages' };
}
let fetcher: ReturnType<typeof vi.fn<typeof fetch>>;
let oldOrigin: string | undefined;
beforeEach(() => {
	oldOrigin = process.env.APPVIEW_ORIGIN;
	process.env.APPVIEW_ORIGIN = 'https://appview.example.test';
	fetcher = vi.fn<typeof fetch>(async (url) =>
		String(url).endsWith('/v1/whoami')
			? response({ did: A.did, handle: A.handle })
			: response()
	);
	vi.stubGlobal('fetch', fetcher);
});
afterEach(() => {
	if (oldOrigin === undefined) delete process.env.APPVIEW_ORIGIN;
	else process.env.APPVIEW_ORIGIN = oldOrigin;
	vi.unstubAllGlobals();
	vi.restoreAllMocks();
});

describe('browser cookies and untrusted metadata', () => {
	it('uses host-only secure HttpOnly SameSite=Lax cookies and stable random device identity', () => {
		const jar = new CookieJar(),
			url = new URL(origin);
		const first = device(jar.cookies, url);
		expect(device(jar.cookies, url)).toBe(first);
		expect(jar.changes).toHaveLength(1);
		expect(prefix(url)).toBe('__Host-craftsky_');
		expect(options(url)).toMatchObject({
			path: '/',
			httpOnly: true,
			secure: true,
			sameSite: 'lax'
		});
		expect(options(url)).not.toHaveProperty('domain');
		expect(prefix(new URL('http://127.0.0.1:1234'))).toBe('craftsky_');
	});
	it('rejects malformed, oversized, extra-field and mismatched account cookies without token exposure', async () => {
		const jar = new CookieJar().add(A).active(A.id);
		jar.values.set(
			'__Host-craftsky_account_' + B.id,
			encode({ ...B, id: A.id })
		);
		jar.values.set(
			'__Host-craftsky_account_bad',
			encode({ ...B, oauthRefreshToken: 'PRIVATE_CANARY' })
		);
		expect(credentials(jar.cookies, new URL(origin))).toEqual([A]);
		expect(decode('[')).toBeUndefined();
		expect(decode('a'.repeat(4001))).toBeUndefined();
		const body = await (await call(session, event(jar, '/api/session'))).text();
		expect(body).toContain(A.did);
		expect(body).not.toContain(A.token);
		expect(body).not.toContain('PRIVATE_CANARY');
		expect(body).not.toContain('token');
	});
	it('never falls back to another account for an invalid explicit selection', () => {
		const jar = new CookieJar().add(A).add(B).active(B.id);
		expect(selected(jar.cookies, new URL(origin), 'unknown')).toBeUndefined();
		expect(selected(jar.cookies, new URL(origin), '')).toBeUndefined();
		expect(selected(jar.cookies, new URL(origin), A.id)).toEqual(A);
	});
	it('normalizes return paths and rejects external, encoded traversal and control targets', () => {
		for (const raw of [
			'https://evil.test',
			'//evil.test',
			'/app/../../evil',
			'/app/\\evil',
			'/app/%2e%2e/evil',
			'/app/%5cevil',
			'/feed\r\nLocation:x'
		])
			expect(returnPath(raw)).toBe('/feed');
		expect(returnPath('/messages?from=profile')).toBe('/messages?from=profile');
	});
	it('rejects tampered intent/pending shape, purpose, deadlines and capability encodings', () => {
		expect(readIntent(encode(intent()))).toBeDefined();
		expect(readPending(encode(pending()))).toBeDefined();
		for (const value of [
			{ ...intent(), expiresAt: Date.now() - 1 },
			{ ...intent(), source: accountB },
			{ ...intent(), returnTo: 'https://evil.test' },
			{ ...intent(), state: 'x'.repeat(43) },
			{ ...intent(), extra: 'unexpected' }
		])
			expect(readIntent(encode(value))).toBeUndefined();
		for (const value of [
			{ ...pending(), kind: 'chat' },
			{ ...pending(), returnTo: '//evil.test' },
			{ ...pending(), token: 'synthetic' },
			{ ...pending(), confirmBy: new Date(Date.now() - 1).toISOString() },
			{ ...pending(), oauthAccessToken: 'PRIVATE_CANARY' }
		])
			expect(readPending(encode(value))).toBeUndefined();
		expect(sameState(capabilities, capabilities)).toBe(true);
		expect(sameState(capabilities, token(7))).toBe(false);
	});
});

describe('login and durable browser confirmation', () => {
	it.each(['login', 'chat'] as const)(
		'accepts enriched %s handoffs and retains only validated authentication fields',
		(kind) => {
			const expected = {
				...handoff(),
				...(kind === 'chat'
					? { purpose: 'chat' as const, sourceTokenHash: token(5) }
					: {})
			};
			const enriched = {
				...expected,
				accountType: 'personal',
				crafts: ['social.craftsky.feed.defs#sewing'],
				customisation: { colour: 'cobalt', profileBackground: 'none' },
				active: accountB,
				returnTo: 'https://evil.test',
				kind: 'register',
				oauthAccessToken: 'PRIVATE_CANARY'
			};
			expect(handoffResult(enriched)).toEqual(expected);
			expect(handoffResult({ ...enriched, token: 'invalid' })).toBeUndefined();
			expect(
				handoffResult({ ...enriched, purpose: 'unknown' })
			).toBeUndefined();
			// Upstream additions are ignored; browser cookie shapes stay strict.
			expect(
				readPending(encode({ ...pending(), accountType: 'personal' }))
			).toBeUndefined();
		}
	);
	it('completes an AppView identity-enriched exchange without copying profile or control fields into the pending cookie', async () => {
		const jar = new CookieJar();
		const started = intent(),
			exchanged = handoff();
		jar.values.set('__Host-craftsky_intent', encode(started));
		fetcher.mockResolvedValueOnce(
			response({
				...exchanged,
				accountType: 'personal',
				crafts: [],
				customisation: { colour: 'cobalt', profileBackground: 'none' },
				active: accountB,
				returnTo: 'https://evil.test',
				kind: 'chat',
				oauthAccessToken: 'PRIVATE_CANARY'
			})
		);
		expect(
			await location(
				complete,
				event(
					jar,
					`/auth/browser/complete?state=${capabilities}&code=${token(9)}`
				)
			)
		).toBe('/auth/browser/confirm');
		const saved = jar.values.get('__Host-craftsky_pending');
		expect(readPending(saved)).toEqual({
			...exchanged,
			active: started.active,
			kind: started.kind,
			returnTo: started.returnTo
		});
		expect(JSON.stringify(decode(saved))).not.toContain('PRIVATE_CANARY');
	});
	it('creates an exact nonce-bound browser callback and emits no token', async () => {
		const jar = new CookieJar().add(A).active(A.id);
		fetcher.mockResolvedValue(
			response({
				authUrl: 'https://provider.example.test/authorize?request_uri=example'
			})
		);
		const result = await call(
			start,
			event(jar, '/api/auth/start', 'POST', {
				kind: 'login',
				handle: 'alice.example',
				returnTo: '/messages'
			})
		);
		expect(result.status).toBe(200);
		const body = JSON.parse(String(fetcher.mock.calls[0][1]?.body));
		const callback = new URL(body.browserRedirectUri);
		const saved = readIntent(jar.values.get('__Host-craftsky_intent'))!;
		expect(callback.origin).toBe(origin);
		expect(callback.pathname).toBe('/auth/browser/complete');
		expect(callback.searchParams.get('state')).toBe(saved.state);
		expect(body.handoffMode).toBe('browser');
		expect(await result.text()).not.toContain(A.token);
	});
	it('rejects cross-site, malformed JSON, extra fields and unknown explicit accounts before contacting AppView', async () => {
		const jar = new CookieJar().add(A);
		const requests = [
			event(
				jar,
				'/api/auth/start',
				'POST',
				{ kind: 'login', handle: 'alice.example' },
				{ origin: 'https://evil.test' }
			),
			event(jar, '/api/auth/start', 'POST', '{broken'),
			event(jar, '/api/auth/start', 'POST', {
				kind: 'chat',
				returnUrl: 'https://evil.test'
			}),
			event(
				jar,
				'/api/auth/start',
				'POST',
				{ kind: 'chat' },
				{ 'X-Craftsky-Account-Id': accountB }
			)
		];
		for (const request of requests)
			expect((await call(start, request)).status).toBeGreaterThanOrEqual(400);
		expect(fetcher).not.toHaveBeenCalled();
	});
	it('uses the captured account for messaging authorization when another account is selected', async () => {
		const jar = new CookieJar().add(A).add(B).active(B.id);
		fetcher.mockResolvedValue(
			response({ authUrl: 'https://provider.example.test/authorize' })
		);
		expect(
			(
				await call(
					start,
					event(
						jar,
						'/api/auth/start',
						'POST',
						{ kind: 'chat', returnTo: '/messages' },
						{ 'X-Craftsky-Account-Id': A.id }
					)
				)
			).status
		).toBe(200);
		const headers = new Headers(fetcher.mock.calls[0][1]?.headers);
		expect(headers.get('authorization')).toBe('Bearer ' + A.token);
		expect(readIntent(jar.values.get('__Host-craftsky_intent'))?.source).toBe(
			A.id
		);
		expect(
			JSON.parse(String(fetcher.mock.calls[0][1]?.body))
		).not.toHaveProperty('pushEnabled');
	});
	it('requires exact callback state and query structure before exchanging', async () => {
		for (const query of [
			`state=${token(8)}&code=${capabilities}`,
			`state=${capabilities}&state=${capabilities}&code=${capabilities}`,
			`state=${capabilities}&code=${capabilities}&next=https://evil.test`,
			`state=${capabilities}&code=${capabilities}&error=canceled`
		]) {
			const jar = new CookieJar();
			jar.values.set('__Host-craftsky_intent', encode(intent()));
			expect(
				await location(complete, event(jar, '/auth/browser/complete?' + query))
			).toMatch(/^\/sign-in/);
		}
		expect(fetcher).not.toHaveBeenCalled();
	});
	it('persists only a pending credential in the first response then activates on the clean confirm response', async () => {
		const jar = new CookieJar().add(A).active(A.id);
		jar.values.set('__Host-craftsky_intent', encode(intent()));
		fetcher
			.mockResolvedValueOnce(response(handoff()))
			.mockResolvedValueOnce(response(undefined, 204));
		expect(
			await location(
				complete,
				event(
					jar,
					`/auth/browser/complete?state=${capabilities}&code=${token(9)}`
				)
			)
		).toBe('/auth/browser/confirm');
		expect(fetcher).toHaveBeenCalledTimes(1);
		expect(credentials(jar.cookies, new URL(origin))[0].token).toBe(A.token);
		expect(readPending(jar.values.get('__Host-craftsky_pending'))?.token).toBe(
			token(4)
		);
		expect(await location(confirm, event(jar, '/auth/browser/confirm'))).toBe(
			'/messages'
		);
		expect(fetcher).toHaveBeenCalledTimes(3);
		expect(credentials(jar.cookies, new URL(origin))[0].token).toBe(token(4));
		expect(jar.values.has('__Host-craftsky_pending')).toBe(false);
		expect(
			new Headers(fetcher.mock.calls[1][1]?.headers).get('authorization')
		).toBe('Bearer ' + token(4));
	});
	it('retains a durable pending credential during a 503 and retries the same receipt and token', async () => {
		const jar = new CookieJar().add(A).active(A.id);
		jar.values.set('__Host-craftsky_pending', encode(pending()));
		fetcher
			.mockResolvedValueOnce(response({ error: 'unavailable' }, 503))
			.mockResolvedValueOnce(response(undefined, 204));
		expect(
			await location(confirm, event(jar, '/auth/browser/confirm'))
		).toContain('confirmation_pending');
		expect(jar.values.has('__Host-craftsky_pending')).toBe(true);
		expect(credentials(jar.cookies, new URL(origin))[0].token).toBe(A.token);
		expect(await location(confirm, event(jar, '/auth/browser/confirm'))).toBe(
			'/messages'
		);
		expect(fetcher.mock.calls[0][1]?.body).toBe(fetcher.mock.calls[1][1]?.body);
		expect(
			new Headers(fetcher.mock.calls[0][1]?.headers).get('authorization')
		).toBe(new Headers(fetcher.mock.calls[1][1]?.headers).get('authorization'));
	});
	it('preserves a newer active-account choice made while OAuth was in progress', async () => {
		const jar = new CookieJar().add(A).add(B).active(B.id);
		jar.values.set('__Host-craftsky_pending', encode(pending()));
		fetcher.mockResolvedValueOnce(response(undefined, 204));
		await location(confirm, event(jar, '/auth/browser/confirm'));
		expect(jar.values.get('__Host-craftsky_active')).toBe(B.id);
	});
	it('fails chat confirmation closed when the source bearer changed', async () => {
		const jar = new CookieJar().add(A).active(A.id);
		const hash = Buffer.from(
			await crypto.subtle.digest('SHA-256', new TextEncoder().encode(B.token))
		).toString('base64url');
		jar.values.set(
			'__Host-craftsky_pending',
			encode({
				...pending('chat'),
				purpose: 'chat',
				source: A.id,
				sourceTokenHash: hash
			})
		);
		expect(
			await location(confirm, event(jar, '/auth/browser/confirm'))
		).toContain('source_unavailable');
		expect(fetcher).not.toHaveBeenCalled();
		expect(jar.values.has('__Host-craftsky_pending')).toBe(false);
		expect(credentials(jar.cookies, new URL(origin))[0]).toEqual(A);
	});
	it('rejects ordinary exchange results for a messaging intent', async () => {
		const jar = new CookieJar().add(A).active(A.id);
		jar.values.set('__Host-craftsky_intent', encode(intent('chat')));
		fetcher.mockResolvedValue(response(handoff()));
		expect(
			await location(
				complete,
				event(
					jar,
					`/auth/browser/complete?state=${capabilities}&code=${token(9)}`
				)
			)
		).toContain('handoff_unavailable');
		expect(jar.values.has('__Host-craftsky_pending')).toBe(false);
	});
});

describe('account-specific proxy and session actions', () => {
	it('forwards only reviewed headers and exact account credentials, with no upstream Set-Cookie', async () => {
		const jar = new CookieJar().add(A).add(B).active(B.id);
		fetcher.mockResolvedValue(
			response({ ok: true }, 200, {
				'Set-Cookie': 'ATTACK=canary',
				'X-Request-ID': receiptID
			})
		);
		const request = event(
			jar,
			'/api/appview/v1/profiles/me',
			'GET',
			undefined,
			{
				'X-Craftsky-Account-Id': A.id,
				authorization: 'Bearer PRIVATE_CLIENT_TOKEN',
				'X-Dev-DID': 'did:plc:attacker',
				cookie: 'PRIVATE_COOKIE'
			}
		);
		const result = await call(proxyGet, request);
		const headers = new Headers(fetcher.mock.calls[0][1]?.headers);
		expect(headers.get('authorization')).toBe('Bearer ' + A.token);
		expect(headers.has('cookie')).toBe(false);
		expect(headers.has('X-Dev-DID')).toBe(false);
		expect(result.headers.has('Set-Cookie')).toBe(false);
		expect(result.headers.get('cache-control')).toBe('no-store');
		expect(result.headers.get('X-Request-ID')).toBe(receiptID);
		expect(fetcher.mock.calls[0][1]?.redirect).toBe('manual');
	});
	it('removes only the rejected account on 401 and retains every account on 503', async () => {
		const jar = new CookieJar().add(A).add(B).active(B.id);
		fetcher
			.mockResolvedValueOnce(response({ error: 'unavailable' }, 503))
			.mockResolvedValueOnce(response({ error: 'unauthorized' }, 401));
		const request = () =>
			event(jar, '/api/appview/v1/profiles/me', 'GET', undefined, {
				'X-Craftsky-Account-Id': A.id
			});
		expect((await call(proxyGet, request())).status).toBe(503);
		expect(credentials(jar.cookies, new URL(origin))).toHaveLength(2);
		expect((await call(proxyGet, request())).status).toBe(401);
		expect(credentials(jar.cookies, new URL(origin))).toEqual([B]);
		expect(jar.values.get('__Host-craftsky_active')).toBe(B.id);
	});
	it('blocks unsupported, auth-capability, double-encoded, traversal and URL-control routes', async () => {
		const jar = new CookieJar().add(A);
		for (const path of [
			'v1/auth/handoffs/exchange',
			'v1/chat/authorization',
			'v1/account-deletion/intents',
			'v1/profiles/%2e%2e',
			'v1/profiles/..',
			'v1/profiles/me?private=1',
			'v1/profiles/me\\evil',
			'dev/seed'
		]) {
			const request = event(jar, '/api/appview/v1/profiles/me', 'POST', {});
			request.params.path = path;
			expect((await call(proxyPost, request)).status).toBe(404);
		}
		expect(fetcher).not.toHaveBeenCalled();
		expect(allowedRoute('POST', '/v1/blobs/videos/authorization')).toBe(true);
		expect(allowedRoute('POST', `/v1/blobs/videos/${jobID}/status`)).toBe(true);
		expect(allowedRoute('POST', `/v1/blobs/videos/${jobID}/cancel`)).toBe(true);
		expect(allowedRoute('GET', `/v1/blobs/videos/${jobID}/cancel`)).toBe(false);
		expect(allowedRoute('GET', `/v1/blobs/videos/${jobID}/status`)).toBe(false);
		expect(allowedRoute('POST', '/v1/blobs/videos/not-a-job/status')).toBe(
			false
		);
		expect(allowedRoute('POST', '/v1/blobs/images/authorization')).toBe(true);
		expect(safeProxyPath('v1/profiles/@did:plc:alice')).toBe(
			'/v1/profiles/%40did%3Aplc%3Aalice'
		);
	});
	it('polls video status as the selected account without forwarding vendor capabilities', async () => {
		const jar = new CookieJar().add(A).add(B).active(A.id);
		fetcher.mockResolvedValueOnce(
			response({ jobStatus: { jobId: jobID, state: 'JOB_STATE_PROCESSING' } })
		);
		const request = event(
			jar,
			`/api/appview/v1/blobs/videos/${jobID}/status`,
			'POST',
			undefined,
			{
				'X-Craftsky-Account-Id': B.id,
				AuthorizationSignature: 'must-not-forward',
				AccessKey: 'must-not-forward'
			}
		);
		expect((await call(proxyPost, request)).status).toBe(200);
		const [url, options] = fetcher.mock.calls[0];
		expect(String(url)).toContain(`/v1/blobs/videos/${jobID}/status`);
		expect(options?.method).toBe('POST');
		const headers = new Headers(options?.headers);
		expect(headers.get('authorization')).toBe(`Bearer ${B.token}`);
		expect(headers.get('AuthorizationSignature')).toBeNull();
		expect(headers.get('AccessKey')).toBeNull();
	});
	it('forwards encoded did:web ports as identifier data with exactly one re-encoding', async () => {
		const jar = new CookieJar().add(A);
		const profile = event(jar, '/api/appview/v1/profiles/me');
		profile.params.path = 'v1/profiles/@did:web:example.test%3A8443:user:alice';
		expect((await call(proxyGet, profile)).status).toBe(200);
		expect(String(fetcher.mock.calls[0][0])).toBe(
			'https://appview.example.test/v1/profiles/%40did%3Aweb%3Aexample.test%253A8443%3Auser%3Aalice'
		);
		expect(safeProxyPath('v1/posts/did:web:example.test%3a8443/root')).toBe(
			'/v1/posts/did%3Aweb%3Aexample.test%253a8443/root'
		);
		for (const id of [
			'@did:web:example.test%253A8443',
			'did:web:example.test%3A8443%2fadmin',
			'did:web:example.test%3A8443%5cadmin',
			'did:web:example.test%2e%2e',
			'did:web:example.test%3A65536',
			'did:web:example..test%3A8443',
			'other%3A8443'
		])
			expect(safeProxyPath('v1/profiles/' + id)).toBeUndefined();
	});
	it('requires same-origin mutations and never follows an upstream redirect', async () => {
		const jar = new CookieJar().add(A);
		expect(
			(
				await call(
					proxyPost,
					event(
						jar,
						'/api/appview/v1/chat/conversations',
						'POST',
						{},
						{ origin: 'https://evil.test' }
					)
				)
			).status
		).toBe(403);
		expect(fetcher).not.toHaveBeenCalled();
		fetcher.mockResolvedValue(
			response({}, 302, { location: 'https://evil.test' })
		);
		expect(
			(await call(proxyGet, event(jar, '/api/appview/v1/profiles/me'))).status
		).toBe(502);
	});
	it('keeps cookies when logout cannot reach AppView, then removes only the named account after success', async () => {
		const jar = new CookieJar().add(A).add(B).active(B.id);
		fetcher
			.mockResolvedValueOnce(response({ error: 'unavailable' }, 503))
			.mockResolvedValueOnce(response(undefined, 204));
		expect(
			(
				await call(
					logout,
					event(jar, '/api/auth/logout', 'POST', { accountId: A.id })
				)
			).status
		).toBe(503);
		expect(credentials(jar.cookies, new URL(origin))).toHaveLength(2);
		expect(
			(
				await call(
					logout,
					event(jar, '/api/auth/logout', 'POST', { accountId: A.id })
				)
			).status
		).toBe(204);
		expect(credentials(jar.cookies, new URL(origin))).toEqual([B]);
		expect(jar.values.get('__Host-craftsky_active')).toBe(B.id);
	});
	it('validates switching and rejects cross-site session inspection', async () => {
		const jar = new CookieJar().add(A).add(B).active(A.id);
		expect(
			(
				await call(
					switchAccount,
					event(jar, '/api/auth/switch', 'POST', { accountId: 'unknown' })
				)
			).status
		).toBe(400);
		expect(
			(
				await call(
					switchAccount,
					event(jar, '/api/auth/switch', 'POST', { accountId: B.id })
				)
			).status
		).toBe(204);
		expect(jar.values.get('__Host-craftsky_active')).toBe(B.id);
		expect(
			(
				await call(
					session,
					event(jar, '/api/session', 'GET', undefined, {
						'sec-fetch-site': 'cross-site'
					})
				)
			).status
		).toBe(403);
	});
});

describe('browser deletion reauthentication', () => {
	it('binds start, callback and proof to the captured account and never serializes proof', async () => {
		const jar = new CookieJar().add(A).add(B).active(B.id);
		fetcher
			.mockResolvedValueOnce(
				response({
					jobId: jobID,
					confirmationDid: A.did,
					authUrl: 'https://provider.example.test/delete',
					warning: {
						code: 'provider_billing_not_canceled',
						message: 'PRIVATE_WARNING_CANARY'
					},
					expiresAt: new Date(Date.now() + 300000).toISOString()
				})
			)
			.mockResolvedValueOnce(new Response(null, { status: 202 }));
		const headers = { 'X-Craftsky-Account-Id': A.id };
		const started = await call(
			deletionStart,
			event(jar, '/api/auth/deletion/start', 'POST', {}, headers)
		);
		expect(started.status).toBe(200);
		const intentBody = await started.clone().text();
		expect(intentBody).toContain('provider_billing_not_canceled');
		expect(intentBody).not.toContain('PRIVATE_WARNING_CANARY');
		expect(
			new Headers(fetcher.mock.calls[0][1]?.headers).get('authorization')
		).toBe('Bearer ' + A.token);
		const saved = decode<Intent>(
			jar.values.get('__Host-craftsky_deletion_intent')
		)!;
		expect(
			await location(
				deletionComplete,
				event(
					jar,
					`/account-deletion/browser/reauth-complete?state=${saved.state}&job-id=${jobID}&proof=${capabilities}`
				)
			)
		).toContain('deletion=ready');
		const status = await (
			await call(
				deletionStatus,
				event(jar, '/api/auth/deletion/status', 'GET', undefined, headers)
			)
		).text();
		expect(status).toContain(A.did);
		expect(status).not.toContain(capabilities);
		expect(status).not.toContain('proof');
		expect(
			(
				await call(
					deletionConfirm,
					event(
						jar,
						'/api/auth/deletion/confirm',
						'POST',
						{ jobId: jobID, confirmationDid: A.did },
						headers
					)
				)
			).status
		).toBe(204);
		expect(credentials(jar.cookies, new URL(origin))).toEqual([B]);
		expect(jar.values.get('__Host-craftsky_active')).toBe(B.id);
	});
	it('rejects confirmation from a different account without touching AppView', async () => {
		const jar = new CookieJar().add(A).add(B).active(B.id);
		jar.values.set(
			'__Host-craftsky_deletion_pending',
			encode({
				state: capabilities,
				jobId: jobID,
				accountId: A.id,
				did: A.did,
				expiresAt: Date.now() + 300000,
				proof: token(8)
			})
		);
		expect(
			(
				await call(
					deletionConfirm,
					event(
						jar,
						'/api/auth/deletion/confirm',
						'POST',
						{ jobId: jobID, confirmationDid: A.did },
						{ 'X-Craftsky-Account-Id': B.id }
					)
				)
			).status
		).toBe(400);
		expect(fetcher).not.toHaveBeenCalled();
		expect(jar.values.has('__Host-craftsky_deletion_pending')).toBe(true);
	});
});

describe('bounded trusted upstream transport', () => {
	it('bounds handoff JSON responses and protects underlying error prose', async () => {
		fetcher.mockResolvedValueOnce(
			response({ payload: 'x'.repeat(1024 * 1024 + 1) })
		);
		await expect(
			upstream('/v1/auth/handoffs/exchange', accountA, { code: capabilities })
		).rejects.toMatchObject({ status: 502, code: 'invalid_upstream_response' });
		const cause = new Error('PRIVATE_TOKEN_CANARY');
		fetcher.mockRejectedValueOnce(cause);
		try {
			await upstream('/v1/auth/login', accountA, {});
		} catch (error) {
			expect(error).toBeInstanceOf(UpstreamError);
			expect(String(error)).not.toContain('PRIVATE_TOKEN_CANARY');
			expect((error as Error).cause).toBe(cause);
			expect(await safeFailure(error).text()).not.toContain(
				'PRIVATE_TOKEN_CANARY'
			);
		}
	});
});

it.each(['login', 'chat'] as const)(
	'keeps refreshed %s credentials and newer selection when an old request returns 401',
	async (kind) => {
		const old = new CookieJar().add(A).add(B).active(A.id);
		let finish!: (response: Response) => void;
		fetcher.mockImplementationOnce(
			() =>
				new Promise((resolve) => {
					finish = resolve;
				})
		);
		const waiting = call(proxyGet, event(old, '/api/appview/v1/profiles/me'));
		await vi.waitFor(() => expect(fetcher).toHaveBeenCalled());
		const browser = new CookieJar().add(A).add(B).active(A.id);
		const fresh = store(browser.cookies, new URL(origin), {
			...pending(kind),
			...(kind === 'chat'
				? { source: A.id, purpose: 'chat' as const, sourceTokenHash: token(5) }
				: {})
		});
		expect(fresh.id).toBe(A.id);
		expect(fresh.generation).toBeDefined();
		browser.active(B.id);
		browser.values.set('__Host-craftsky_intent', encode(intent()));
		old.changes = [];
		finish(response({ error: 'unauthorized' }, 401));
		await waiting;
		for (const change of old.changes) {
			if (change.value === undefined) browser.values.delete(change.name);
			else browser.values.set(change.name, change.value);
		}
		expect(
			credentials(browser.cookies, new URL(origin)).find(
				(a) => a.id === fresh.id
			)?.token
		).toBe(fresh.token);
		expect(selected(browser.cookies, new URL(origin))?.id).toBe(B.id);
		expect(browser.values.has('__Host-craftsky_intent')).toBe(true);
	}
);
it('rejects declared oversized JSON before fetching upstream', async () => {
	const jar = new CookieJar().add(A).active(A.id);
	const result = await call(
		proxyPost,
		event(jar, '/api/appview/v1/posts', 'POST', '{}', {
			'content-length': '1048577'
		})
	);
	expect(result.status).toBe(413);
	expect(fetcher).not.toHaveBeenCalled();
});
