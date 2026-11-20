import { reportFailure } from '../diagnostics/client';
vi.mock('../diagnostics/client', () => ({ reportFailure: vi.fn() }));
import { describe, expect, it, vi } from 'vitest';
import {
	CraftskyAPI,
	APIError,
	type ChatConversation,
	type ChatStatus
} from '#lib/api/client.ts';
import { MessagesController } from './controller';
import { MessagesSession } from './session';
class MemoryStorage {
	values = new Map<string, string>();
	getItem(key: string) {
		return this.values.get(key) ?? null;
	}
	setItem(key: string, value: string) {
		this.values.set(key, value);
	}
	removeItem(key: string) {
		this.values.delete(key);
	}
}
const ready: ChatStatus = {
	enabled: true,
	generation: 1,
	pushEnabled: false,
	state: 'ready',
	capabilities: {
		canRead: true,
		canSend: true,
		canReport: true,
		canSetPolicy: true
	}
};
const conversation = (id = 'conversation'): ChatConversation => ({
	id,
	revision: 'r1',
	status: 'accepted',
	muted: false,
	unreadCount: 2,
	canSend: true,
	peer: { did: 'did:plc:peer', handle: 'peer.example' }
});
const message = (id = 'message') => ({
	id,
	kind: 'text' as const,
	text: 'Hello',
	revision: '1',
	senderDid: 'did:plc:peer',
	sentAt: '2026-01-01T00:00:00Z'
});
function setup(storage = new MemoryStorage()) {
	const api = new CraftskyAPI(
		vi.fn<typeof fetch>(async () => {
			throw new Error('Unexpected request');
		})
	);
	vi.spyOn(api.chat, 'status').mockResolvedValue(ready);
	vi.spyOn(api.chat, 'details').mockImplementation(async (id) => ({
		conversation: conversation(id),
		messages: { items: [message()] }
	}));
	vi.spyOn(api.chat, 'conversation').mockImplementation(async (id) =>
		conversation(id)
	);
	vi.spyOn(api.chat, 'conversations').mockResolvedValue({
		items: [conversation()]
	});
	vi.spyOn(api.chat, 'unread').mockResolvedValue({ accepted: 2, requests: 1 });
	const publish = vi.fn();
	const session = new MessagesSession(api);
	const controller = new MessagesController(
		api,
		'account-a',
		storage,
		publish,
		session
	);
	return { api, controller, storage, publish, session };
}
describe('account-owned messaging operations', () => {
	it('retains the block write key on retry and disables sending after confirmation', async () => {
		const { api, controller } = setup();
		await controller.navigate(['conversation']);
		const block = vi
			.spyOn(api.profiles, 'block')
			.mockRejectedValueOnce(new APIError(0, 'network_error', 'Offline'))
			.mockResolvedValue({
				did: 'did:plc:peer',
				muted: false,
				blocking: true,
				blockedBy: false
			});
		await controller.blockPeer();
		expect(controller.state.peerBlocked).toBe(false);
		await controller.blockPeer();
		expect(block.mock.calls[1]).toEqual(block.mock.calls[0]);
		expect(controller.state.peerBlocked).toBe(true);
	});
	it('does not publish a late block result into a different conversation', async () => {
		const { api, controller } = setup();
		await controller.navigate(['conversation']);
		let finish!: (value: {
			did: string;
			muted: boolean;
			blocking: boolean;
			blockedBy: boolean;
		}) => void;
		vi.spyOn(api.profiles, 'block').mockImplementation(
			() => new Promise((resolve) => (finish = resolve))
		);
		const pending = controller.blockPeer();
		await controller.navigate(['another']);
		finish({
			did: 'did:plc:peer',
			muted: false,
			blocking: true,
			blockedBy: false
		});
		await pending;
		expect(controller.state.peerBlocked).toBe(false);
		expect(controller.state.busy).toBe(false);
	});
	it('persists the frozen operation before calling the server and fails closed if persistence fails', async () => {
		const { api, controller, storage } = setup();
		await controller.navigate(['conversation']);
		controller.edit('My message');
		vi.spyOn(storage, 'setItem').mockImplementation(() => {
			throw new Error('Storage full');
		});
		const send = vi.spyOn(api.chat, 'send');
		await controller.send();
		expect(send).not.toHaveBeenCalled();
		expect(controller.state.error).not.toBe('');
		expect(controller.state.draft.text).toBe('My message');
	});
	it('retries an ambiguous request with the same ID and frozen text while keeping newer edits', async () => {
		const { api, controller } = setup();
		await controller.navigate(['conversation']);
		controller.edit('First text');
		const send = vi
			.spyOn(api.chat, 'send')
			.mockRejectedValueOnce(new APIError(0, 'network_error', 'Offline'))
			.mockImplementation(async (_id, operationId) => ({
				operationId,
				kind: 'send',
				state: 'confirmed',
				messageId: 'sent-message',
				message: message('sent-message')
			}));
		await controller.send();
		const original = send.mock.calls[0];
		expect(controller.state.draft.pending?.state).toBe('uncertain');
		controller.edit('New draft');
		await controller.send(true);
		expect(send.mock.calls[1]).toEqual(original);
		expect(controller.state.draft).toEqual({ text: 'New draft' });
	});
	it('checks persisted ambiguous receipts on reload without creating a fresh send', async () => {
		const { api, controller, storage } = setup();
		await controller.navigate(['conversation']);
		controller.edit('Keep me');
		const send = vi
			.spyOn(api.chat, 'send')
			.mockRejectedValue(new APIError(0, 'network_error', 'Offline'));
		await controller.send();
		controller.dispose();
		const next = new MessagesController(api, 'account-a', storage, () => {});
		const receipt = vi
			.spyOn(api.chat, 'receipt')
			.mockRejectedValue(new APIError(404, 'not_found', 'Not found'));
		await next.navigate(['conversation']);
		await vi.waitFor(() => expect(receipt).toHaveBeenCalledOnce());
		expect(send).toHaveBeenCalledOnce();
		expect(next.state.draft.pending?.text).toBe('Keep me');
	});
	it('does not publish a late send response into a disposed account activation', async () => {
		const { api, controller, publish } = setup();
		await controller.navigate(['conversation']);
		controller.edit('Account A text');
		let finish!: (value: Awaited<ReturnType<typeof api.chat.send>>) => void;
		vi.spyOn(api.chat, 'send').mockImplementation(
			() => new Promise((resolve) => (finish = resolve))
		);
		const pending = controller.send();
		await vi.waitFor(() =>
			expect(controller.state.draft.pending).toBeDefined()
		);
		const operationId = controller.state.draft.pending!.operationId;
		controller.dispose();
		publish.mockClear();
		finish({
			operationId,
			kind: 'send',
			state: 'confirmed',
			messageId: 'sent'
		});
		await pending;
		expect(publish).not.toHaveBeenCalled();
	});
	it('rejects stale conversation loads after navigation', async () => {
		const { api, controller } = setup();
		let finish!: (value: Awaited<ReturnType<typeof api.chat.details>>) => void;
		vi.mocked(api.chat.details).mockImplementation((id) =>
			id === 'old'
				? new Promise((resolve) => (finish = resolve))
				: Promise.resolve({
						conversation: conversation(id),
						messages: { items: [message('new-message')] }
					})
		);
		const old = controller.navigate(['old']);
		await vi.waitFor(() => expect(finish).toBeDefined());
		await controller.navigate(['new-conversation']);
		finish({
			conversation: conversation('old'),
			messages: { items: [message('old-message')] }
		});
		await old;
		expect(controller.state.conversation?.id).toBe('new-conversation');
		expect(controller.state.messages[0].id).toBe('new-message');
	});
	it('marks only presented known messages read and never regresses the marker', async () => {
		const { api, controller } = setup();
		vi.mocked(api.chat.details).mockResolvedValue({
			conversation: conversation(),
			messages: {
				items: [
					message('first'),
					{ ...message('last'), sentAt: '2026-01-02T00:00:00Z' }
				]
			}
		});
		const read = vi.spyOn(api.chat, 'read').mockResolvedValue(conversation());
		await controller.navigate(['conversation']);
		expect(read).not.toHaveBeenCalled();
		await controller.presented('not-in-history');
		await controller.presented('last');
		await controller.presented('first');
		expect(read.mock.calls).toEqual([['conversation', 'last']]);
	});
	it('a read receipt updates unread metadata without replacing confirmed send permission', async () => {
		const { api, controller } = setup();
		vi.spyOn(api.chat, 'read').mockResolvedValue({
			...conversation(),
			unreadCount: 0,
			canSend: false
		});
		await controller.navigate(['conversation']);
		await controller.presented('message');
		expect(controller.state.conversation?.unreadCount).toBe(0);
		expect(controller.state.conversation?.canSend).toBe(true);
	});
	it('muting preserves send authority and accepting verifies it with checked metadata', async () => {
		const { api, controller } = setup();
		await controller.navigate(['conversation']);
		vi.spyOn(api.chat, 'mute').mockResolvedValue({
			...conversation(),
			muted: true,
			canSend: false
		});
		await controller.conversationAction('mute');
		expect(controller.state.conversation?.muted).toBe(true);
		expect(controller.state.conversation?.canSend).toBe(true);
		vi.spyOn(api.chat, 'accept').mockResolvedValue({
			...conversation(),
			canSend: false
		});
		vi.mocked(api.chat.details).mockClear();
		await controller.conversationAction('accept');
		expect(api.chat.conversation).toHaveBeenCalledWith('conversation');
		expect(api.chat.details).not.toHaveBeenCalled();
		expect(controller.state.conversation?.canSend).toBe(true);
	});
	it('syncs visibility without automatically resending or marking fetched history read', async () => {
		const { api, controller } = setup();
		const sync = vi
			.spyOn(api.chat, 'sync')
			.mockResolvedValue({ revision: 2, state: 'ready' });
		const read = vi.spyOn(api.chat, 'read'),
			send = vi.spyOn(api.chat, 'send');
		await controller.navigate(['conversation']);
		await controller.sync(false);
		expect(sync).not.toHaveBeenCalled();
		await controller.sync(true);
		await controller.sync(false);
		await controller.sync(false);
		expect(sync.mock.calls.map(([view]) => view)).toEqual([
			'conversation',
			'none'
		]);
		expect(read).not.toHaveBeenCalled();
		expect(send).not.toHaveBeenCalled();
	});
	it('retains uncertain reports with their exact payload and operation across reload', async () => {
		const { api, controller, storage } = setup();
		await controller.navigate(['conversation', 'report', 'message']);
		const report = vi
			.spyOn(api.chat, 'report')
			.mockRejectedValue(new APIError(0, 'network_error', 'Offline'));
		await controller.report(
			'com.atproto.moderation.defs#reasonSpam',
			'Frozen explanation'
		);
		expect(controller.state.report?.state).toBe('uncertain');
		const next = new MessagesController(api, 'account-a', storage, () => {});
		await next.navigate(['conversation', 'report', 'message']);
		await next.report(
			'com.atproto.moderation.defs#reasonOther',
			'Changed text'
		);
		expect(report.mock.calls[1]).toEqual(report.mock.calls[0]);
	});
	it('clears previously displayed private content when access is revoked', async () => {
		const { api, controller } = setup();
		await controller.navigate(['conversation']);
		vi.spyOn(api.chat, 'sync').mockRejectedValue(
			new APIError(403, 'chat_restricted', 'Restricted')
		);
		await controller.sync(true);
		expect(controller.state.messages).toEqual([]);
		expect(controller.state.conversation).toBeUndefined();
	});
});

