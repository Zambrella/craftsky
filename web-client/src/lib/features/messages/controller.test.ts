import { describe, expect, it, vi } from 'vitest';
import {
	CraftskyAPI,
	APIError,
	type ChatConversation,
	type ChatStatus
} from '#lib/api/client.ts';
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
