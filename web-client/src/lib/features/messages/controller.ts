import { stageSharedLink, takeSharedLink } from './share';
import { reportFailure } from '../diagnostics/client';
import type { DiagnosticOperation } from '../diagnostics/privacy';
import { userErrorMessage } from '#lib/user-facing-error.ts';
import { isCanceled } from '#lib/api/error-message.ts';
import {
	MessagesSession,
	chatAccessDenied,
	chatConversationUnavailable,
	emptyUnread,
	statusAllowsReading
} from './session';
import {
	APIError,
	type CraftskyAPI,
	type Author,
	type ChatConversation,
	type ChatDetails,
	type ChatMessage,
	type ChatStatus,
	type ChatIncomingSetting,
	type ChatReceipt
} from '#lib/api/client.ts';
import {
	MessageDraftStore,
	freezeSend,
	reconcileSend,
	mergeMessages,
	validateText,
	reportReasons,
	sharedCraftskyLink,
	prefillSharedLink,
	type MessageDraft,
	type FrozenReport,
	type StorageLike
} from './model';

export type MessageView =
	'inbox' | 'new' | 'settings' | 'conversation' | 'report';
export interface MessagesState {
	view: MessageView;
	bucket: 'accepted' | 'request';
	id?: string;
	reportMessage?: string;
	loading: boolean;
	confirming: boolean;
	inboxScroll: number;
	busy: boolean;
	loadingMore: boolean;
	error: string;
	notice: string;
	status?: ChatStatus;
	counts: { accepted: number; requests: number };
	conversations: ChatConversation[];
	cursor?: string | null;
	conversation?: ChatConversation;
	peerBlocked: boolean;
	messages: ChatMessage[];
	historyCursor?: string | null;
	draft: MessageDraft;
	recipients: Author[];
	recipientCursor?: string | null;
	policy?: ChatIncomingSetting;
	push?: boolean;
	report?: FrozenReport;
}
const initial = (): MessagesState => ({
	view: 'inbox',
	bucket: 'accepted',
	loading: true,
	confirming: false,
	inboxScroll: 0,
	busy: false,
	loadingMore: false,
	error: '',
	notice: '',
	counts: emptyUnread(),
	conversations: [],
	peerBlocked: false,
	messages: [],
	draft: { text: '' },
	recipients: []
});
const errorMessage = (error: unknown): string =>
	error instanceof APIError && error.code === 'chat_relationship_unavailable'
		? 'This person cannot receive a new message from you right now.'
		: userErrorMessage(error);