describe('account-owned shared-link recipient selection', () => {
	const link = 'https://app.craftsky.social/posts/did%3Aplc%3Aalice/post';
	it('prefills a shared link once without persisting or sending it', async () => {
		const { api, controller, storage } = setup();
		vi.spyOn(api.chat, 'availability').mockResolvedValue({
			canChat: true,
			conversationId: 'existing'
		});
		const send = vi.spyOn(api.chat, 'send');
		const start = vi.spyOn(api.chat, 'start');
		expect(await controller.start('did:plc:peer', link)).toBe('existing');
		await controller.navigate(['existing']);
		expect(controller.state.draft.text).toBe(link);
		expect(storage.values.size).toBe(0);
		expect(send).not.toHaveBeenCalled();
		expect(start).not.toHaveBeenCalled();
		await controller.navigate(['existing'], true);
		expect(controller.state.draft.text).toBe('');
	});
	it('discards a recipient result after the active controller is disposed', async () => {
		const { api, controller, storage } = setup();
		let finish!: (value: { canChat: boolean; conversationId: string }) => void;
		vi.spyOn(api.chat, 'availability').mockImplementation(
			() =>
				new Promise((resolve) => {
					finish = resolve;
				})
		);
		const pending = controller.start('did:plc:peer', link);
		controller.dispose();
		finish({ canChat: true, conversationId: 'existing' });
		expect(await pending).toBeUndefined();
		expect(storage.values.size).toBe(0);
	});
	it('rejects an unrecognized link before contacting messaging services', async () => {
		const { api, controller } = setup();
		const read = vi.spyOn(api.chat, 'availability');
		expect(
			await controller.start('did:plc:peer', 'https://evil.test')
		).toBeUndefined();
		expect(read).not.toHaveBeenCalled();
	});
});

