import { graphemeCount } from '#lib/text.ts';
import type { ChatMessage, ChatReceipt } from '#lib/api/client.ts';

export interface FrozenSend {
	operationId: string;
	text: string;
	state: 'pending' | 'uncertain' | 'failed';
}
export interface MessageDraft {
	text: string;
	pending?: FrozenSend;
}
export interface FrozenReport {
	operationId: string;
	reasonType: string;
	reason: string;
	state: 'pending' | 'uncertain' | 'failed';
}
export interface StorageLike {
	getItem(key: string): string | null;
	setItem(key: string, value: string): void;
	removeItem(key: string): void;
}
const operationID =
	/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const states = new Set(['pending', 'uncertain', 'failed']);
const encoder = new TextEncoder();
export { graphemeCount } from '#lib/text.ts';
export function validateText(text: string, report = false): string | undefined {
	if (!text.isWellFormed())
		return 'Remove the invalid character before continuing.';
	if (!report && !text.trim()) return 'Write a message before sending.';
	if (
		encoder.encode(text).length > (report ? 20_000 : 10_000) ||
		graphemeCount(text) > (report ? 2_000 : 1_000)
	)
		return report
			? 'Use at most 2,000 characters and 20,000 UTF-8 bytes.'
			: 'Use at most 1,000 characters and 10,000 UTF-8 bytes.';
	return undefined;
}

/** Drafts are private browser data, partitioned by the exact account session. */
export class MessageDraftStore {
	constructor(
		private readonly storage: StorageLike,
		private readonly account: string
	) {}
	private key(id: string): string {
		return `craftsky.chat.v1:${encodeURIComponent(this.account)}:${encodeURIComponent(id)}`;
	}
	load(id: string): MessageDraft {
		const raw = this.storage.getItem(this.key(id));
		if (!raw) return { text: '' };
		try {
			if (raw.length > 50_000) throw new Error('Oversized draft');
			const value = JSON.parse(raw) as MessageDraft;
			if (
				!value ||
				typeof value.text !== 'string' ||
				value.text.length > 20_000
			)
				throw new Error('Invalid draft');
			if (
				value.pending &&
				(!operationID.test(value.pending.operationId) ||
					typeof value.pending.text !== 'string' ||
					validateText(value.pending.text) ||
					!states.has(value.pending.state))
			)
				throw new Error('Invalid attempt');
			return {
				text: value.text,
				...(value.pending
					? {
							pending: {
								...value.pending,
								state:
									value.pending.state === 'pending'
										? 'uncertain'
										: value.pending.state
							}
						}
					: {})
			};
		} catch {
			throw new Error('The saved message draft could not be read.');
		}
	}
	save(id: string, draft: MessageDraft): void {
		if (!draft.text && !draft.pending) this.storage.removeItem(this.key(id));
		else this.storage.setItem(this.key(id), JSON.stringify(draft));
	}
	clear(id: string): void {
		this.storage.removeItem(this.key(id));
	}
	loadReport(id: string, message?: string): FrozenReport | undefined {
		const raw = this.storage.getItem(this.key(`report:${id}:${message || ''}`));
		if (!raw) return;
		if (raw.length > 50_000) throw new Error('Invalid saved report');
		const value = JSON.parse(raw) as FrozenReport;
		if (
			!value ||
			!operationID.test(value.operationId) ||
			!states.has(value.state) ||
			typeof value.reason !== 'string' ||
			validateText(value.reason, true) ||
			!reportReasons.some((item) => item[1] === value.reasonType)
		)
			throw new Error('Invalid saved report');
		return {
			...value,
			state: value.state === 'pending' ? 'uncertain' : value.state
		};
	}
	saveReport(
		id: string,
		message: string | undefined,
		report: FrozenReport | undefined
	): void {
		const key = this.key(`report:${id}:${message || ''}`);
		if (report) this.storage.setItem(key, JSON.stringify(report));
		else this.storage.removeItem(key);
	}
}

/** Freeze and persist this value before starting the request. Retries reuse it. */
export function freezeSend(
	draft: MessageDraft,
	newID: () => string = () => crypto.randomUUID()
): MessageDraft {
	if (draft.pending) throw new Error('Resolve the existing attempt first.');
	const invalid = validateText(draft.text);
	if (invalid) throw new Error(invalid);
	const id = newID();
	if (!operationID.test(id)) throw new Error('Invalid message operation.');
	return {
		text: draft.text,
		pending: { operationId: id, text: draft.text, state: 'pending' }
	};
}

export function reconcileSend(
	draft: MessageDraft,
	receipt: ChatReceipt
): MessageDraft {
	const attempt = draft.pending;
	if (
		!attempt ||
		receipt.operationId !== attempt.operationId ||
		receipt.kind !== 'send' ||
		!['confirmed', 'failed', 'uncertain'].includes(receipt.state) ||
		(receipt.state === 'confirmed' && !receipt.messageId)
	)
		throw new Error('The messaging receipt did not match this attempt.');
	if (receipt.state === 'confirmed')
		return { text: draft.text === attempt.text ? '' : draft.text };
	return { ...draft, pending: { ...attempt, state: receipt.state } };
}

