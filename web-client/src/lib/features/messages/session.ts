import {
	APIError,
	type ChatConversation,
	type ChatDetails,
	type ChatMessage,
	type ChatStatus,
	type CraftskyAPI,
	type Page
} from '#lib/api/client.ts';
import { reloadPages } from '#lib/pagination.ts';
import { mergeMessages } from './model';

export const messagesSessionContext = Symbol('messages-session');
export type InboxBucket = 'accepted' | 'request';
export type UnreadCounts = { accepted: number; requests: number };
export const emptyUnread = (): UnreadCounts => ({ accepted: 0, requests: 0 });
const statusFreshMs = 30_000;
const dataFreshMs = 30_000;
const threadLifetimeMs = 120_000;
const recentThreadLimit = 5;
const recentMessageLimit = 50;
const newerMessage = (
	current: ChatMessage | undefined,
	candidate: ChatMessage
) =>
	current && Date.parse(current.sentAt) > Date.parse(candidate.sentAt)
		? current
		: candidate;

type Snapshot<T> = { data: T; time: number; revision?: number };
type Inbox = Snapshot<Page<ChatConversation>> & { scroll: number };
type Thread = Snapshot<ChatDetails> & {
	readID?: string;
	deletedIDs: Set<string>;
	latestSend?: ChatMessage;
};

export function statusAllowsReading(status: ChatStatus | undefined): boolean {
	return (
		!!status?.enabled &&
		status.capabilities.canRead &&
		['ready', 'unavailable'].includes(status.state)
	);
}

export function chatAccessDenied(error: unknown): boolean {
	return (
		error instanceof APIError &&
		(error.status === 401 ||
			error.status === 403 ||
			[
				'chat_authorization_required',
				'chat_restricted',
				'chat_disconnected',
				'chat_connection_changed'
			].includes(error.code))
	);
}

export function chatConversationUnavailable(error: unknown): boolean {
	return (
		error instanceof APIError && error.code === 'chat_conversation_unavailable'
	);
}

function canceled(): DOMException {
	return new DOMException('Messaging view changed', 'AbortError');
}

/** Owns transient reads for a single authenticated shell activation. */
export class MessagesSession {
	private active = true;
	private epoch = 0;
	private status?: Snapshot<ChatStatus>;
	private inboxes = new Map<InboxBucket, Inbox>();
	private threads = new Map<string, Thread>();
	private pending = new Map<string, Promise<unknown>>();
	private counts = emptyUnread();
	private countTime?: number;
	private heartbeatQueue: Promise<unknown> = Promise.resolve();
	private countListeners = new Set<(counts: UnreadCounts) => void>();
	private accessListeners = new Set<() => void>();
	private resumeAt = 0;
	private rateLimit?: APIError;

	constructor(
		readonly api: CraftskyAPI,
		private now = Date.now
	) {}

	get unread(): UnreadCounts {
		return this.counts;
	}

	subscribeCounts(listener: (counts: UnreadCounts) => void): () => void {
		this.countListeners.add(listener);
		listener(this.counts);
		return () => this.countListeners.delete(listener);
	}

	subscribeAccessInvalidation(listener: () => void): () => void {
		this.accessListeners.add(listener);
		return () => this.accessListeners.delete(listener);
	}

	private publishCounts(counts: UnreadCounts) {
		this.counts = counts;
		for (const listener of this.countListeners) listener(counts);
	}

	captureAuthority(): () => boolean {
		const epoch = this.epoch;
		return () => this.active && epoch === this.epoch;
	}

	private fence() {
		this.epoch++;
		this.pending.clear();
	}

	invalidateAccess() {
		this.fence();
		this.status = undefined;
		this.inboxes.clear();
		this.threads.clear();
		this.countTime = undefined;
		this.publishCounts(emptyUnread());
		for (const listener of this.accessListeners) listener();
	}

	invalidateRelationships() {
		for (const key of this.pending.keys())
			if (key !== 'status') this.pending.delete(key);
		this.inboxes.clear();
		this.threads.clear();
		this.countTime = undefined;
	}

	dispose() {
		this.active = false;
		this.countListeners.clear();
		this.accessListeners.clear();
		this.invalidateAccess();
	}