it('reports a consumed message failure once with its original cause and action', async () => {
	vi.mocked(reportFailure).mockClear();
	const { api, controller } = setup();
	const cause = new APIError(503, 'internal_error', 'PRIVATE_CANARY');
	vi.spyOn(api.chat, 'status').mockRejectedValueOnce(cause);
	await controller.navigate(['conversation']);
	expect(reportFailure).toHaveBeenCalledExactlyOnceWith(
		cause,
		'messages.navigate'
	);
	expect(controller.state.error).toBe(
		'CraftSky is having trouble right now. Please try again.'
	);
});

it('does not repeatedly report the same failed polling cycle', async () => {
	const { api, controller } = setup();
	await controller.navigate(['conversation']);
	vi.mocked(reportFailure).mockClear();
	const cause = new Error('PRIVATE_CANARY');
	vi.spyOn(api.chat, 'sync').mockRejectedValue(cause);
	await controller.sync(true);
	await controller.sync(true);
	expect(reportFailure).toHaveBeenCalledExactlyOnceWith(cause, 'messages.sync');
	expect(controller.state.error).not.toContain('PRIVATE_CANARY');
});

it('inbox refresh reloads its loaded range atomically and retains it if a continuation fails', async () => {
	const { api, controller } = setup();
	let fail = false;
	let finish!: () => void;
	let refreshing = false;
	let delayed = Promise.resolve();
	vi.mocked(api.chat.conversations).mockImplementation(
		async (_bucket, options) => {
			if (refreshing && options?.cursor) {
				await delayed;
				if (fail) throw new APIError(503, 'unavailable', 'Offline');
			}
			return options?.cursor
				? { items: [conversation('older')] }
				: { items: [conversation('newest')], cursor: 'next' };
		}
	);
	await controller.navigate([]);
	await controller.more();
	expect(controller.state.conversations.map((item) => item.id)).toEqual([
		'newest',
		'older'
	]);
	refreshing = true;
	delayed = new Promise<void>((resolve) => {
		finish = resolve;
	});
	const request = controller.refresh();
	await Promise.resolve();
	expect(controller.state.conversations).toHaveLength(2);
	finish();
	await request;
	expect(controller.state.conversations).toHaveLength(2);
	fail = true;
	await controller.refresh();
	expect(controller.state.conversations).toHaveLength(2);
	controller.dispose();
});