/** One immutable account activation owns all async results, composition and polling. */
export class MessagesController {
	state = initial();
	private epoch = 0;
	private disposed = false;
	private store: MessageDraftStore;
	private route = '';
	private syncing = false;
	private syncFailed = false;
	private reportSaveFailed = false;
	private syncRevision?: number;
	private lastReconciled = 0;
	private unsubscribeCounts: () => void;
	private unsubscribeAccess: () => void;
	private viewing = false;
	private searchEpoch = 0;
	private readID?: string;
	private pendingRead?: string;
	private readTarget?: string;
	private readingEpoch?: number;
	private blockAttempt?: { did: string; key: string };
	constructor(
		readonly api: CraftskyAPI,
		readonly account: string,
		storage: StorageLike,
		private publish: (state: MessagesState) => void,
		private session = new MessagesSession(api)
	) {
		this.store = new MessageDraftStore(storage, account);
		this.unsubscribeCounts = session.subscribeCounts((counts) =>
			this.update({ counts })
		);
		this.unsubscribeAccess = session.subscribeAccessInvalidation(() =>
			this.update({
				status: undefined,
				conversation: undefined,
				messages: [],
				conversations: [],
				recipients: [],
				draft: { text: '' },
				cursor: undefined,
				historyCursor: undefined
			})
		);
	}
	private current(epoch: number): boolean {
		return !this.disposed && this.epoch === epoch;
	}
	private update(patch: Partial<MessagesState>): void {
		if (!this.disposed) {
			this.state = { ...this.state, ...patch };
			this.publish(this.state);
		}
	}
	private failure(error: unknown, operation: DiagnosticOperation): void {
		if (isCanceled(error)) return;
		void reportFailure(error, operation);
		this.session.observeRateLimit(error);
		if (chatAccessDenied(error) || chatConversationUnavailable(error)) {
			if (chatAccessDenied(error)) this.session.invalidateAccess();
			else if (this.state.id) this.session.removeThread(this.state.id);
			this.update({
				conversation: undefined,
				messages: [],
				conversations: [],
				recipients: [],
				error:
					'Messaging access changed. Refresh or review your messaging permissions.'
			});
		} else this.update({ error: errorMessage(error) });
	}
	dispose(): void {
		void this.sync(false);
		this.disposed = true;
		this.epoch++;
		this.searchEpoch++;
		this.unsubscribeCounts();
		this.unsubscribeAccess();
	}
	rememberInboxScroll(scroll: number) {
		if (this.state.view === 'inbox')
			this.session.rememberInboxScroll(this.state.bucket, scroll);
	}
	async navigate(path: string[], force = false): Promise<void> {
		const nested = path[0] === 'messages' ? path.slice(1) : path;
		const parts = nested[0] === 'conversations' ? nested.slice(1) : nested;
		const route = parts.join('/');
		if (!force && route === this.route && !this.state.loading) return;
		this.route = route;
		const epoch = ++this.epoch;
		this.searchEpoch++;
		this.syncRevision = undefined;
		this.readID = undefined;
		this.pendingRead = undefined;
		this.readTarget = undefined;
		const first = parts[0];
		const reserved: MessageView[] = ['new', 'settings'];
		const view: MessageView =
			!first || first === 'requests'
				? 'inbox'
				: reserved.includes(first as MessageView)
					? (first as MessageView)
					: parts[1] === 'report'
						? 'report'
						: 'conversation';
		const id = ['conversation', 'report'].includes(view) ? first : undefined;
		const bucket = first === 'requests' ? 'request' : 'accepted';
		const cachedInbox =
			view === 'inbox' ? this.session.cachedInbox(bucket) : undefined;
		const cachedThread =
			view === 'conversation' && id ? this.session.cachedThread(id) : undefined;
		const recents =
			view === 'new' ? this.session.cachedInbox('accepted') : undefined;
		const isWarm = !!(cachedInbox || cachedThread || recents);
		this.syncRevision = cachedInbox?.revision ?? cachedThread?.revision;
		this.readID = cachedThread?.readID;
		this.update({
			...initial(),
			view,
			bucket,
			id,
			counts: this.session.unread,
			loading: !isWarm,
			status: this.session.cachedStatus,
			confirming: !!cachedThread,
			inboxScroll: cachedInbox?.scroll ?? 0,
			conversations: (cachedInbox || recents)?.data.items ?? [],
			cursor: cachedInbox?.data.cursor,
			conversation:
				cachedThread?.data.conversation ??
				(id ? this.session.inboxConversation(id) : undefined),
			messages: cachedThread?.data.messages.items ?? [],
			historyCursor: cachedThread?.data.messages.cursor,
			reportMessage: view === 'report' ? parts[2] : undefined
		});
		try {
			const status = await this.session.loadStatus(force && !isWarm);
			if (!this.current(epoch)) return;
			this.update({ status, push: status.pushEnabled });
			if (view === 'settings') {
				const [policy, push] = await Promise.allSettled([
					this.api.chat.incoming(),
					this.api.chat.pushPreference()
				]);
				if (!this.current(epoch)) return;
				this.update({
					policy: policy.status === 'fulfilled' ? policy.value : undefined,
					push:
						push.status === 'fulfilled'
							? push.value.pushEnabled
							: status.pushEnabled
				});
			} else if (statusAllowsReading(status)) {
				if (id) {
					const details = await this.session.loadDetails(id);
					if (!this.current(epoch)) return;
					if (view === 'conversation') this.session.rememberThread(details);
					let draft: MessageDraft = { text: '' };
					try {
						draft = this.store.load(id);
					} catch (error) {
						void reportFailure(error, 'messages.draft.load');
					}
					const link =
						view === 'conversation'
							? takeSharedLink(this.account, id)
							: undefined;
					if (link) draft = prefillSharedLink(draft, link);
					this.update({
						conversation: details.conversation,
						messages: mergeMessages([], details.messages.items),
						historyCursor: details.messages.cursor,
						draft,
						confirming: false
					});
					if (view === 'report')
						this.update({ report: this.store.loadReport(id, parts[2]) });
					if (draft.pending?.state === 'uncertain') void this.checkSend();
				} else if (view === 'inbox') {
					void this.refreshCounts(epoch);
					await this.refreshInbox(epoch, false);
				} else if (view === 'new') {
					const recent = await this.session.loadInbox('accepted');
					if (this.current(epoch))
						this.update({ conversations: recent.items.slice(0, 20) });
				}
			} else {
				this.update({
					conversation: undefined,
					messages: [],
					conversations: []
				});
			}
			if (this.current(epoch)) this.lastReconciled = Date.now();
		} catch (error) {
			if (this.current(epoch)) this.failure(error, 'messages.navigate');
		} finally {
			if (this.current(epoch)) this.update({ loading: false });
		}
	}
	private async refreshInbox(epoch: number, force = true): Promise<void> {
		const page = await this.session.loadInbox(this.state.bucket, force);
		if (this.current(epoch))
			this.update({
				conversations: page.items,
				cursor: page.cursor,
				error: ''
			});
	}
	private async refreshCounts(epoch: number, force = false): Promise<void> {
		try {
			await this.session.loadCounts(force);
		} catch (error) {
			if (!this.current(epoch) || isCanceled(error)) return;
			if (chatAccessDenied(error)) this.failure(error, 'messages.navigate');
			else void reportFailure(error, 'startup.count.messages');
		}
	}
	async refresh(): Promise<void> {
		const epoch = this.epoch;
		try {
			if (
				this.state.view === 'inbox' &&
				statusAllowsReading(this.state.status)
			) {
				void this.refreshCounts(epoch, true);
				await this.refreshInbox(epoch);
				return;
			}
			if (
				this.state.view === 'conversation' &&
				this.state.conversation &&
				!this.state.confirming
			) {
				await this.refreshContent(epoch);
				return;
			}
			await this.navigate(this.route.split('/').filter(Boolean), true);
		} catch (error) {
			if (this.current(epoch)) this.failure(error, 'messages.navigate');
		}
	}
	async more(): Promise<void> {
		if (this.state.loadingMore || this.state.confirming) return;
		const epoch = this.epoch;
		const { id, cursor, historyCursor } = this.state;
		if (!(id ? historyCursor : cursor)) return;
		this.update({ loadingMore: true });
		try {
			if (id) {
				const page = await this.session.loadOlderMessages(id, historyCursor!);
				if (this.current(epoch))
					this.update({
						messages: mergeMessages(this.state.messages, page.items),
						historyCursor: page.cursor
					});
			} else {
				const page = await this.session.loadMoreInbox(
					this.state.bucket,
					cursor!
				);
				if (this.current(epoch)) {
					const items = new Map(
						this.state.conversations.map((item) => [item.id, item])
					);
					for (const item of page.items) items.set(item.id, item);
					const conversations = [...items.values()];
					this.session.rememberInbox(this.state.bucket, {
						items: conversations,
						cursor: page.cursor
					});
					this.update({ conversations, cursor: page.cursor });
				}
			}
		} catch (error) {
			if (this.current(epoch)) {
				// Continuations can expire or be evicted while a snapshot is kept in memory.
				if (error instanceof APIError && error.status === 400) {
					try {
						if (id) {
							const details = await this.session.loadDetails(id);
							if (this.current(epoch)) {
								this.session.rememberThread(details);
								this.update({
									conversation: details.conversation,
									messages: mergeMessages([], details.messages.items),
									historyCursor: details.messages.cursor
								});
							}
						} else {
							await this.refreshInbox(epoch);
						}
					} catch (cause) {
						if (this.current(epoch)) this.failure(cause, 'messages.more');
					}
				} else this.failure(error, 'messages.more');
			}
		} finally {
			if (this.current(epoch)) this.update({ loadingMore: false });
		}
	}
	async search(query: string, cursor?: string | null): Promise<void> {
		const search = ++this.searchEpoch;
		const epoch = this.epoch;
		if (!query.trim()) {
			this.update({ recipients: [], recipientCursor: undefined });
			return;
		}
		try {
			const page = await this.api.search.profiles(query, { limit: 30, cursor });
			if (!this.current(epoch) || search !== this.searchEpoch) return;
			this.update({
				recipients: cursor
					? [...this.state.recipients, ...page.items]
					: page.items,
				recipientCursor: page.cursor,
				error: ''
			});
		} catch (error) {
			if (this.current(epoch) && search === this.searchEpoch)
				this.failure(error, 'messages.search');
		}
	}
	async start(
		recipient: string,
		sharedLink?: string
	): Promise<string | undefined> {
		if (this.state.busy) return;
		const link =
			sharedLink === undefined ? undefined : sharedCraftskyLink(sharedLink);
		if (sharedLink !== undefined && !link) {
			this.update({
				error: 'Open the original CraftSky content to share this link.'
			});
			return;
		}
		const epoch = this.epoch;
		this.update({ busy: true, error: '' });
		try {
			const availability = await this.api.chat.availability(recipient);
			if (!this.current(epoch)) return;
			if (!availability.canChat && !availability.conversationId) {
				this.update({
					error: 'This person cannot receive a new message from you right now.'
				});
				return;
			}
			const id =
				availability.conversationId ||
				(await this.api.chat.start(recipient)).id;
			if (!this.current(epoch)) return;
			if (link) stageSharedLink(this.account, id, link);
			return id;
		} catch (error) {
			if (this.current(epoch)) this.failure(error, 'messages.start');
		} finally {
			if (this.current(epoch)) this.update({ busy: false });
		}
	}
	edit(text: string): void {
		if (!this.state.id || this.state.confirming) return;
		this.update({ draft: { ...this.state.draft, text } });
	}
	private async setDraft(draft: MessageDraft): Promise<boolean> {
		const epoch = this.epoch,
			id = this.state.id,
			textBefore = this.state.draft.text;
		if (!id) return false;
		try {
			await this.store.save(id, draft);
			if (!this.current(epoch)) return false;
			if (this.state.draft.text !== textBefore)
				draft = { ...draft, text: this.state.draft.text };
			this.update({ draft });
			return true;
		} catch (error) {
			if (this.current(epoch)) this.failure(error, 'messages.draft.save');
			return false;
		}
	}
	async send(retry = false): Promise<void> {
		const { id, conversation } = this.state;
		if (
			!id ||
			this.state.busy ||
			this.state.peerBlocked ||
			this.state.confirming ||
			!this.state.status?.capabilities.canSend ||
			!conversation?.canSend
		)
			return;
		const epoch = this.epoch;
		const live = this.session.captureAuthority();
		let draft = this.state.draft;
		try {
			if (!retry) draft = freezeSend(draft);
			else if (!draft.pending || draft.pending.state === 'failed') return;
		} catch (error) {
			void reportFailure(error, 'messages.send.prepare');
			this.update({
				error: userErrorMessage(error)
			});
			return;
		}
		this.update({ busy: true });
		if (!(await this.setDraft(draft))) {
			if (!this.current(epoch)) return;
			this.update({ busy: false });
			return;
		}
		if (!this.current(epoch) || !live()) {
			if (this.current(epoch)) this.update({ busy: false });
			return;
		}
		const attempt = draft.pending!;
		this.update({ busy: true, error: '', notice: '' });
		try {
			const receipt = await this.api.chat.send(
				id,
				attempt.operationId,
				attempt.text
			);
			if (!this.current(epoch) || !live()) return;
			if (
				!(await this.setDraft(reconcileSend(this.state.draft, receipt))) ||
				!live()
			)
				return;
			if (receipt.state === 'confirmed' && receipt.message) {
				const latest = this.state.conversation ?? conversation;
				const updated = this.session.confirmedSend(latest, receipt.message);
				this.update({
					conversation: updated,
					messages: mergeMessages(this.state.messages, [receipt.message])
				});
			}
			if (receipt.state === 'confirmed')
				void this.reconcileConfirmedSend(epoch);
		} catch (error) {
			if (!this.current(epoch)) return;
			const pending = this.state.draft.pending;
			if (pending?.operationId === attempt.operationId)
				await this.setDraft({
					...this.state.draft,
					pending: { ...pending, state: 'uncertain' }
				});
			this.failure(error, 'messages.send');
		} finally {
			if (this.current(epoch)) this.update({ busy: false });
		}
	}
	async checkSend(): Promise<void> {
		const pending = this.state.draft.pending;
		if (!pending || this.state.busy) return;
		const epoch = this.epoch;
		const live = this.session.captureAuthority();
		this.update({ busy: true, error: '' });
		try {
			const receipt = await this.api.chat.receipt(pending.operationId);
			if (!this.current(epoch) || !live()) return;
			if (
				!(await this.setDraft(reconcileSend(this.state.draft, receipt))) ||
				!live()
			)
				return;
			if (receipt.state === 'confirmed')
				void this.reconcileConfirmedSend(epoch);
		} catch (error) {
			if (this.current(epoch)) {
				if (error instanceof APIError && error.status === 404)
					this.update({
						notice:
							'Delivery is not confirmed. Retry uses the same message attempt.'
					});
				else this.failure(error, 'messages.checkSend');
			}
		} finally {
			if (this.current(epoch)) this.update({ busy: false });
		}
	}
	async dismissFailed(): Promise<void> {
		if (this.state.draft.pending?.state === 'failed')
			await this.setDraft({
				text: this.state.draft.text || this.state.draft.pending.text
			});
	}
	private async reconcileConfirmedSend(epoch: number): Promise<void> {
		try {
			await this.refreshContent(epoch);
		} catch (error) {
			if (this.current(epoch)) this.failure(error, 'messages.sync');
		}
	}
	private publishDetails(details: ChatDetails) {
		const { items, cursor } = details.messages;
		const returnedIDs = new Set(items.map((item) => item.id));
		const first = mergeMessages([], items)[0];
		// Keep the older visible range only if the newest page reconnects to it.
		// The cache itself retains only the newest authoritative page.
		const joined =
			!!first &&
			this.state.messages.some((message) => returnedIDs.has(message.id));
		const older =
			joined && cursor
				? this.state.messages.filter(
						(message) =>
							Date.parse(message.sentAt) < Date.parse(first.sentAt) &&
							!returnedIDs.has(message.id)
					)
				: [];
		const historyCursor = older.length ? this.state.historyCursor : cursor;
		if (this.state.view === 'conversation')
			this.session.rememberThread(details);
		this.update({
			conversation: details.conversation,
			messages: mergeMessages(older, items),
			historyCursor,
			confirming: false
		});
	}
	private async refreshContent(epoch: number): Promise<void> {
		const { id } = this.state;
		if (!id) return;
		const details = await this.session.loadDetails(id);
		if (this.current(epoch)) this.publishDetails(details);
	}
	async presented(messageID: string): Promise<void> {
		const index = this.state.messages.findIndex(
			(message) => message.id === messageID
		);
		const last = this.state.messages.findIndex(
			(message) => message.id === this.readID
		);
		const pending = this.state.messages.findIndex(
			(message) => message.id === this.pendingRead
		);
		const sending = this.state.messages.findIndex(
			(message) => message.id === this.readTarget
		);
		if (
			!this.state.id ||
			this.state.view !== 'conversation' ||
			this.state.confirming ||
			!statusAllowsReading(this.state.status) ||
			!this.state.conversation?.unreadCount ||
			index < 0 ||
			index <= last ||
			index <= pending ||
			index <= sending ||
			this.disposed
		)
			return;
		this.pendingRead = messageID;
		const epoch = this.epoch;
		if (this.readingEpoch === epoch) return;
		const live = this.session.captureAuthority();
		this.readingEpoch = epoch;
		const id = this.state.id;
		try {
			// Coalesce observers that enter the viewport in the same rendering task.
			await Promise.resolve();
			while (this.pendingRead && this.current(epoch) && live()) {
				const target = this.pendingRead;
				this.pendingRead = undefined;
				this.readTarget = target;
				const conversation = await this.api.chat.read(id, target);
				if (!this.current(epoch) || !live()) return;
				this.readID = target;
				// A read receipt only acknowledges the watermark. Send authority comes
				// from checked conversation/details reads, never this partial response.
				const current = this.state.conversation;
				if (current && conversation.id === current.id) {
					this.update({
						conversation: { ...current, unreadCount: conversation.unreadCount }
					});
					this.session.markRead(conversation, target);
					void this.refreshCounts(epoch, true);
					if (conversation.unreadCount === 0) this.pendingRead = undefined;
				}
			}
		} catch (error) {
			if (this.current(epoch)) this.failure(error, 'messages.presented');
		} finally {
			if (this.readingEpoch === epoch) {
				this.readTarget = undefined;
				this.readingEpoch = undefined;
			}
		}
	}
	async conversationAction(
		action: 'accept' | 'mute' | 'leave',
		afterLeave?: () => void
	): Promise<void> {
		const { id, conversation } = this.state;
		if (!id || !conversation || this.state.confirming) return;
		const live = this.session.captureAuthority();
		await this.action(
			async () => {
				if (action === 'leave') {
					await this.api.chat.leave(id);
					if (live()) this.session.removeThread(id);
					return undefined;
				}
				if (action === 'accept') {
					await this.api.chat.accept(id);
					if (live()) this.session.removeThread(id);
					// Accept acknowledges status. Obtain checked send authority separately.
					const accepted = await this.api.chat.conversation(id);
					if (accepted.id !== id || accepted.status !== 'accepted')
						throw new Error('Invalid accepted conversation response');
					if (live()) this.session.acceptConversation(accepted);
					return accepted;
				}
				const result = await this.api.chat.mute(id, !conversation.muted);
				if (result.id !== id) throw new Error('Invalid conversation response');
				const muted = { ...conversation, muted: result.muted };
				if (live()) this.session.changeConversation(muted);
				return muted;
			},
			(result) => {
				if (result) this.update({ conversation: result });
				else {
					this.update({ conversation: undefined, messages: [] });
					afterLeave?.();
				}
				void this.refreshCounts(this.epoch, true);
			}
		);
	}
	async deleteMessage(message: ChatMessage): Promise<void> {
		const id = this.state.id;
		if (!id || this.state.confirming) return;
		await this.action(
			() => this.api.chat.deleteMessage(id, message.id),
			() => {
				this.update({
					messages: this.state.messages.map((item) =>
						item.id === message.id
							? {
									...item,
									kind: 'deleted',
									text: undefined,
									senderDid: undefined
								}
							: item
					)
				});
			},
			() => this.session.deleteMessage(id, message.id)
		);
	}
	async blockPeer(): Promise<void> {
		const peer = this.state.conversation?.peer;
		if (
			!peer ||
			this.state.peerBlocked ||
			this.state.busy ||
			this.state.confirming
		)
			return;
		// Explicit retries retain the same write key after a lost response.
		if (this.blockAttempt?.did !== peer.did)
			this.blockAttempt = { did: peer.did, key: crypto.randomUUID() };
		const operation = this.blockAttempt;
		await this.action(
			() => this.api.profiles.block(operation.did, operation.key),
			() => {
				this.blockAttempt = undefined;
				this.update({
					peerBlocked: true,
					notice:
						'Account blocked. You can manage blocked accounts in Settings.'
				});
			}
		);
	}
	async setEnabled(enabled: boolean): Promise<void> {
		await this.action(
			() => this.api.chat.setConnection(enabled),
			(status) => {
				this.session.setStatus(status);
				this.update({
					status,
					push: status.pushEnabled,
					conversation: enabled ? this.state.conversation : undefined,
					messages: enabled ? this.state.messages : [],
					conversations: enabled ? this.state.conversations : []
				});
			}
		);
	}
	async setPush(enabled: boolean): Promise<void> {
		await this.action(
			() => this.api.chat.setPushPreference(enabled),
			(value) =>
				this.update({
					push: value.pushEnabled,
					notice: 'Message notification preference saved.'
				})
		);
	}
	async setPolicy(value: ChatIncomingSetting['allowIncoming']): Promise<void> {
		await this.action(
			() => this.api.chat.setIncoming(value),
			(policy) =>
				this.update({
					policy,
					notice: policy.persisted
						? 'Incoming message setting saved.'
						: 'The setting is still being confirmed. Refresh to check it.'
				})
		);
	}
	private async action<T>(
		work: () => Promise<T>,
		commit: (value: T) => void,
		commitShared?: (value: T) => void
	): Promise<void> {
		if (this.state.busy) return;
		const epoch = this.epoch;
		const live = this.session.captureAuthority();
		this.update({ busy: true, error: '', notice: '' });
		try {
			const result = await work();
			if (live()) {
				commitShared?.(result);
				if (this.current(epoch)) commit(result);
			}
		} catch (error) {
			if (this.current(epoch)) this.failure(error, 'messages.setPolicy');
		} finally {
			if (this.current(epoch)) this.update({ busy: false });
		}
	}
	async report(reasonType: string, reason: string): Promise<void> {
		if (this.state.busy || !this.state.id) return;
		const epoch = this.epoch;
		const invalid = validateText(reason, true);
		if (invalid || !reportReasons.some((item) => item[1] === reasonType)) {
			this.update({ error: invalid || 'Choose a report reason.' });
			return;
		}
		const existing = this.state.report;
		const operation = existing || {
			operationId: crypto.randomUUID(),
			reasonType,
			reason,
			state: 'pending' as const
		};
		if (!this.setReport(operation)) return;
		await this.action(
			() =>
				this.api.chat.report(
					operation.operationId,
					this.state.id!,
					operation.reasonType,
					operation.reason || undefined,
					this.state.reportMessage
				),
			(receipt) => this.reportReceipt(receipt, operation.operationId)
		);
		if (this.current(epoch) && this.state.report?.state === 'pending')
			this.setReport({ ...operation, state: 'uncertain' });
	}
	private setReport(report: FrozenReport | undefined): boolean {
		if (!this.state.id) return false;
		this.update({ report });
		try {
			this.store.saveReport(this.state.id, this.state.reportMessage, report);
			this.reportSaveFailed = false;
			return true;
		} catch (error) {
			if (!this.reportSaveFailed)
				void reportFailure(error, 'messages.report.save');
			this.reportSaveFailed = true;
			this.update({
				error:
					'This report could not be saved safely on your device. Try again before sending.'
			});
			return false;
		}
	}
	private reportReceipt(receipt: ChatReceipt, id: string): void {
		if (receipt.operationId !== id || receipt.kind !== 'report')
			throw new Error('Invalid reporting receipt');
		if (receipt.state === 'confirmed') {
			this.setReport(undefined);
			this.update({ notice: 'Report sent to Bluesky.' });
		} else if (this.state.report)
			this.setReport({ ...this.state.report, state: receipt.state });
	}
	async checkReport(): Promise<void> {
		const report = this.state.report;
		if (!report) return;
		await this.action(
			() => this.api.chat.receipt(report.operationId),
			(receipt) => this.reportReceipt(receipt, report.operationId)
		);
	}
	editFailedReport(): void {
		if (this.state.report?.state === 'failed') this.setReport(undefined);
	}
	async sync(visible: boolean): Promise<void> {
		if (this.disposed || !this.state.status?.enabled) return;
		if (!visible) {
			if (!this.viewing) return;
			this.viewing = false;
			try {
				await this.session.heartbeat('none');
			} catch (error) {
				if (!this.disposed && !isCanceled(error))
					this.failure(error, 'messages.sync');
			}
			return;
		}
		if (
			this.syncing ||
			this.state.loading ||
			this.state.confirming ||
			this.session.isBackingOff() ||
			!['conversation', 'inbox'].includes(this.state.view)
		)
			return;
		this.viewing = true;
		this.syncing = true;
		const epoch = this.epoch;
		try {
			const id =
				visible && this.state.view === 'conversation'
					? this.state.id
					: undefined;
			const inbox = visible && this.state.view === 'inbox';
			const snapshot = await this.session.heartbeat(
				id ? 'conversation' : inbox ? 'inbox' : 'none',
				id,
				this.syncRevision
			);
			if (!this.viewing || !this.current(epoch)) return;
			if (
				['authorizationNeeded', 'restricted', 'disabled'].includes(
					snapshot.state
				)
			) {
				const status: ChatStatus = {
					...this.state.status!,
					state: snapshot.state,
					enabled: snapshot.state !== 'disabled',
					capabilities: {
						canRead: false,
						canSend: false,
						canReport: false,
						canSetPolicy: false
					}
				};
				this.session.setStatus(status);
				this.update({
					status,
					conversation: undefined,
					messages: [],
					conversations: [],
					recipients: []
				});
				this.syncRevision = snapshot.revision;
				return;
			}
			const previousRevision = this.syncRevision;
			const changed = previousRevision !== snapshot.revision;
			const fallback =
				Date.now() - this.lastReconciled >= (id ? 30_000 : 60_000);
			if (changed || fallback) {
				if (id) await this.refreshContent(epoch);
				else if (inbox) {
					void this.refreshCounts(epoch, true);
					await this.refreshInbox(epoch);
				}
				if (this.current(epoch)) this.lastReconciled = Date.now();
			}
			if (this.current(epoch)) {
				this.syncRevision = snapshot.revision;
				this.session.rememberRevision(
					inbox ? this.state.bucket : undefined,
					id,
					snapshot.revision
				);
				this.syncFailed = false;
			}
		} catch (error) {
			if (this.current(epoch) && !isCanceled(error) && !this.syncFailed) {
				this.syncFailed = true;
				this.failure(error, 'messages.sync');
			}
		} finally {
			this.syncing = false;
		}
	}
}
