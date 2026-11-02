// @ts-nocheck -- deliberately partial/error responses belong only to this isolated integration double.
// Isolated local AppView double. Test tokens never authenticate against real infrastructure.
import { createServer } from 'node:http';
import { randomBytes, randomUUID, createHash } from 'node:crypto';
let control = { owners: {}, confirmFailures: 0 },
	authEvents = [],
	authFlows = new Map(),
	authSessions = new Map(),
	deletions = new Map();
async function fixtureJSON(req) {
	const chunks = [];
	for await (const chunk of req) {
		chunks.push(chunk);
		if (Buffer.concat(chunks).length > 32768)
			throw new Error('Oversized fixture input');
	}
	return JSON.parse(Buffer.concat(chunks).toString() || '{}');
}
function oauthFlow(name, callback, device, kind, sourceToken) {
	const target = new URL(callback);
	if (
		target.origin !== 'http://127.0.0.1:19846' ||
		![
			'/auth/browser/complete',
			'/account-deletion/browser/reauth-complete'
		].includes(target.pathname)
	)
		throw new Error('Invalid fixture callback');
	const id = randomUUID(),
		flow = {
			id,
			name,
			callback,
			device,
			kind,
			code: randomBytes(32).toString('base64url'),
			token: randomBytes(32).toString('base64url'),
			receiptId: randomUUID(),
			confirmBy: new Date(Date.now() + 300000).toISOString(),
			sourceToken,
			state: 'ready'
		};
	authFlows.set(id, flow);
	return flow;
}
const owners = { alice: 'did:plc:alice', bob: 'did:plc:bob' };
let settings = new Map();
let calls = [],
	posts = new Map(),
	mutations = new Map(),
	schedules = new Map(),
	media = new Map();