it('refresh reloads restricted inbox status without requesting private conversations', async () => {
	const { api, controller } = setup();
	const restricted: ChatStatus = {
		...ready,
		state: 'restricted',
		capabilities: {
			canRead: false,
			canSend: false,
			canReport: false,
			canSetPolicy: false
		}
	};
	vi.mocked(api.chat.status)
		.mockResolvedValueOnce(restricted)
		.mockResolvedValueOnce({
			...restricted,
			enabled: false,
			state: 'disabled'
		});
	await controller.navigate([]);
	expect(controller.state.status?.state).toBe('restricted');
	await controller.refresh();
	expect(controller.state.status?.state).toBe('disabled');
	expect(api.chat.conversations).not.toHaveBeenCalled();
	controller.dispose();
});

it('shares the warm inbox request and displays rows while the badge is pending', async () => {
	const { api, controller, session } = setup();
	let showInbox!: (
		value: Awaited<ReturnType<typeof api.chat.conversations>>
	) => void;
	let showCounts!: (value: Awaited<ReturnType<typeof api.chat.unread>>) => void;
	vi.mocked(api.chat.conversations).mockImplementation(
		() => new Promise((resolve) => (showInbox = resolve))
	);
	vi.mocked(api.chat.unread).mockImplementation(
		() => new Promise((resolve) => (showCounts = resolve))
	);
	const preload = session.warmInbox();
	const visit = controller.navigate([]);
	await vi.waitFor(() => expect(showInbox).toBeDefined());
	expect(api.chat.status).toHaveBeenCalledTimes(1);
	expect(api.chat.conversations).toHaveBeenCalledTimes(1);
	showInbox({ items: [conversation()] });
	await Promise.all([preload, visit]);
	expect(controller.state.loading).toBe(false);
	expect(controller.state.conversations).toHaveLength(1);
	expect(controller.state.counts.accepted).toBe(0);
	showCounts({ accepted: 3, requests: 2 });
	await vi.waitFor(() => expect(controller.state.counts.requests).toBe(2));
});

