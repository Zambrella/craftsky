import {
	APIError,
	type CraftskyAPI,
	type Author,
	type ChatConversation,
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
	type MessageDraft,
	type FrozenReport,
	type StorageLike
} from './model';

export type MessageView =
	'inbox' | 'new' | 'settings' | 'conversation' | 'details' | 'report';
export interface MessagesState {
	view: MessageView;
	bucket: 'accepted' | 'request';
	id?: string;
	reportMessage?: string;
	loading: boolean;
	busy: boolean;
	loadingMore: boolean;
	error: string;
	notice: string;
	status?: ChatStatus;
	counts: { accepted: number; requests: number };
	conversations: ChatConversation[];
	cursor?: string | null;
	conversation?: ChatConversation;
	messages: ChatMessage[];
	historyCursor?: string | null;
	draft: MessageDraft;
	draftSaved: boolean;
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
	busy: false,
	loadingMore: false,
	error: '',
	notice: '',
	counts: { accepted: 0, requests: 0 },
	conversations: [],
	messages: [],
	draft: { text: '' },
	draftSaved: true,
	recipients: []
});
const accessDenied = (error: unknown): boolean =>
	error instanceof APIError &&
	(error.status === 401 ||
		error.status === 403 ||
		[
			'chat_conversation_unavailable',
			'chat_authorization_required',
			'chat_restricted',
			'chat_disconnected',
			'chat_connection_changed'
		].includes(error.code));
const errorMessage = (error: unknown): string =>
	error instanceof APIError && error.code === 'chat_relationship_unavailable'
		? 'This person cannot receive a new message from you right now.'
		: 'Messaging is temporarily unavailable. Try again.';

