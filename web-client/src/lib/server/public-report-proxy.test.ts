import { afterEach, expect, it, vi } from 'vitest';
import { publicReportResponse } from './public-report-proxy';

afterEach(() => {
	vi.unstubAllGlobals();
	vi.unstubAllEnvs();
});
const url = new URL(
	'https://app.craftsky.social/api/public/posts/did:plc:alice/first/reports'
);
function request(body: unknown, origin = url.origin) {
	return new Request(url, {
		method: 'POST',
		headers: {
			origin,
			'content-type': 'application/json',
			authorization: 'Bearer PRIVATE',
			cookie: 'PRIVATE'
		},
		body: JSON.stringify(body)
	});
}

it('proxies bounded guest reports with no account credentials and a minimal acknowledgement', async () => {
	vi.stubEnv('APPVIEW_ORIGIN', 'https://appview.example');
	const fetcher = vi.fn<typeof fetch>(async () =>
		Response.json(
			{
				reportId: 'report-1',
				status: 'accepted',
				details: 'PRIVATE_CANARY',
				contactEmail: 'PRIVATE_CANARY'
			},
			{ status: 201 }
		)
	);
	vi.stubGlobal('fetch', fetcher);
	const body = {
		reasonType: 'spam',
		details: 'private report',
		contactEmail: 'visitor@example.test'
	};
	const response = await publicReportResponse(
		request(body),
		url,
		'posts/did:plc:alice/first/reports'
	);
	expect(response.status).toBe(201);
	expect(response.headers.get('cache-control')).toBe('no-store');
	expect(await response.json()).toEqual({
		reportId: 'report-1',
		status: 'accepted'
	});
	const [path, options] = fetcher.mock.calls[0];
	expect(path).toBe(
		'https://appview.example/v1/public/posts/did%3Aplc%3Aalice/first/reports'
	);
	expect(options?.method).toBe('POST');
	const headers = new Headers(options?.headers);
	expect(headers.has('authorization')).toBe(false);
	expect(headers.has('cookie')).toBe(false);
	expect(JSON.parse(String(options?.body))).toEqual(body);
});

it('rejects cross-site, unsupported, spoofed and oversized submissions before upstream work', async () => {
	const fetcher = vi.fn<typeof fetch>();
	vi.stubGlobal('fetch', fetcher);
	for (const [req, target, path, status] of [
		[
			request({ reasonType: 'spam' }, 'https://evil.test'),
			url,
			'posts/did:plc:alice/first/reports',
			403
		],
		[request({ reasonType: 'spam' }), url, 'profiles/me/reports', 400],
		[
			request({ reasonType: 'spam' }),
			url,
			'posts/did:plc:alice/first/likes',
			400
		],
		[
			request({ reasonType: 'spam', reporterDid: 'did:plc:fake' }),
			url,
			'posts/did:plc:alice/first/reports',
			400
		],
		[
			request({ reasonType: 'spam', details: 'x'.repeat(8193) }),
			url,
			'posts/did:plc:alice/first/reports',
			413
		],
		[
			request({ reasonType: 'spam' }),
			new URL(url + '?viewerDid=fake'),
			'posts/did:plc:alice/first/reports',
			400
		]
	] as const) {
		expect((await publicReportResponse(req, target, path)).status).toBe(status);
	}
	expect(fetcher).not.toHaveBeenCalled();
});

it('preserves rate-limit recovery without exposing upstream private details', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () =>
			Response.json(
				{ error: 'rate_limited', details: 'PRIVATE_CANARY' },
				{ status: 429, headers: { 'retry-after': '6' } }
			)
		)
	);
	const response = await publicReportResponse(
		request({ reasonType: 'spam' }),
		url,
		'posts/did:plc:alice/first/reports'
	);
	expect(response.status).toBe(429);
	expect(response.headers.get('retry-after')).toBe('6');
	expect(await response.text()).not.toContain('PRIVATE_CANARY');
});