it('reopens a cached thread and holds actions until fresh authorization is available', async () => {
	const { api, controller, session, storage, publish } = setup();
	await controller.navigate(['conversation']);
	controller.dispose();
	const reopened = new MessagesController(
		api,
		'account-a',
		storage,
		publish,
		session
	);
	let showDetails!: (
		value: Awaited<ReturnType<typeof api.chat.details>>
	) => void;
	vi.mocked(api.chat.details).mockImplementation(
		() => new Promise((resolve) => (showDetails = resolve))
	);
	const visit = reopened.navigate(['conversation']);
	expect(reopened.state.messages.map((item) => item.id)).toEqual(['message']);
	expect(reopened.state.loading).toBe(false);
	expect(reopened.state.confirming).toBe(true);
	const read = vi.spyOn(api.chat, 'read');
	await reopened.presented('message');
	expect(read).not.toHaveBeenCalled();
	await vi.waitFor(() => expect(showDetails).toBeDefined());
	showDetails({
		conversation: conversation(),
		messages: { items: [message('fresh')] }
	});
	await visit;
	expect(reopened.state.confirming).toBe(false);
	expect(reopened.state.messages.map((item) => item.id)).toEqual(['fresh']);
});

it('retains no more than five recent conversations and expires them by time', async () => {
	const { api } = setup();
	let time = 0;
	const session = new MessagesSession(api, () => time);
	for (let i = 1; i <= 6; i++) {
		const id = String(i);
		session.rememberThread({
			conversation: conversation(id),
			messages: { items: [message()] }
		});
	}
	expect(session.cachedThread('1')).toBeUndefined();
	expect(session.cachedThread('6')?.data.conversation.id).toBe('6');
	time = 120_001;
	expect(session.cachedThread('6')).toBeUndefined();
});

it('unlocks a confirmed send before a held history refresh', async () => {
	const { api, controller, session } = setup();
	await controller.navigate([]);
	await controller.navigate(['conversation']);
	const sent = { ...message('sent'), sentAt: '2026-02-03T12:00:00Z' };
	vi.spyOn(api.chat, 'send').mockImplementation(async (_, operationId) => ({
		operationId,
		message: sent,
		messageId: sent.id,
		kind: 'send',
		state: 'confirmed'
	}));
	let showDetails!: (
		value: Awaited<ReturnType<typeof api.chat.details>>
	) => void;
	vi.mocked(api.chat.details).mockImplementation(
		() => new Promise((resolve) => (showDetails = resolve))
	);
	controller.edit('A message');
	await controller.send();
	expect(showDetails).toBeDefined();
	expect(controller.state.busy).toBe(false);
	expect(controller.state.draft.pending).toBeUndefined();
	expect(controller.state.messages.at(-1)?.id).toBe('sent');
	expect(session.cachedInbox('accepted')?.data.items[0].lastMessage?.id).toBe(
		'sent'
	);
	showDetails({
		conversation: conversation(),
		messages: { items: [message()] }
	});
	await vi.waitFor(() =>
		expect(controller.state.messages.at(-1)?.id).toBe('sent')
	);
});

it('keeps a confirmed deletion across navigation and masks stale responses', async () => {
	const { api, controller, session, storage, publish } = setup();
	await controller.navigate(['conversation']);
	let confirm!: (
		value: Awaited<ReturnType<typeof api.chat.deleteMessage>>
	) => void;
	vi.spyOn(api.chat, 'deleteMessage').mockImplementation(
		() => new Promise((resolve) => (confirm = resolve))
	);
	const deleting = controller.deleteMessage(message());
	controller.dispose();
	confirm({ deleted: true });
	await deleting;
	const reopened = new MessagesController(
		api,
		'account-a',
		storage,
		publish,
		session
	);
	await reopened.navigate(['conversation']);
	expect(reopened.state.messages[0]).toMatchObject({
		id: 'message',
		kind: 'deleted',
		text: undefined
	});
});

it('does not replay a mutation invalidation over unrelated pending status', async () => {
	const { api, session } = setup();
	await session.loadStatus();
	const newest = { ...message('newer'), sentAt: '2026-06-01T00:00:00Z' };
	vi.mocked(api.chat.conversations).mockResolvedValue({
		items: [conversation('first'), { ...conversation(), lastMessage: newest }]
	});
	await session.loadInbox('accepted');
	let status!: (value: ChatStatus) => void;
	vi.mocked(api.chat.status).mockImplementation(
		() => new Promise((resolve) => (status = resolve))
	);
	const pending = session.loadStatus(true);
	session.confirmedSend(conversation(), message('earlier'));
	status(ready);
	await expect(pending).resolves.toEqual(ready);
	expect(
		session.cachedInbox('accepted')?.data.items.map((item) => item.id)
	).toEqual(['first', 'conversation']);
	expect(session.inboxConversation('conversation')?.lastMessage?.id).toBe(
		'newer'
	);
});