	private invalidateReads(id: string, counts = false) {
		for (const key of this.pending.keys()) {
			if (
				key === 'details:' + id ||
				key.startsWith('history:' + id + ':') ||
				key.startsWith('inbox:') ||
				key.startsWith('inbox-page:') ||
				(counts && key === 'counts')
			)
				this.pending.delete(key);
		}
	}

	private request<T>(
		key: string,
		load: () => Promise<T>,
		accept?: (result: T) => T | void,
		retry = true
	): Promise<T> {
		if (!this.active) return Promise.reject(canceled());
		if (this.rateLimit && this.isBackingOff())
			return Promise.reject(this.rateLimit);
		const previous = this.pending.get(key);
		if (previous) return previous as Promise<T>;
		const epoch = this.epoch;
		const request = load()
			.then((result) => {
				if (!this.active || epoch !== this.epoch) throw canceled();
				if (this.pending.get(key) !== request) throw canceled();
				return accept?.(result) ?? result;
			})
			.catch((error: unknown) => {
				if (!this.active || epoch !== this.epoch) throw canceled();
				if (this.pending.get(key) !== request)
					return retry
						? this.request(key, load, accept, false)
						: Promise.reject(canceled());
				if (chatAccessDenied(error)) this.invalidateAccess();
				this.observeRateLimit(error);
				throw error;
			})
			.finally(() => {
				if (this.pending.get(key) === request) this.pending.delete(key);
			});
		this.pending.set(key, request);
		return request;
	}

	isBackingOff(): boolean {
		return this.now() < this.resumeAt;
	}

	observeRateLimit(error: unknown) {
		if (
			!(error instanceof APIError) ||
			error.status !== 429 ||
			error === this.rateLimit
		)
			return;
		const resumeAt = this.now() + (error.retryAfter ?? 10) * 1000;
		if (resumeAt >= this.resumeAt) {
			this.resumeAt = resumeAt;
			this.rateLimit = error;
		}
	}

	setStatus(status: ChatStatus) {
		const previous = this.status?.data;
		if (
			previous?.capabilities.canRead &&
			(previous.generation !== status.generation ||
				!status.enabled ||
				!status.capabilities.canRead ||
				['authorizationNeeded', 'disabled', 'restricted'].includes(
					status.state
				))
		)
			this.invalidateAccess();
		this.status = { data: status, time: this.now() };
	}

	get cachedStatus(): ChatStatus | undefined {
		return this.status?.data;
	}

	loadStatus(force = false): Promise<ChatStatus> {
		if (this.status && !force && this.now() - this.status.time < statusFreshMs)
			return Promise.resolve(this.status.data);
		return this.request(
			'status',
			() => this.api.chat.status(),
			(status) => this.setStatus(status)
		);
	}

	cachedInbox(bucket: InboxBucket): Inbox | undefined {
		return this.inboxes.get(bucket);
	}

	async warmInbox(): Promise<void> {
		const status = await this.loadStatus();
		if (status.state === 'ready' && statusAllowsReading(status))
			await this.loadInbox('accepted');
	}

	loadInbox(
		bucket: InboxBucket,
		force = false
	): Promise<Page<ChatConversation>> {
		const cached = this.cachedInbox(bucket);
		if (cached && !force && this.now() - cached.time < dataFreshMs)
			return Promise.resolve(cached.data);
		return this.request(
			'inbox:' + bucket,
			() =>
				reloadPages(
					(cursor) => this.inboxPage(bucket, cursor),
					1,
					(item) => item.id,
					cached?.data.items.length
				),
			(data) => this.rememberInbox(bucket, data)
		);
	}

	rememberInbox(bucket: InboxBucket, data: Page<ChatConversation>) {
		const previous = this.inboxes.get(bucket);
		this.inboxes.set(bucket, {
			data,
			scroll: previous?.scroll ?? 0,
			revision: previous?.revision,
			time: this.now()
		});
	}

	rememberRevision(
		bucket: InboxBucket | undefined,
		id: string | undefined,
		revision: number
	) {
		const snapshot = id
			? this.threads.get(id)
			: bucket && this.inboxes.get(bucket);
		if (snapshot) snapshot.revision = revision;
	}

	rememberInboxScroll(bucket: InboxBucket, scroll: number) {
		const inbox = this.cachedInbox(bucket);
		if (inbox) inbox.scroll = scroll;
	}