export function mergeMessages(
	current: ChatMessage[],
	incoming: ChatMessage[]
): ChatMessage[] {
	const values = new Map(current.map((message) => [message.id, message]));
	for (const message of incoming) values.set(message.id, message);
	return [...values.values()].sort(
		(a, b) =>
			Date.parse(a.sentAt) - Date.parse(b.sentAt) || a.id.localeCompare(b.id)
	);
}

export const conversationURL = (id: string): string =>
	`/app/messages/${encodeURIComponent(id)}`;
export const actorLabel = (actor: {
	displayName?: string | null;
	handle?: string | null;
}): string => actor.displayName || actor.handle || 'Someone';
export const messageText = (message: ChatMessage): string =>
	message.kind === 'deleted'
		? 'Message deleted'
		: message.kind === 'text'
			? message.text || ''
			: 'Unsupported message';
export const safeAvatar = (raw?: string | null): string | undefined => {
	if (!raw) return undefined;
	try {
		const url = new URL(raw);
		return url.protocol === 'https:' && !url.username && !url.password
			? url.href
			: undefined;
	} catch {
		return undefined;
	}
};
export const reportReasons = [
	['Spam', 'com.atproto.moderation.defs#reasonSpam'],
	[
		'Illegal or prohibited content',
		'com.atproto.moderation.defs#reasonViolation'
	],
	['Misleading content', 'com.atproto.moderation.defs#reasonMisleading'],
	['Sexual content', 'com.atproto.moderation.defs#reasonSexual'],
	['Harassment or abuse', 'com.atproto.moderation.defs#reasonRude'],
	['Other', 'com.atproto.moderation.defs#reasonOther']
] as const;

export interface MessageLink {
	text: string;
	href?: string;
}
export function messageLinks(text: string): MessageLink[] {
	const parts: MessageLink[] = [];
	let offset = 0;
	for (const match of text.matchAll(/https?:\/\/[^\s<>"\u0000-\u0020]+/gi)) {
		const raw = match[0].replace(/[.,;!)\]}]+$/, '');
		try {
			const url = new URL(raw);
			if (!url.hostname || url.username || url.password) continue;
			if (match.index! > offset)
				parts.push({ text: text.slice(offset, match.index) });
			parts.push({ text: raw, href: url.href });
			offset = match.index! + raw.length;
		} catch {
			continue;
		}
	}
	if (offset < text.length) parts.push({ text: text.slice(offset) });
	return parts;
}
export interface CraftskyMessageLink {
	kind: 'profile' | 'post' | 'event';
	did: string;
	rkey?: string;
	focus?: string;
	href: string;
}
export function craftskyMessageLink(
	text: string
): CraftskyMessageLink | undefined {
	for (const link of messageLinks(text)) {
		if (!link.href) continue;
		const url = new URL(link.href);
		if (url.origin !== 'https://app.craftsky.social' || url.hash) continue;
		try {
			const path = url.pathname.split('/').slice(1).map(decodeURIComponent);
			const did = path[1];
			if (!/^did:[a-z]+:[^/\s?#]+$/.test(did || '')) continue;
			if (path.length === 2 && path[0] === 'profiles' && !url.search)
				return {
					kind: 'profile',
					did,
					href: `/app/profile/${encodeURIComponent(did)}`
				};
			if (
				path.length !== 3 ||
				!['posts', 'events'].includes(path[0]) ||
				!path[2] ||
				/[\s/?#]/.test(path[2])
			)
				continue;
			let focus: string | undefined;
			if (url.search) {
				if (
					path[0] !== 'posts' ||
					url.searchParams.size !== 1 ||
					url.searchParams.getAll('focus').length !== 1
				)
					continue;
				focus = url.searchParams.get('focus')!;
				if (
					!/^at:\/\/did:[a-z]+:[^/\s]+\/social\.craftsky\.feed\.post\/[^/\s?#]+$/.test(
						focus
					)
				)
					continue;
			}
			return {
				kind: path[0] === 'posts' ? 'post' : 'event',
				did,
				rkey: path[2],
				focus,
				href: `/app/${path[0]}/${encodeURIComponent(did)}/${encodeURIComponent(path[2])}${focus ? '?focus=' + encodeURIComponent(focus) : ''}`
			};
		} catch {
			continue;
		}
	}
}

/** Only a complete, recognized public CraftSky URL may enter the share bridge. */
export function sharedCraftskyLink(
	raw: string | undefined | null
): string | undefined {
	if (!raw || raw.length > 2048 || /[\s\u0000-\u001f\u007f\\]/.test(raw))
		return;
	const parts = messageLinks(raw);
	if (
		parts.length !== 1 ||
		parts[0].text !== raw ||
		!parts[0].href ||
		!craftskyMessageLink(raw)
	)
		return;
	return parts[0].href;
}

export function prefillSharedLink(
	draft: MessageDraft,
	link: string
): MessageDraft {
	const safe = sharedCraftskyLink(link);
	if (!safe)
		throw new Error('Open the original CraftSky content to share this link.');
	if (messageLinks(draft.text).some((part) => part.href === safe)) return draft;
	// An existing delivery keeps its frozen text/ID; the shared link becomes a separate editable draft.
	const text =
		draft.pending && draft.text === draft.pending.text ? '' : draft.text;
	return { ...draft, text: text ? `${text}\n\n${safe}` : safe };
}
