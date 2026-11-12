import { reportFailure } from '../diagnostics/client';
vi.mock('../diagnostics/client', () => ({ reportFailure: vi.fn() }));
import { describe, expect, it, vi } from 'vitest';
import {
	CraftskyAPI,
	APIError,
	type ChatConversation,
	type ChatStatus
} from '#lib/api/client.ts';
import { MessageDraftStore } from './model';
import { MessagesController } from './controller';
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
	vi.spyOn(api.chat, 'conversations').mockResolvedValue({
		items: [conversation()]
	});
	vi.spyOn(api.chat, 'unread').mockResolvedValue({ accepted: 2, requests: 1 });
	const publish = vi.fn();
	const controller = new MessagesController(api, 'account-a', storage, publish);
	return { api, controller, storage, publish };
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
		expect(controller.state.draftSaved).toBe(false);
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
	it('muting preserves send authority and accepting verifies it with a checked detail read', async () => {
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
		expect(api.chat.details).toHaveBeenCalledWith('conversation');
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
		await controller.sync(true);
		expect(sync.mock.calls[0][0]).toBe('none');
		expect(sync.mock.calls[1][0]).toBe('conversation');
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
	it('keeps a shared link in memory until an explicit save and never sends', async () => {
		const { api, controller, storage } = setup();
		const store = new MessageDraftStore(storage, 'account-a');
		await store.save('existing', { text: 'My draft' });
		vi.spyOn(api.chat, 'availability').mockResolvedValue({
			canChat: true,
			conversationId: 'existing'
		});
		const send = vi.spyOn(api.chat, 'send'),
			start = vi.spyOn(api.chat, 'start');
		expect(await controller.start('did:plc:peer', link)).toBe('existing');
		expect(store.load('existing').text).toBe('My draft');
		expect(
			new MessageDraftStore(storage, 'account-b').load('existing')
		).toEqual({ text: '' });
		expect(send).not.toHaveBeenCalled();
		expect(start).not.toHaveBeenCalled();
	});
	it('does not require persistence to choose a recipient and retains the saved draft', async () => {
		const { api, controller, storage } = setup();
		const store = new MessageDraftStore(storage, 'account-a');
		await store.save('existing', { text: 'Keep' });
		vi.spyOn(api.chat, 'availability').mockResolvedValue({
			canChat: true,
			conversationId: 'existing'
		});
		vi.spyOn(storage, 'setItem').mockImplementation(() => {
			throw new Error('quota');
		});
		expect(await controller.start('did:plc:peer', link)).toBe('existing');
		expect(store.load('existing').text).toBe('Keep');
		expect(controller.state.error).toBe('');
		expect(controller.state.busy).toBe(false);
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