	async loadCounts(force = false): Promise<UnreadCounts> {
		const status = await this.loadStatus();
		if (!statusAllowsReading(status)) return this.counts;
		if (
			!force &&
			this.countTime !== undefined &&
			this.now() - this.countTime < dataFreshMs
		)
			return this.counts;
		return this.request(
			'counts',
			() => this.api.chat.unread(),
			(counts) => {
				this.countTime = this.now();
				this.publishCounts(counts);
			}
		);
	}

	heartbeat(
		view: 'conversation' | 'inbox' | 'none',
		id?: string,
		knownRevision?: number
	) {
		const epoch = this.epoch;
		const result = this.heartbeatQueue.then(async () => {
			if (!this.active || epoch !== this.epoch || this.isBackingOff())
				throw canceled();
			try {
				const snapshot = await this.api.chat.sync(view, id, knownRevision);
				if (!this.active || epoch !== this.epoch) throw canceled();
				return snapshot;
			} catch (error) {
				if (!this.active || epoch !== this.epoch) throw canceled();
				if (chatAccessDenied(error)) this.invalidateAccess();
				this.observeRateLimit(error);
				throw error;
			}
		});
		this.heartbeatQueue = result.catch(() => {});
		return result;
	}

	loadOlderMessages(id: string, cursor: string) {
		return this.request('history:' + id + ':' + cursor, async () => {
			const page = await this.api.chat.messages(id, { cursor });
			return {
				...page,
				items: page.items.map((item) => this.displayMessage(id, item))
			};
		});
	}

	private async inboxPage(bucket: InboxBucket, cursor?: string | null) {
		const page = await this.api.chat.conversations(bucket, { cursor });
		return {
			...page,
			items: page.items.map((item) => this.displayConversation(item))
		};
	}

	loadMoreInbox(bucket: InboxBucket, cursor: string) {
		return this.request('inbox-page:' + bucket + ':' + cursor, () =>
			this.inboxPage(bucket, cursor)
		);
	}

	cachedThread(id: string): Thread | undefined {
		const thread = this.threads.get(id);
		if (!thread || this.now() - thread.time >= threadLifetimeMs) {
			this.threads.delete(id);
			return;
		}
		this.threads.delete(id);
		this.threads.set(id, thread);
		return thread;
	}

	inboxConversation(id: string): ChatConversation | undefined {
		for (const bucket of ['accepted', 'request'] as const) {
			const row = this.cachedInbox(bucket)?.data.items.find(
				(item) => item.id === id
			);
			if (row) return row;
		}
	}

	loadDetails(id: string): Promise<ChatDetails> {
		return this.request(
			'details:' + id,
			() => this.api.chat.details(id),
			(details) => {
				if (details.conversation.id !== id)
					throw new Error('Invalid conversation response');
				const thread = this.threads.get(id);
				const messages = details.messages.items.map((item) =>
					this.displayMessage(id, item)
				);
				const pending = thread?.latestSend;
				const acknowledged =
					pending && messages.some((message) => message.id === pending.id);
				const comparablePage =
					!details.messages.cursor ||
					(pending &&
						messages.some(
							(message) =>
								Date.parse(message.sentAt) <= Date.parse(pending.sentAt)
						));
				const conversation = this.displayConversation(details.conversation);
				if (thread && (acknowledged || !comparablePage))
					thread.latestSend = undefined;
				return {
					conversation,
					messages: {
						...details.messages,
						items:
							pending && comparablePage && !acknowledged
								? mergeMessages(messages, [pending])
								: messages
					}
				};
			}
		);
	}

	private displayMessage(id: string, item: ChatMessage): ChatMessage {
		return this.threads.get(id)?.deletedIDs.has(item.id)
			? { ...item, kind: 'deleted', text: undefined, senderDid: undefined }
			: item;
	}

	private displayConversation(
		conversation: ChatConversation
	): ChatConversation {
		const pending = this.threads.get(conversation.id)?.latestSend;
		const lastMessage = pending
			? newerMessage(conversation.lastMessage, pending)
			: conversation.lastMessage;
		return lastMessage
			? {
					...conversation,
					lastMessage: this.displayMessage(conversation.id, lastMessage)
				}
			: conversation;
	}

