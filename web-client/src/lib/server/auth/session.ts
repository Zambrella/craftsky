import { authenticatedRoute } from '#lib/routes.ts';
import { Buffer } from 'node:buffer';
import type { Cookies } from '@sveltejs/kit';

export type BrowserAccount = { id: string; did: string; handle: string };
export type Credential = BrowserAccount & {
	token: string;
	generation?: string;
};
export type Intent = {
	state: string;
	active: string;
	kind: 'login' | 'register' | 'chat';
	returnTo: string;
	source?: string;
	expiresAt: number;
};
export type Pending = {
	token: string;
	did: string;
	handle: string;
	receiptId: string;
	confirmBy: string;
	purpose?: 'chat';
	sourceTokenHash?: string;
	active: string;
	returnTo: string;
	source?: string;
	kind: Intent['kind'];
};
export type DeletionIntent = {
	state: string;
	jobId: string;
	accountId: string;
	did: string;
	expiresAt: number;
};
export type DeletionPending = DeletionIntent & { proof: string };
const lifetime = 60 * 60 * 24 * 180;
export const accountID = (value: unknown): value is string =>
	typeof value === 'string' &&
	/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(
		value
	);
const did = (value: unknown): value is string =>
	typeof value === 'string' &&
	value.length <= 2048 &&
	/^did:[a-z]+:[a-zA-Z0-9._:%-]+$/.test(value);
const handle = (value: unknown): value is string =>
	typeof value === 'string' &&
	value.length <= 253 &&
	value.length > 0 &&
	/^[a-zA-Z0-9.-]+$/.test(value);
export function capability(value: unknown): value is string {
	return (
		typeof value === 'string' &&
		/^[A-Za-z0-9_-]{43}$/.test(value) &&
		Buffer.from(value, 'base64url').length === 32 &&
		Buffer.from(value, 'base64url').toString('base64url') === value
	);
}
const object = (value: unknown): value is Record<string, unknown> =>
	!!value && typeof value === 'object' && !Array.isArray(value);
const keys = (value: Record<string, unknown>, allowed: string[]) =>
	Object.keys(value).every((key) => allowed.includes(key));
const validChoice = (value: unknown) => value === '' || accountID(value);
const unexpired = (value: unknown) =>
	typeof value === 'number' &&
	Number.isSafeInteger(value) &&
	value > Date.now() &&
	value <= Date.now() + 15 * 60 * 1000;