it('keeps a confirmed block local when the shell invalidates relationship-dependent reads', async () => {
	const { api, controller, session } = setup();
	await controller.navigate(['conversation']);
	vi.spyOn(api.profiles, 'block').mockImplementation(async () => {
		session.invalidateRelationships();
		return {
			did: 'did:plc:peer',
			muted: false,
			blocking: true,
			blockedBy: false
		};
	});
	await controller.blockPeer();
	expect(controller.state.peerBlocked).toBe(true);
	expect(controller.state.notice).toContain('Account blocked');
	expect(controller.state.messages).toHaveLength(1);
	expect(session.cachedThread('conversation')).toBeUndefined();
});

it('lets a warm thread retry after a transient authorization check failure', async () => {
	const { api, controller, session, storage, publish } = setup();
	await controller.navigate(['conversation']);
	controller.dispose();
	const reopened = new MessagesController(
		api,
		'account-a',
		storage,
		publish,
		session
	);
	vi.mocked(api.chat.details).mockRejectedValueOnce(
		new APIError(503, 'unavailable', 'Offline')
	);
	await reopened.navigate(['conversation']);
	expect(reopened.state.messages).toHaveLength(1);
	expect(reopened.state.confirming).toBe(true);
	reopened.edit('Do not lose an enabled composer edit');
	expect(reopened.state.draft.text).toBe('');
	await reopened.refresh();
	expect(reopened.state.confirming).toBe(false);
	expect(reopened.state.error).toBe('');
});

it('does not append a confirmed send when it falls before a disjoint latest page', async () => {
	const { api, controller, session } = setup();
	await controller.navigate(['conversation']);
	const sent = { ...message('sent'), sentAt: '2026-01-02T00:00:00Z' };
	session.confirmedSend(conversation(), sent);
	vi.mocked(api.chat.details).mockResolvedValue({
		conversation: conversation(),
		messages: {
			items: [{ ...message('new-window'), sentAt: '2026-01-10T00:00:00Z' }],
			cursor: 'older'
		}
	});
	await controller.refresh();
	expect(controller.state.messages.map((item) => item.id)).toEqual([
		'new-window'
	]);
	expect(controller.state.historyCursor).toBe('older');
});

it('shows a retryable failure instead of a false empty inbox during shared rate-limit backoff', async () => {
	const { api, publish, storage } = setup();
	let time = 0;
	const session = new MessagesSession(api, () => time);
	session.observeRateLimit(
		new APIError(429, 'rate_limited', 'Wait', undefined, undefined, 30)
	);
	const controller = new MessagesController(
		api,
		'account-a',
		storage,
		publish,
		session
	);
	await controller.navigate([]);
	expect(controller.state.error).toContain('Too many requests');
	expect(api.chat.status).not.toHaveBeenCalled();
	time = 29_999;
	await controller.refresh();
	expect(api.chat.status).not.toHaveBeenCalled();
	time = 30_000;
	await controller.refresh();
	expect(controller.state.error).toBe('');
	expect(controller.state.conversations).toHaveLength(1);
});

it('keeps inbox rows and scroll through a long visit and revalidates stale data in place', async () => {
	const { api, publish, storage } = setup();
	let time = 0;
	const session = new MessagesSession(api, () => time);
	await session.warmInbox();
	session.rememberInboxScroll('accepted', 250);
	time = 120_001;
	let showInbox!: (
		value: Awaited<ReturnType<typeof api.chat.conversations>>
	) => void;
	vi.mocked(api.chat.conversations).mockImplementation(
		() => new Promise((resolve) => (showInbox = resolve))
	);
	const controller = new MessagesController(
		api,
		'account-a',
		storage,
		publish,
		session
	);
	const visit = controller.navigate([]);
	expect(controller.state.loading).toBe(false);
	expect(controller.state.conversations).toHaveLength(1);
	expect(controller.state.inboxScroll).toBe(250);
	await vi.waitFor(() => expect(showInbox).toBeDefined());
	showInbox({ items: [conversation('newest')] });
	await visit;
	expect(controller.state.conversations[0].id).toBe('newest');
	expect(session.cachedInbox('accepted')?.scroll).toBe(250);
});