	rememberThread(details: ChatDetails) {
		if (!this.active) return;
		const { conversation, messages } = details;
		const id = conversation.id;
		const existing = this.threads.get(id);
		const items = mergeMessages([], messages.items);
		const trimmed = items.length > recentMessageLimit;
		this.threads.delete(id);
		this.threads.set(id, {
			data: {
				conversation,
				messages: {
					cursor: trimmed ? undefined : messages.cursor,
					items: items.slice(-recentMessageLimit)
				}
			},
			time: this.now(),
			readID: existing?.readID,
			revision: existing?.revision,
			deletedIDs: existing?.deletedIDs ?? new Set(),
			latestSend: existing?.latestSend
		});
		while (this.threads.size > recentThreadLimit)
			this.threads.delete(this.threads.keys().next().value!);
	}

	markRead(conversation: ChatConversation, messageID: string) {
		this.invalidateReads(conversation.id, true);
		const thread = this.threads.get(conversation.id);
		if (thread) thread.readID = messageID;
		this.updateConversation(conversation, {
			unreadCount: conversation.unreadCount
		});
		this.countTime = undefined;
	}

	confirmedSend(
		conversation: ChatConversation,
		message: ChatMessage
	): ChatConversation {
		this.invalidateReads(conversation.id);
		const thread = this.threads.get(conversation.id);
		const lastMessage = newerMessage(conversation.lastMessage, message);
		const preview = { ...conversation, lastMessage };
		if (thread) {
			thread.latestSend = newerMessage(thread.latestSend, message);
			this.rememberThread({
				conversation: preview,
				messages: {
					...thread.data.messages,
					items: mergeMessages(thread.data.messages.items, [message])
				}
			});
		}
		for (const inbox of this.inboxes.values()) {
			const row = inbox.data.items.find((item) => item.id === conversation.id);
			if (!row || newerMessage(row.lastMessage, message) !== message) continue;
			inbox.data = {
				...inbox.data,
				items: [
					{ ...row, lastMessage: message },
					...inbox.data.items.filter((item) => item.id !== row.id)
				]
			};
		}
		return preview;
	}

	private updateConversation(
		conversation: ChatConversation,
		patch: Partial<ChatConversation>
	) {
		const thread = this.threads.get(conversation.id);
		if (thread)
			thread.data = {
				...thread.data,
				conversation: { ...thread.data.conversation, ...patch }
			};
		for (const inbox of this.inboxes.values()) {
			const row = inbox.data.items.find((item) => item.id === conversation.id);
			if (!row) continue;
			inbox.data = {
				...inbox.data,
				items: inbox.data.items.map((item) =>
					item.id === row.id ? { ...item, ...patch } : item
				)
			};
		}
	}

	changeConversation(conversation: ChatConversation) {
		this.invalidateReads(conversation.id, true);
		this.updateConversation(conversation, { muted: conversation.muted });
		this.countTime = undefined;
	}

	acceptConversation(conversation: ChatConversation) {
		const thread = this.threads.get(conversation.id);
		this.removeThread(conversation.id);
		if (thread) {
			thread.data = { ...thread.data, conversation };
			this.threads.set(conversation.id, thread);
		}
		const inbox = this.cachedInbox('accepted');
		if (inbox)
			inbox.data = {
				...inbox.data,
				items: [conversation, ...inbox.data.items]
			};
	}

	removeThread(id: string) {
		this.invalidateReads(id, true);
		this.threads.delete(id);
		this.countTime = undefined;
		for (const inbox of this.inboxes.values()) {
			inbox.data = {
				...inbox.data,
				items: inbox.data.items.filter((item) => item.id !== id)
			};
			inbox.time = this.now() - dataFreshMs;
		}
	}

	deleteMessage(id: string, messageID: string) {
		this.invalidateReads(id);
		const thread = this.threads.get(id);
		if (!thread) return;
		thread.deletedIDs.add(messageID);
		if (thread.latestSend?.id === messageID) thread.latestSend = undefined;
		const tombstone = (item: ChatMessage) => this.displayMessage(id, item);
		thread.data = {
			...thread.data,
			messages: {
				...thread.data.messages,
				items: thread.data.messages.items.map(tombstone)
			}
		};
		if (thread.data.conversation.lastMessage?.id === messageID) {
			this.updateConversation(thread.data.conversation, {
				lastMessage: tombstone(thread.data.conversation.lastMessage)
			});
		}
	}
}