export function prefix(url: URL) {
	return url.protocol === 'https:' ? '__Host-craftsky_' : 'craftsky_';
}
export function options(url: URL, maxAge = lifetime) {
	return {
		path: '/',
		httpOnly: true,
		secure: url.protocol === 'https:',
		sameSite: 'lax' as const,
		maxAge
	};
}
export function encode(value: unknown) {
	const encoded = Buffer.from(JSON.stringify(value), 'utf8').toString(
		'base64url'
	);
	if (encoded.length > 3800)
		throw new Error('Browser session exceeds cookie limit');
	return encoded;
}
/** Cookie payloads are untrusted metadata. AppView validates the opaque bearer. */
export function decode<T>(value: string | undefined): T | undefined {
	if (!value || value.length > 3800 || !/^[A-Za-z0-9_-]+$/.test(value)) return;
	try {
		const bytes = Buffer.from(value, 'base64url');
		if (bytes.toString('base64url') !== value) return;
		const result: unknown = JSON.parse(bytes.toString('utf8'));
		return object(result) ? (result as T) : undefined;
	} catch {
		return;
	}
}
export function returnPath(raw: unknown): string {
	if (
		typeof raw !== 'string' ||
		raw.length > 2048 ||
		!raw.startsWith('/') ||
		/[\\\u0000-\u0020\u007f]/.test(raw) ||
		/%(?:2f|5c|00|0a|0d)/i.test(raw)
	)
		return '/feed';
	try {
		const url = new URL(raw, 'https://app.craftsky.social');
		return url.origin === 'https://app.craftsky.social' &&
			authenticatedRoute(url.pathname)
			? url.pathname + url.search + url.hash
			: '/feed';
	} catch {
		return '/feed';
	}
}
export function readIntent(value: string | undefined): Intent | undefined {
	const item = decode<Record<string, unknown>>(value);
	if (
		!item ||
		!keys(item, [
			'state',
			'active',
			'kind',
			'returnTo',
			'source',
			'expiresAt'
		]) ||
		!capability(item.state) ||
		!validChoice(item.active) ||
		!['login', 'register', 'chat'].includes(String(item.kind)) ||
		typeof item.returnTo !== 'string' ||
		returnPath(item.returnTo) !== item.returnTo ||
		!unexpired(item.expiresAt)
	)
		return;
	if (
		item.kind === 'chat'
			? !accountID(item.source) || item.active !== item.source
			: item.source !== undefined
	)
		return;
	return item as Intent;
}
export function handoffResult(
	value: unknown
): Omit<Pending, 'active' | 'returnTo' | 'source' | 'kind'> | undefined {
	// AppView identity decorators add profile fields to did+handle responses.
	// Accept additive upstream fields, then project only validated auth data.
	// Browser cookie schemas remain exact in readPending/readIntent.
	if (
		!object(value) ||
		!capability(value.token) ||
		!did(value.did) ||
		!handle(value.handle) ||
		!accountID(value.receiptId) ||
		typeof value.confirmBy !== 'string'
	)
		return;
	const confirmBy = Date.parse(value.confirmBy);
	if (
		!Number.isFinite(confirmBy) ||
		confirmBy <= Date.now() ||
		confirmBy > Date.now() + 60 * 60 * 1000
	)
		return;
	if (
		value.purpose === 'chat'
			? !capability(value.sourceTokenHash)
			: (value.purpose !== undefined && value.purpose !== '') ||
				value.sourceTokenHash !== undefined
	)
		return;
	return {
		token: value.token,
		did: value.did,
		handle: value.handle,
		receiptId: value.receiptId,
		confirmBy: value.confirmBy,
		...(value.purpose === 'chat'
			? { purpose: 'chat', sourceTokenHash: value.sourceTokenHash as string }
			: {})
	};
}
export function readPending(value: string | undefined): Pending | undefined {
	const item = decode<Record<string, unknown>>(value);
	if (
		!item ||
		!keys(item, [
			'token',
			'did',
			'handle',
			'receiptId',
			'confirmBy',
			'purpose',
			'sourceTokenHash',
			'active',
			'returnTo',
			'source',
			'kind'
		])
	)
		return;
	const { active, returnTo, source, kind, ...handoff } = item;
	const result = handoffResult(handoff);
	if (
		!result ||
		!validChoice(active) ||
		typeof returnTo !== 'string' ||
		returnPath(returnTo) !== returnTo ||
		!['login', 'register', 'chat'].includes(String(kind))
	)
		return;
	if (
		kind === 'chat'
			? !accountID(source) || active !== source || result.purpose !== 'chat'
			: source !== undefined || result.purpose !== undefined
	)
		return;
	return {
		...result,
		active: active as string,
		returnTo,
		kind: kind as Intent['kind'],
		...(source ? { source: source as string } : {})
	};
}
export function readDeletionIntent(
	value: string | undefined,
	pending = false
): DeletionIntent | DeletionPending | undefined {
	const item = decode<Record<string, unknown>>(value);
	if (
		!item ||
		!keys(item, [
			'state',
			'jobId',
			'accountId',
			'did',
			'expiresAt',
			...(pending ? ['proof'] : [])
		]) ||
		!capability(item.state) ||
		!accountID(item.jobId) ||
		!accountID(item.accountId) ||
		!did(item.did) ||
		!unexpired(item.expiresAt) ||
		(pending && !capability(item.proof))
	)
		return;
	return item as DeletionIntent | DeletionPending;
}
export function device(cookies: Cookies, url: URL) {
	const name = prefix(url) + 'device';
	let id = cookies.get(name);
	if (!accountID(id)) {
		id = crypto.randomUUID();
		cookies.set(name, id, options(url));
	}
	return id;
}
export function credentials(cookies: Cookies, url: URL): Credential[] {
	const key = prefix(url) + 'account_';
	const out: Credential[] = [];
	for (const item of cookies.getAll()) {
		if (!item.name.startsWith(key)) continue;
		const value = decode<Record<string, unknown>>(item.value);
		if (
			value &&
			keys(value, ['id', 'token', 'did', 'handle', 'generation']) &&
			accountID(value.id) &&
			(value.generation === undefined || accountID(value.generation)) &&
			item.name ===
				key + value.id + (value.generation ? '_' + value.generation : '') &&
			capability(value.token) &&
			did(value.did) &&
			handle(value.handle) &&
			!out.some((account) => account.id === value.id)
		)
			out.push(value as Credential);
	}
	return out.slice(0, 8);
}
export function selected(cookies: Cookies, url: URL, explicit?: string) {
	const list = credentials(cookies, url);
	if (explicit !== undefined)
		return accountID(explicit)
			? list.find((item) => item.id === explicit)
			: undefined;
	const active = cookies.get(prefix(url) + 'active');
	return list.find((item) => item.id === active) || list[0];
}
export function publicAccount(value: Credential): BrowserAccount {
	return { id: value.id, did: value.did, handle: value.handle };
}
export function credentialCookie(url: URL, value: Credential) {
	return (
		prefix(url) +
		'account_' +
		value.id +
		(value.generation ? '_' + value.generation : '')
	);
}
export function remove(cookies: Cookies, url: URL, id: string) {
	// The response may arrive after another tab refreshed credentials or selected an account.
	// Only this activation's uniquely named credential may be invalidated. Selection
	// falls back when its referenced credential is absent; shared intent cookies expire.
	const credential = credentials(cookies, url).find((item) => item.id === id);
	if (credential)
		cookies.delete(credentialCookie(url, credential), { path: '/' });
}
export function store(cookies: Cookies, url: URL, pending: Pending) {
	if (!readPending(encode(pending)))
		throw new Error('Invalid pending browser session');
	const list = credentials(cookies, url);
	let existing = list.find((item) => item.did === pending.did);
	if (pending.kind === 'chat') {
		existing = list.find((item) => item.id === pending.source);
		if (!existing || existing.did !== pending.did)
			throw new Error('Messaging source account unavailable');
	}
	if (!existing && list.length >= 8) throw new Error('Account limit reached');
	// Every credential activation gets a distinct cookie name. A late response for
	// an old activation cannot overwrite or delete the new bearer.
	const id = existing?.id ?? crypto.randomUUID();
	const value: Credential = {
		id,
		generation: crypto.randomUUID(),
		token: pending.token,
		did: pending.did,
		handle: pending.handle
	};
	const current = selected(cookies, url)?.id || '';
	if (existing) remove(cookies, url, existing.id);
	cookies.set(credentialCookie(url, value), encode(value), options(url));
	if (!current || current === pending.active || current === existing?.id)
		cookies.set(prefix(url) + 'active', id, options(url));
	cookies.delete(prefix(url) + 'pending', { path: '/' });
	return value;
}
export function nonce() {
	return Buffer.from(crypto.getRandomValues(new Uint8Array(32))).toString(
		'base64url'
	);
}
export function sameState(a: string | undefined, b: string | undefined) {
	if (!capability(a) || !capability(b)) return false;
	let diff = 0;
	for (let i = 0; i < 43; i++) diff |= a.charCodeAt(i) ^ b.charCodeAt(i);
	return diff === 0;
}