const profile = (owner) => ({
	did: owner,
	handle: owner.endsWith('bob') ? 'bob.test' : 'alice.test',
	displayName: owner.endsWith('bob') ? 'Bob Maker' : 'Alice Maker',
	description: 'Making a little every day.',
	crafts: [
		'social.craftsky.feed.defs#knitting',
		'social.craftsky.feed.defs#sewing'
	],
	links: [],
	viewerIsFollowing: false,
	isCraftskyProfile: true,
	followerCount: 2,
	followingCount: 3,
	projectCount: 0,
	customisation: settings.get(owner)?.customisation || {
		colour: 'cobalt',
		profileBackground: 'none'
	},
	accountType: 'business',
	business: settings.get(owner)?.business
});
const post = (owner, rkey = 'first', payload = {}) => ({
	uri: `at://${owner}/social.craftsky.feed.post/${rkey}`,
	cid: `cid-${rkey}`,
	rkey,
	text: 'A little knitting for the weekend.',
	author: profile(owner),
	createdAt: '2026-11-01T12:00:00Z',
	indexedAt: '2026-11-01T12:00:01Z',
	tags: [],
	langs: ['en'],
	sponsored: false,
	likeCount: 0,
	repostCount: 0,
	quoteCount: 0,
	replyCount: 0,
	viewerHasLiked: false,
	viewerHasReposted: false,
	viewerHasReplied: false,
	viewerHasSaved: false,
	...payload
});
function reset() {
	control = { owners: {}, confirmFailures: 0 };
	authEvents = [];
	authFlows = new Map();
	deletions = new Map();
	authSessions = new Map([
		[Buffer.alloc(32, 1).toString('base64url'), 'alice'],
		[Buffer.alloc(32, 2).toString('base64url'), 'bob']
	]);
	calls = [];
	settings = new Map(
		Object.values(owners).map((owner) => [
			owner,
			{
				languages: { primaryLanguage: 'en', contentLanguages: ['en'] },
				customisation: { colour: 'cobalt', profileBackground: 'none' },
				notifications: {
					like: { scope: 'everyone', pushEnabled: true },
					follow: { scope: 'everyone', pushEnabled: true },
					moderation: { scope: 'everyone', pushEnabled: true },
					futureCategory: { scope: 'everyone', pushEnabled: false }
				},
				business: {
					cid: 'business-cid-0',
					businessTypes: [
						{ value: 'dyer', known: true },
						{ value: 'future-type', known: false }
					],
					offerings: [{ value: 'yarn', known: true }],
					tagline: 'Original tagline',
					products: [
						{
							title: 'Blue wool',
							uri: 'https://shop.example/wool',
							image: {
								cid: 'bafyproduct',
								mime: 'image/png',
								size: 200,
								alt: 'Blue wool'
							},
							price: { amount: '12.5', currency: 'GBP' }
						}
					]
				},
				events: [],
				verification: null,
				instagram: null,
				imports: [],
				suggestions: [],
				license: {
					id: 'license-1',
					subscriptionId: 'subscription-1',
					tier: 'business',
					assignable: true,
					anomaly: 'none'
				}
			}
		])
	);
	posts = new Map();
	mutations = new Map();
	schedules = new Map();
	media = new Map();
	for (const owner of Object.values(owners))
		posts.set(`${owner}/first`, post(owner));
}
reset();
createServer(async (req, res) => {
	const url = new URL(req.url || '/', 'http://localhost');
	const pathname = decodeURIComponent(url.pathname),
		method = req.method;
	const send = (status, value, headers = {}) => {
		res.writeHead(status, { 'content-type': 'application/json', ...headers });
		res.end(status === 204 ? undefined : JSON.stringify(value));
	};
	if (pathname === '/__test/reset') {
		reset();
		return send(200, { ok: true });
	}
	if (pathname === '/__test/requests') return send(200, calls);
	if (pathname === '/__test/control' && method === 'POST') {
		const value = await fixtureJSON(req);
		control = {
			...control,
			...value,
			owners: { ...control.owners, ...value.owners }
		};
		return send(200, { ok: true });
	}
	if (pathname === '/__test/auth-state')
		return send(200, {
			events: authEvents,
			deletions: [...deletions.values()].map((value) => ({
				jobId: value.jobId,
				name: value.name,
				accepted: !!value.accepted
			}))
		});
	const provider = pathname.match(/^\/__test\/oauth-provider\/([0-9a-f-]+)$/);
	if (provider) {
		const flow = authFlows.get(provider[1]);
		if (!flow) return send(400, { error: 'invalid_fixture_flow' });
		const callback = new URL(flow.callback);
		if (flow.kind === 'delete') {
			const deletion = deletions.get(flow.jobId);
			callback.searchParams.set('job-id', flow.jobId);
			callback.searchParams.set('proof', deletion.proof);
		} else callback.searchParams.set('code', flow.code);
		authEvents.push({ stage: 'provider', name: flow.name });
		res.writeHead(303, {
			location: callback.href,
			'cache-control': 'no-store'
		});
		return res.end();
	}
	if (
		[
			'/v1/auth/login',
			'/v1/auth/registrations',
			'/v1/chat/authorization'
		].includes(pathname) &&
		method === 'POST'
	) {
		const body = await fixtureJSON(req),
			source = req.headers.authorization?.replace('Bearer ', ''),
			name = pathname.includes('/chat/')
				? authSessions.get(source)
				: body.handle === 'bob.test'
					? 'bob'
					: 'alice';
		if (!name) return send(401, { error: 'unauthorized' });
		const flow = oauthFlow(
			name,
			body.browserRedirectUri,
			req.headers['x-craftsky-device-id'],
			pathname.includes('/chat/') ? 'chat' : 'login',
			source
		);
		authEvents.push({ stage: 'start', name });
		return send(200, {
			authUrl: `http://localhost:19847/__test/oauth-provider/${flow.id}`
		});
	}
	if (pathname === '/v1/auth/handoffs/exchange' && method === 'POST') {
		const body = await fixtureJSON(req),
			flow = [...authFlows.values()].find((value) => value.code === body.code);
		if (
			!flow ||
			flow.device !== req.headers['x-craftsky-device-id'] ||
			flow.state === 'confirmed'
		)
			return send(400, { error: 'invalid_handoff' });
		flow.state = 'pending';
		authEvents.push({ stage: 'exchange', name: flow.name });
		return send(200, {
			token: flow.token,
			did: owners[flow.name],
			handle: flow.name + '.test',
			receiptId: flow.receiptId,
			confirmBy: flow.confirmBy,
			// Production response decorators enrich every did+handle identity,
			// including handoff responses from both ordinary and chat OAuth.
			accountType: 'personal',
			crafts: ['social.craftsky.feed.defs#sewing'],
			customisation: { colour: 'cobalt', profileBackground: 'none' },
			...(flow.kind === 'chat'
				? {
						purpose: 'chat',
						sourceTokenHash: createHash('sha256')
							.update(flow.sourceToken)
							.digest('base64url')
					}
				: {})
		});
	}
	if (pathname === '/v1/auth/handoffs/confirm' && method === 'POST') {
		const body = await fixtureJSON(req),
			bearer = req.headers.authorization?.replace('Bearer ', ''),
			flow = [...authFlows.values()].find(
				(value) => value.token === bearer && value.receiptId === body.receiptId
			);
		if (!flow || flow.device !== req.headers['x-craftsky-device-id'])
			return send(400, { error: 'invalid_handoff' });
		authEvents.push({ stage: 'confirm', name: flow.name });
		if (control.confirmFailures > 0) {
			control.confirmFailures--;
			return send(503, { error: 'handoff_unavailable' });
		}
		flow.state = 'confirmed';
		authSessions.set(flow.token, flow.name);
		return send(204);
	}

	const token = req.headers.authorization?.replace('Bearer ', '');
	const name = authSessions.get(token);
	const owner = name && owners[name];
	if (!owner)
		return send(401, {
			error: 'session_expired',
			message: 'Fixture session required'
		});
	if (req.headers.cookie)
		return send(500, {
			error: 'cookie_forwarded',
			message: 'Browser cookies must not reach AppView'
		});
	const chunks = [];
	for await (const chunk of req) chunks.push(chunk);
	const raw = Buffer.concat(chunks);
	const type = req.headers['content-type'] || '';
	let body;
	try {
		body =
			raw.length && type.includes('application/json')
				? JSON.parse(raw.toString())
				: undefined;
	} catch {
		return send(400, {
			error: 'malformed_body',
			message: 'Malformed fixture request'
		});
	}
	calls.push({
		method,
		path: pathname,
		query: Object.fromEntries(url.searchParams),
		body,
		bytes: body ? undefined : raw.length,
		contentType: type,
		owner,
		key: req.headers['idempotency-key'],
		ifMatch: req.headers['if-match']
	});

	const flags = control.owners[name] || {};
	if (flags.eligibility === 'restricted' && pathname === '/v1/profiles/me')
		return send(403, {
			error: 'account_age_restricted',
			message: 'Retained account controls remain available'
		});
	if (flags.apiStatus && (!flags.apiPath || flags.apiPath === pathname))
		return send(flags.apiStatus, {
			error:
				flags.apiStatus === 401 ? 'session_expired' : 'temporary_unavailable',
			message: 'Controlled fixture response'
		});
	if (pathname === '/v1/auth/logout' && method === 'POST') {
		authSessions.delete(token);
		return send(204);
	}
	if (pathname === '/v1/onboarding/completion' && method === 'POST') {
		if (body.meetsMinimumAge !== true || body.policyVersion !== 'v1')
			return send(400, { error: 'invalid_declaration' });
		control.owners[name] = { ...flags, completed: true };
		authEvents.push({
			stage: 'onboarding',
			name,
			meetsMinimumAge: body.meetsMinimumAge,
			policyVersion: body.policyVersion
		});
		return send(200, { completed: true, requiredPolicyVersion: 'v1' });
	}
	if (pathname === '/v1/account-deletion/intents' && method === 'POST') {
		const jobId = randomUUID(),
			proof = randomBytes(32).toString('base64url'),
			flow = oauthFlow(
				name,
				body.browserRedirectUri,
				req.headers['x-craftsky-device-id'],
				'delete'
			);
		flow.jobId = jobId;
		deletions.set(jobId, { jobId, proof, name, device: flow.device });
		authEvents.push({ stage: 'deletion-start', name });
		return send(201, {
			jobId,
			confirmationDid: owner,
			authUrl: `http://localhost:19847/__test/oauth-provider/${flow.id}`,
			expiresAt: new Date(Date.now() + 300000).toISOString()
		});
	}
	const deletion = pathname.match(/^\/v1\/account-deletions\/([0-9a-f-]+)$/);
	if (deletion && method === 'POST') {
		const value = deletions.get(deletion[1]);
		if (
			!value ||
			value.name !== name ||
			value.device !== req.headers['x-craftsky-device-id'] ||
			value.proof !== body.reauthProof ||
			body.confirmationDid !== owner
		)
			return send(400, { error: 'invalid_deletion_proof' });
		value.accepted = true;
		authEvents.push({ stage: 'deletion-accepted', name });
		res.writeHead(202);
		return res.end();
	}
	const cancelDeletion = pathname.match(
		/^\/v1\/account-deletion\/intents\/([0-9a-f-]+)$/
	);
	if (cancelDeletion && method === 'DELETE') {
		const value = deletions.get(cancelDeletion[1]);
		if (!value || value.name !== name)
			return send(400, { error: 'invalid_deletion_proof' });
		deletions.delete(cancelDeletion[1]);
		return send(204);
	}
	if (pathname === '/v1/whoami')
		return send(200, { did: owner, handle: `${name}.test` });
	if (pathname === '/v1/onboarding/status')
		return send(200, {
			completed: flags.completed !== false,
			requiredPolicyVersion: 'v1'
		});
	if (pathname === '/v1/account/eligibility')
		return send(200, {
			state: flags.eligibility || 'eligible',
			appealable: flags.eligibility === 'restricted',
			appealGuidance:
				flags.eligibility === 'restricted'
					? 'Review your standing or manage your account.'
					: undefined
		});
	if (pathname === '/v1/moderation/standing')
		return send(200, {
			suspended: false,
			severeSuspended: false,
			thresholdSuspended: false,
			activeStrikeCount: 0,
			strikeThreshold: 3
		});
	if (
		pathname === '/v1/moderation/history' ||
		pathname.startsWith('/v1/moderation/history/')
	)
		return send(200, {
			items: [
				{
					caseReference: 'fixture-case-reference',
					eventType: 'decision',
					reason: 'spam',
					userSafeDetail:
						'Review this content against the community guidelines.',
					effects: [{ type: 'visibilityWarn', action: 'apply' }],
					appealStatus: 'none',
					occurredAt: '2026-11-02T12:00:00Z',
					safeSnapshot: {
						type: 'post',
						did: owner,
						uri: `at://${owner}/social.craftsky.feed.post/moderated-post`
					}
				}
			]
		});
	if (pathname === '/v1/subscriptions/access')
		return send(200, {
			did: owner,
			effectiveTier: 'business',
			givesAccess: true
		});
	if (pathname === '/v1/notifications/new-count')
		return send(200, { newCount: 0 });
	if (pathname === '/v1/chat/unread-counts')
		return send(200, { accepted: 0, requests: 0 });
	if (pathname === '/v1/languages/preferences') {
		const state = settings.get(owner);
		if (method === 'PUT') state.languages = body;
		return send(200, state.languages);
	}
	const state = settings.get(owner);
	if (pathname === '/v1/notifications/preferences') {
		if (method === 'PATCH')
			for (const [category, patch] of Object.entries(body.preferences))
				state.notifications[category] = {
					...state.notifications[category],
					...patch
				};
		return send(200, { preferences: state.notifications });
	}
	if (pathname === '/v1/profiles/me/customisation') {
		state.customisation = body;
		return send(200, body);
	}
	if (pathname === '/v1/profiles/me/business') {
		if (
			!req.headers['idempotency-key'] ||
			req.headers['if-match'] !== state.business?.cid
		)
			return send(409, {
				error: 'business_conflict',
				message: 'Business profile changed'
			});
		if (method === 'DELETE') {
			state.business = null;
			return send(204);
		}
		const previous = state.business;
		state.business = {
			...body,
			cid: `${previous.cid}-next`,
			businessTypes: [
				...body.businessTypes.map((value) => ({ value, known: true })),
				...previous.businessTypes.filter((value) => !value.known)
			],
			offerings: body.offerings.map((value) => ({ value, known: true })),
			products: body.preserveProducts ? previous.products : body.products
		};
		return send(200, { cid: state.business.cid });
	}
	if (pathname === '/v1/events') {
		if (method === 'POST') {
			if (!req.headers['idempotency-key'])
				return send(400, { error: 'missing_key', message: 'Key required' });
			const item = {
				...body,
				did: owner,
				rkey: 'event-1',
				uri: `at://${owner}/social.craftsky.business.event/event-1`,
				cid: 'event-cid-1',
				roles: body.roles.map((value) => ({ value, known: true })),
				mode: { value: body.mode, known: true },
				status: { value: body.status, known: true }
			};
			state.events.push(item);
			return send(201, {
				did: owner,
				rkey: item.rkey,
				uri: item.uri,
				cid: item.cid
			});
		}
		return send(200, { items: state.events });
	}
	if (pathname.startsWith('/v1/events/')) {
		const item = state.events.find((event) =>
			pathname.endsWith('/' + event.rkey)
		);
		if (!item)
			return send(404, { error: 'event_missing', message: 'Not found' });
		if (req.headers['if-match'] !== item.cid)
			return send(409, { error: 'event_conflict', message: 'Event changed' });
		if (method === 'DELETE') {
			state.events = [];
			return send(204);
		}
		Object.assign(item, body, {
			cid: item.cid + '-next',
			roles: body.roles.map((value) => ({ value, known: true })),
			mode: { value: body.mode, known: true },
			status: { value: body.status, known: true }
		});
		return send(200, {
			cid: item.cid,
			did: owner,
			rkey: item.rkey,
			uri: item.uri
		});
	}
	if (pathname === '/v1/billing/account')
		return send(200, {
			billingAccountId: 'billing-1',
			revenueCatAppUserId: 'private-rc-id',
			requestedGeneration: 1,
			reconciledGeneration: 1,
			reconciliationStale: false,
			subscriptions: [
				{
					id: 'subscription-1',
					productId: 'business-monthly',
					store: 'app_store',
					status: 'active',
					givesAccess: true,
					pendingPayment: false,
					anomaly: 'none'
				}
			],
			licenses: [state.license]
		});
	if (pathname === '/v1/billing/reconciliation')
		return send(200, { generation: 1 });
	if (pathname === '/v1/billing/licenses/license-1/assignment') {
		if (method === 'DELETE') {
			delete state.license.assignedDid;
			return send(204);
		}
		state.license.assignedDid = body.targetDid;
		return send(200, {
			licenseId: 'license-1',
			targetDid: body.targetDid,
			assignedAt: '2026-11-02T12:00:00Z'
		});
	}
	if (pathname === '/v1/migrations/instagram/account') {
		if (method === 'DELETE') {
			state.instagram = null;
			return send(204);
		}
		return send(200, { integrationAvailable: true, account: state.instagram });
	}
	if (
		pathname === '/v1/migrations/instagram/verifications' &&
		method === 'POST'
	) {
		state.verification = {
			verificationId: 'verify-1',
			state: 'pendingDm',
			expiresAt: '2026-12-01T12:00:00Z'
		};
		return send(201, {
			...state.verification,
			challenge: 'craftsky-test-challenge',
			dmUrl: 'https://www.instagram.com/direct/t/fixture'
		});
	}
	if (pathname === '/v1/migrations/instagram/verifications/current') {
		if (state.verification?.state === 'pendingDm')
			state.verification = {
				...state.verification,
				state: 'pendingConfirmation',
				candidateUsername: 'alice_makes'
			};
		return send(200, { verification: state.verification });
	}
	if (pathname.endsWith('/verifications/verify-1/confirm')) {
		state.instagram = {
			state: 'active',
			username: 'alice_makes',
			discoverable: body.discoverable,
			verifiedAt: '2026-11-02T12:00:00Z'
		};
		state.verification = { ...state.verification, state: 'confirmed' };
		return send(200, { state: 'confirmed', account: state.instagram });
	}
	if (pathname === '/v1/migrations/instagram/settings') {
		Object.assign(state.instagram, body);
		return send(200, { integrationAvailable: true, account: state.instagram });
	}
	if (pathname === '/v1/migrations/instagram/imports') {
		if (method === 'POST') {
			const item = {
				importId: 'import-1',
				state: 'active',
				sourceType: body.sourceType,
				followingCount: body.entries.length,
				createdAt: '2026-11-02T12:00:00Z'
			};
			state.imports.push(item);
			state.suggestions = [
				{
					suggestionId: 'suggestion-1',
					target: {
						did: owners.bob,
						handle: 'bob.test',
						displayName: 'Bob Maker'
					}
				}
			];
			return send(201, {
				import: item,
				counts: { followingCount: item.followingCount }
			});
		}
		return send(200, { items: state.imports });
	}
	if (pathname === '/v1/migrations/instagram/suggestions')
		return send(200, { items: state.suggestions });
	if (pathname === '/v1/migrations/instagram/suggestions/suggestion-1/accept') {
		if (!req.headers['idempotency-key'])
			return send(400, { error: 'missing_key', message: 'Key required' });
		state.suggestions = [];
		return send(200, { suggestionId: 'suggestion-1', state: 'followed' });
	}
	if (pathname === '/v1/feed/timeline')
		return send(200, {
			items: [...posts.values()]
				.filter((value) => value.author.did === owner)
				.map((value) => ({ itemKey: value.uri, post: value }))
		});
	if (pathname === '/v1/posts' && method === 'POST') {
		const key = req.headers['idempotency-key'];
		if (!key)
			return send(400, { error: 'missing_key', message: 'Key required' });
		const cached = mutations.get(`${owner}/${key}`);
		if (cached) {
			if (cached.body !== JSON.stringify(body))
				return send(409, { error: 'key_reused', message: 'Payload changed' });
			return send(201, cached.value);
		}
		const rkey = `created-${[...posts.values()].filter((value) => value.author.did === owner).length}`;
		const value = post(owner, rkey, body);
		posts.set(`${owner}/${rkey}`, value);
		mutations.set(`${owner}/${key}`, { body: JSON.stringify(body), value });
		if (body.text.startsWith('Ambiguous'))
			return send(202, { status: 'ambiguous' }, { 'retry-after': '1' });
		return send(201, value);
	}
	const target = pathname.match(
		/^\/v1\/posts\/(did:[^/]+)\/([^/]+)(?:\/(.*))?$/
	);
	if (target) {
		const [, _did, rkey, action] = target;
		const value = posts.get(`${_did}/${rkey}`);
		if (!value)
			return send(404, { error: 'post_not_found', message: 'Not found' });
		if (action === 'edit')
			return send(200, {
				post: value,
				baseCid: value.cid,
				recordRevision: null,
				editable: {
					text: value.text,
					sponsored: value.sponsored,
					langs: value.langs,
					...(value.project ? { project: value.project } : {}),
					images: (value.images || []).map((image, index) => ({
						...image,
						existingIndex: index
					}))
				}
			});
		if (action === 'reports' && method === 'POST') {
			if (!['spam', 'other'].includes(body.reasonType))
				return send(400, { error: 'invalid_reason' });
			return send(201, { reportId: 'fixture-report', status: 'accepted' });
		}
		if (action === 'comments')
			return send(200, {
				post: value,
				comments: { items: [] },
				sort: 'oldest'
			});
		if (action === 'likes' || action === 'reposts') {
			if (method === 'GET') return send(200, { items: [], totalCount: 0 });
			if (method === 'DELETE') return send(204);
			return send(201, {
				uri: `at://${owner}/social.craftsky.feed.like/liked`,
				cid: 'like-cid'
			});
		}
		if (method === 'PATCH') {
			if (
				body.changes.text === 'Conflict changes' ||
				req.headers['if-match'] !== value.cid
			)
				return send(409, {
					error: 'post_edit_conflict',
					message: 'The post changed since you opened it'
				});
			Object.assign(value, body.changes, {
				cid: `${value.cid}-edited`,
				editedAt: new Date().toISOString()
			});
			return send(200, {
				acceptedCid: value.cid,
				changed: true,
				editedAt: value.editedAt
			});
		}
		if (method === 'DELETE') {
			posts.delete(`${_did}/${rkey}`);
			return send(204);
		}
		return send(200, value);
	}
	if (pathname === '/v1/blobs/images' && method === 'POST')
		return send(200, {
			blob: {
				$type: 'blob',
				ref: { $link: 'bafy-image' },
				mimeType: type,
				size: raw.length
			},
			cid: 'bafy-image',
			mime: type,
			size: raw.length
		});
	const mediaId = pathname.match(/^\/v1\/scheduled-post-media\/([^/]+)$/)?.[1];
	if (mediaId) {
		if (method === 'PUT') {
			media.set(`${owner}/${mediaId}`, { raw, type });
			return send(200, {
				id: mediaId,
				state: 'ready',
				mimeType: type,
				sizeBytes: raw.length
			});
		}
		if (method === 'DELETE') {
			media.delete(`${owner}/${mediaId}`);
			return send(204);
		}
		const value = media.get(`${owner}/${mediaId}`);
		if (!value)
			return send(404, { error: 'media_not_found', message: 'Not found' });
		res.writeHead(200, { 'content-type': value.type });
		return res.end(value.raw);
	}
	if (pathname === '/v1/scheduled-posts') {
		if (method === 'POST') {
			const item = {
				id: body.operationId,
				operationId: body.operationId,
				owner,
				status: 'scheduled',
				scheduledAt: body.scheduledAt,
				payload: body.payload
			};
			schedules.set(`${owner}/${item.id}`, item);
			return send(201, item);
		}
		return send(200, {
			items: [...schedules.values()]
				.filter((item) => item.owner === owner)
				.map((item) => ({
					id: item.id,
					kind: item.payload.kind,
					status: item.status,
					textPreview: item.payload.text,
					scheduledAt: item.scheduledAt,
					projectTitle: item.payload.project?.common?.title
				})),
			count: schedules.size,
			needsAttentionCount: 0
		});
	}
	const scheduled = pathname.match(
		/^\/v1\/scheduled-posts\/([^/]+)(\/publication)?$/
	);
	if (scheduled) {
		const item = schedules.get(`${owner}/${scheduled[1]}`);
		if (!item)
			return send(404, {
				error: 'scheduled_post_not_found',
				message: 'Not found'
			});
		if (scheduled[2]) {
			schedules.delete(`${owner}/${item.id}`);
			return send(202, { status: 'publishing' });
		}
		if (method === 'DELETE') {
			schedules.delete(`${owner}/${item.id}`);
			return send(204);
		}
		if (method === 'PUT') Object.assign(item, body);
		return send(200, item);
	}
	if (pathname.startsWith('/v1/profiles/')) {
		const match = pathname.match(/^\/v1\/profiles\/([^/]+)(?:\/(.*))?$/);
		const id = match?.[1].replace(/^@/, '');
		const did =
			id === 'me'
				? owner
				: id?.startsWith('did:')
					? id
					: id === 'bob.test'
						? owners.bob
						: owners.alice;
		const sub = match?.[2];
		if (sub === 'pins') return send(200, {});
		if (sub && ['posts', 'projects', 'comments', 'reposts'].includes(sub))
			return send(200, {
				items: [...posts.values()].filter(
					(item) =>
						item.author.did === did && (sub !== 'projects' || !!item.project)
				)
			});
		if (sub === 'follows')
			return send(200, {
				...profile(did),
				viewerIsFollowing: method === 'POST'
			});
		if (
			sub === 'mutual-followers' ||
			sub === 'followers' ||
			sub === 'following' ||
			sub === 'mutes' ||
			sub === 'blocks'
		)
			return send(200, { items: [], totalCount: 0 });
		return send(200, profile(did));
	}
	if (pathname === '/v1/search/profiles')
		return send(200, { items: Object.values(owners).map(profile) });
	if (pathname === '/v1/search/suggestions')
		return send(200, {
			profiles: { items: [], hasMore: false },
			hashtags: { items: [], hasMore: false }
		});
	if (
		pathname === '/v1/search/recent' ||
		pathname === '/v1/search/hashtags' ||
		pathname === '/v1/notifications' ||
		pathname === '/v1/saved-posts' ||
		pathname === '/v1/saved-post-folders'
	)
		return send(200, { items: [] });
	if (pathname === '/v1/search/hashtags/top') return send(200, { groups: [] });
	if (pathname === '/v1/notifications/seen') return send(204);
	if (pathname.startsWith('/v1/facets/'))
		return send(
			200,
			pathname.endsWith('/resolve') ? { did: owners.alice } : { items: [] }
		);
	return send(404, {
		error: 'fixture_route_missing',
		message: `No fixture for ${method} ${pathname}`
	});
}).listen(19847, '127.0.0.1');