/** One immutable account activation owns all async results, drafts and polling. */
export class MessagesController {
	state = initial();
	private epoch = 0;
	private disposed = false;
	private store: MessageDraftStore;
	private route = '';
	private syncing = false;
	private syncRevision?: number;
	private searchEpoch = 0;
	private readID?: string;
	private pendingRead?: string;
	private readingEpoch?: number;
	constructor(
		readonly api: CraftskyAPI,
		readonly account: string,
		storage: StorageLike,
		private publish: (state: MessagesState) => void
	) {
		this.store = new MessageDraftStore(storage, account);
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
	private failure(error: unknown): void {
		if (accessDenied(error))
			this.update({
				conversation: undefined,
				messages: [],
				conversations: [],
				recipients: [],
				error:
					'Messaging access changed. Refresh or review your messaging permissions.'
			});
		else this.update({ error: errorMessage(error) });
	}
	dispose(): void {
		this.disposed = true;
		this.epoch++;
		this.searchEpoch++;
	}
	async navigate(path: string[], force = false): Promise<void> {
		const parts = path[0] === 'messages' ? path.slice(1) : path;
		const route = parts.join('/');
		if (!force && route === this.route && !this.state.loading) return;
		this.route = route;
		const epoch = ++this.epoch;
		this.searchEpoch++;
		this.syncRevision = undefined;
		this.readID = undefined;
		this.pendingRead = undefined;
		const first = parts[0];
		const reserved: MessageView[] = ['new', 'settings'];
		const view: MessageView =
			!first || first === 'requests'
				? 'inbox'
				: reserved.includes(first as MessageView)
					? (first as MessageView)
					: parts[1] === 'details'
						? 'details'
						: parts[1] === 'report'
							? 'report'
							: 'conversation';
		const id = ['conversation', 'details', 'report'].includes(view)
			? first
			: undefined;
		this.update({
			...initial(),
			view,
			bucket: first === 'requests' ? 'request' : 'accepted',
			id,
			reportMessage: view === 'report' ? parts[2] : undefined
		});
		try {
			const status = await this.api.chat.status();
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
			} else if (status.capabilities.canRead) {
				if (id) {
					const details = await this.api.chat.details(id);
					if (!this.current(epoch)) return;
					if (details.conversation.id !== id)
						throw new Error('Invalid conversation response');
					let draft: MessageDraft = { text: '' };
					let draftSaved = true;
					try {
						draft = this.store.load(id);
					} catch {
						draftSaved = false;
					}
					this.update({
						conversation: details.conversation,
						messages: mergeMessages([], details.messages.items),
						historyCursor: details.messages.cursor,
						draft,
						draftSaved
					});
					if (view === 'report')
						this.update({ report: this.store.loadReport(id, parts[2]) });
					if (draft.pending?.state === 'uncertain') void this.checkSend();
				} else if (view === 'inbox') {
					const [page, counts] = await Promise.all([
						this.api.chat.conversations(this.state.bucket),
						this.api.chat.unread()
					]);
					if (!this.current(epoch)) return;
					this.update({
						conversations: page.items,
						cursor: page.cursor,
						counts
					});
				} else if (view === 'new') {
					const recent = await this.api.chat.conversations('accepted', {
						limit: 20
					});
					if (!this.current(epoch)) return;
					this.update({ conversations: recent.items });
				}
			}
		} catch (error) {
			if (this.current(epoch)) this.failure(error);
		} finally {
			if (this.current(epoch)) this.update({ loading: false });
		}
	}
	refresh(): Promise<void> {
		return this.navigate(this.route.split('/').filter(Boolean), true);
	}
	async more(): Promise<void> {
		if (this.state.loadingMore) return;
		const epoch = this.epoch;
		const { id, cursor, historyCursor } = this.state;
		if (!(id ? historyCursor : cursor)) return;
		this.update({ loadingMore: true });
		try {
			if (id) {
				const page = await this.api.chat.messages(id, {
					cursor: historyCursor
				});
				if (this.current(epoch))
					this.update({
						messages: mergeMessages(this.state.messages, page.items),
						historyCursor: page.cursor
					});
			} else {
				const page = await this.api.chat.conversations(this.state.bucket, {
					cursor
				});
				if (this.current(epoch)) {
					const items = new Map(
						this.state.conversations.map((item) => [item.id, item])
					);
					for (const item of page.items) items.set(item.id, item);
					this.update({
						conversations: [...items.values()],
						cursor: page.cursor
					});
				}
			}
		} catch (error) {
			if (this.current(epoch)) this.failure(error);
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
				this.failure(error);
		}
	}
	async start(recipient: string): Promise<string | undefined> {
		if (this.state.busy) return;
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
			if (this.current(epoch)) return id;
		} catch (error) {
			if (this.current(epoch)) this.failure(error);
		} finally {
			if (this.current(epoch)) this.update({ busy: false });
		}
	}
	edit(text: string): void {
		if (!this.state.id) return;
		this.update({ draft: { ...this.state.draft, text } });
		this.saveDraft();
	}
	saveDraft(): boolean {
		if (!this.state.id) return false;
		try {
			this.store.save(this.state.id, this.state.draft);
			this.update({ draftSaved: true });
			return true;
		} catch {
			this.update({ draftSaved: false });
			return false;
		}
	}
	private setDraft(draft: MessageDraft): boolean {
		this.update({ draft });
		return this.saveDraft();
	}
	async send(retry = false): Promise<void> {
		const { id, conversation } = this.state;
		if (!id || this.state.busy || !conversation?.canSend) return;
		const epoch = this.epoch;
		let draft = this.state.draft;
		try {
			if (!retry) draft = freezeSend(draft);
			else if (!draft.pending || draft.pending.state === 'failed') return;
		} catch (error) {
			this.update({
				error:
					error instanceof Error
						? error.message
						: 'The message could not be sent.'
			});
			return;
		}
		if (!this.setDraft(draft)) {
			this.update({
				error:
					'Save this draft before sending so its delivery can be checked safely.'
			});
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
			if (!this.current(epoch)) return;
			this.setDraft(reconcileSend(this.state.draft, receipt));
			if (receipt.message)
				this.update({
					messages: mergeMessages(this.state.messages, [receipt.message])
				});
			if (receipt.state === 'confirmed') await this.refreshContent(epoch);
		} catch (error) {
			if (!this.current(epoch)) return;
			const pending = this.state.draft.pending;
			if (pending?.operationId === attempt.operationId)
				this.setDraft({
					...this.state.draft,
					pending: { ...pending, state: 'uncertain' }
				});
			this.failure(error);
		} finally {
			if (this.current(epoch)) this.update({ busy: false });
		}
	}
	async checkSend(): Promise<void> {
		const pending = this.state.draft.pending;
		if (!pending || this.state.busy) return;
		const epoch = this.epoch;
		this.update({ busy: true, error: '' });
		try {
			const receipt = await this.api.chat.receipt(pending.operationId);
			if (!this.current(epoch)) return;
			this.setDraft(reconcileSend(this.state.draft, receipt));
			if (receipt.state === 'confirmed') await this.refreshContent(epoch);
		} catch (error) {
			if (this.current(epoch)) {
				if (error instanceof APIError && error.status === 404)
					this.update({
						notice:
							'Delivery is not confirmed. Retry uses the same message attempt.'
					});
				else this.failure(error);
			}
		} finally {
			if (this.current(epoch)) this.update({ busy: false });
		}
	}
	dismissFailed(): void {
		if (this.state.draft.pending?.state === 'failed')
			this.setDraft({
				text: this.state.draft.text || this.state.draft.pending.text
			});
	}
	discardDraft(): void {
		if (
			this.state.draft.pending?.state === 'uncertain' ||
			this.state.draft.pending?.state === 'pending'
		)
			return;
		this.setDraft({ text: '' });
	}
	private async refreshContent(epoch: number): Promise<void> {
		const { id } = this.state;
		if (!id) return;
		const details = await this.api.chat.details(id);
		if (!this.current(epoch)) return;
		this.update({
			conversation: details.conversation,
			messages: mergeMessages(this.state.messages, details.messages.items)
		});
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
		if (
			!this.state.id ||
			index < 0 ||
			index <= last ||
			index <= pending ||
			this.disposed
		)
			return;
		this.pendingRead = messageID;
		const epoch = this.epoch;
		if (this.readingEpoch === epoch) return;
		this.readingEpoch = epoch;
		const id = this.state.id;
		try {
			while (this.pendingRead && this.current(epoch)) {
				const target = this.pendingRead;
				this.pendingRead = undefined;
				const conversation = await this.api.chat.read(id, target);
				if (!this.current(epoch)) return;
				this.readID = target;
				this.update({ conversation });
			}
		} catch (error) {
			if (this.current(epoch)) this.failure(error);
		} finally {
			if (this.readingEpoch === epoch) this.readingEpoch = undefined;
		}
	}
	async conversationAction(
		action: 'accept' | 'mute' | 'leave',
		afterLeave?: () => void
	): Promise<void> {
		const { id, conversation } = this.state;
		if (!id || !conversation) return;
		await this.action(
			async () => {
				if (action === 'leave') {
					await this.api.chat.leave(id);
					return undefined;
				}
				return action === 'accept'
					? this.api.chat.accept(id)
					: this.api.chat.mute(id, !conversation.muted);
			},
			(result) => {
				if (result) this.update({ conversation: result });
				else {
					this.update({ conversation: undefined, messages: [] });
					afterLeave?.();
				}
			}
		);
	}
	async deleteMessage(message: ChatMessage): Promise<void> {
		const id = this.state.id;
		if (!id) return;
		await this.action(
			() => this.api.chat.deleteMessage(id, message.id),
			() =>
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
				})
		);
	}
	async setEnabled(enabled: boolean): Promise<void> {
		await this.action(
			() => this.api.chat.setConnection(enabled),
			(status) =>
				this.update({
					status,
					push: status.pushEnabled,
					conversation: enabled ? this.state.conversation : undefined,
					messages: enabled ? this.state.messages : [],
					conversations: enabled ? this.state.conversations : []
				})
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
		commit: (value: T) => void
	): Promise<void> {
		if (this.state.busy) return;
		const epoch = this.epoch;
		this.update({ busy: true, error: '', notice: '' });
		try {
			const result = await work();
			if (this.current(epoch)) commit(result);
		} catch (error) {
			if (this.current(epoch)) this.failure(error);
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
			return true;
		} catch {
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
		if (this.syncing || this.disposed || !this.state.status?.enabled) return;
		this.syncing = true;
		const epoch = this.epoch;
		try {
			const id = visible ? this.state.id : undefined;
			const snapshot = await this.api.chat.sync(
				visible ? (id ? 'conversation' : 'inbox') : 'none',
				id,
				this.syncRevision
			);
			if (!this.current(epoch)) return;
			if (
				['authorizationNeeded', 'restricted', 'disabled'].includes(
					snapshot.state
				)
			) {
				const status = this.state.status!;
				this.update({
					status: {
						...status,
						state: snapshot.state,
						enabled: snapshot.state !== 'disabled',
						capabilities: {
							canRead: false,
							canSend: false,
							canReport: false,
							canSetPolicy: false
						}
					},
					conversation: undefined,
					messages: [],
					conversations: [],
					recipients: []
				});
				this.syncRevision = snapshot.revision;
				return;
			}
			if (
				visible &&
				(this.syncRevision === undefined ||
					this.syncRevision !== snapshot.revision)
			) {
				if (id) await this.refreshContent(epoch);
				else if (this.state.view === 'inbox') {
					const [page, counts] = await Promise.all([
						this.api.chat.conversations(this.state.bucket),
						this.api.chat.unread()
					]);
					if (this.current(epoch))
						this.update({
							conversations: page.items,
							cursor: page.cursor,
							counts
						});
				}
			}
			if (this.current(epoch)) this.syncRevision = snapshot.revision;
		} catch (error) {
			if (this.current(epoch)) this.failure(error);
		} finally {
			this.syncing = false;
		}
	}
}
