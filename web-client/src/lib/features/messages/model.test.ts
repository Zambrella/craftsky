import { describe, expect, it } from 'vitest';
import {
	MessageDraftStore,
	sharedCraftskyLink,
	prefillSharedLink,
	freezeSend,
	reconcileSend,
	validateText,
	mergeMessages,
	craftskyMessageLink,
	messageLinks,
	type StorageLike
} from './model';
const key = 'a2ec3903-9d32-4f55-b83c-b12c9230a480';
export class MemoryStorage implements StorageLike {
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
describe('private message drafts and delivery', () => {
	it('retains the frozen text and operation across a reload without silently resending', async () => {
		const storage = new MemoryStorage();
		const store = new MessageDraftStore(storage, 'account-a');
		const attempt = freezeSend({ text: 'Original message' }, () => key);
		await store.save('conversation', attempt);
		expect(
			new MessageDraftStore(storage, 'account-a').load('conversation')
		).toEqual({
			text: 'Original message',
			pending: {
				operationId: key,
				text: 'Original message',
				state: 'uncertain'
			}
		});
		expect(
			new MessageDraftStore(storage, 'account-b').load('conversation')
		).toEqual({ text: '' });
		expect(store.load('other-conversation')).toEqual({ text: '' });
	});
	it('keeps a newer draft when confirmation arrives for an earlier frozen message', () => {
		const draft = {
			...freezeSend({ text: 'Sent text' }, () => key),
			text: 'A newer draft'
		};
		expect(
			reconcileSend(draft, {
				operationId: key,
				kind: 'send',
				state: 'confirmed',
				messageId: 'message'
			})
		).toEqual({ text: 'A newer draft' });
		expect(() =>
			reconcileSend(draft, {
				operationId: 'different',
				kind: 'send',
				state: 'confirmed',
				messageId: 'message'
			})
		).toThrow();
		expect(() =>
			reconcileSend(draft, {
				operationId: key,
				kind: 'report',
				state: 'confirmed'
			})
		).toThrow();
		expect(() => freezeSend(draft)).toThrow();
	});
	it('distinguishes blank, Unicode grapheme and UTF8 limits without truncating text', () => {
		expect(validateText('  \n')).toBeTruthy();
		expect(validateText('\ud800')).toBeTruthy();
		expect(validateText('👩🏽‍🔧'.repeat(1000))).toBeTruthy(); // Within graphemes, exceeds byte limit.
		expect(validateText('🧶'.repeat(1000))).toBeUndefined();
		expect(validateText('🧶'.repeat(1001))).toBeTruthy();
		expect(validateText('', true)).toBeUndefined();
		expect(validateText('a'.repeat(2001), true)).toBeTruthy();
	});
	it('merges overlapping history by opaque identity and allows deletion updates', () => {
		const first = {
			id: 'a',
			revision: '1',
			sentAt: '2026-01-01T00:00:00Z',
			kind: 'text' as const,
			text: 'Private text'
		};
		const deleted = {
			...first,
			revision: '2',
			kind: 'deleted' as const,
			text: undefined
		};
		expect(
			mergeMessages(
				[first],
				[
					{
						id: 'b',
						revision: '1',
						sentAt: '2025-01-01T00:00:00Z',
						kind: 'unsupported'
					},
					deleted
				]
			)
		).toEqual([
			{
				id: 'b',
				revision: '1',
				sentAt: '2025-01-01T00:00:00Z',
				kind: 'unsupported'
			},
			deleted
		]);
	});
	it('binds report recovery to account, conversation and exact message', () => {
		const storage = new MemoryStorage(),
			store = new MessageDraftStore(storage, 'account-a');
		store.saveReport('conversation', 'message', {
			operationId: key,
			reasonType: 'com.atproto.moderation.defs#reasonSpam',
			reason: 'Explanation',
			state: 'pending'
		});
		expect(store.loadReport('conversation', 'message')?.state).toBe(
			'uncertain'
		);
		expect(store.loadReport('conversation')).toBeUndefined();
		expect(
			new MessageDraftStore(storage, 'account-b').loadReport(
				'conversation',
				'message'
			)
		).toBeUndefined();
	});
});
describe('message link handling', () => {
	it('leaves private URLs as text until activated and excludes credential URLs', () => {
		expect(messageLinks('See https://example.test/item).')).toEqual([
			{ text: 'See ' },
			{ text: 'https://example.test/item', href: 'https://example.test/item' },
			{ text: ').' }
		]);
		expect(messageLinks('https://user:secret@example.test')).toEqual([
			{ text: 'https://user:secret@example.test' }
		]);
	});
	it('only selects exact controlled CraftSky links for current-viewer previews', () => {
		expect(
			craftskyMessageLink(
				'https://app.craftsky.social/posts/did:plc:alice/post'
			)
		).toMatchObject({ kind: 'post', did: 'did:plc:alice', rkey: 'post' });
		for (const url of [
			'http://app.craftsky.social/profiles/did:plc:alice',
			'https://app.craftsky.social.evil.test/profiles/did:plc:alice',
			'https://app.craftsky.social/profiles/did:plc:alice?query=private',
			'https://app.craftsky.social/events/did:plc:alice/event#fragment'
		])
			expect(craftskyMessageLink(url)).toBeUndefined();
	});
});

describe('review-before-send CraftSky link bridge', () => {
	const link = 'https://app.craftsky.social/posts/did%3Aplc%3Aalice/post';
	it('only accepts an entire controlled public link, not embedded text or external credential URLs', () => {
		expect(sharedCraftskyLink(link)).toBe(link);
		expect(
			sharedCraftskyLink(
				'https://app.craftsky.social/profiles/did%3Aplc%3Aalice'
			)
		).toBeTruthy();
		expect(
			sharedCraftskyLink(
				'https://app.craftsky.social/events/did%3Aplc%3Aalice/event'
			)
		).toBeTruthy();
		for (const raw of [
			'hello ' + link,
			link + ' extra',
			link + '?secret=value',
			link + '#fragment',
			'https://app.craftsky.social.evil.test/profiles/did:plc:a',
			'https://user:secret@app.craftsky.social/posts/did:plc:a/post',
			link + '\n'
		])
			expect(sharedCraftskyLink(raw)).toBeUndefined();
	});
	it('preserves newer text, appends exactly once and leaves the frozen attempt unchanged', () => {
		const pending = {
			operationId: key,
			text: 'Frozen original',
			state: 'uncertain' as const
		};
		const draft = { text: 'Newer draft', pending };
		const next = prefillSharedLink(draft, link);
		expect(next).toEqual({ text: 'Newer draft\n\n' + link, pending });
		expect(next.pending).toBe(pending);
		expect(prefillSharedLink(next, link)).toBe(next);
		expect(prefillSharedLink({ text: pending.text, pending }, link)).toEqual({
			text: link,
			pending
		});
		expect(draft.text).toBe('Newer draft');
	});
});

it('rejects a stale tab save instead of overwriting frozen delivery', async () => {
	const storage = new MemoryStorage();
	const a = new MessageDraftStore(storage, 'account');
	const b = new MessageDraftStore(storage, 'account');
	a.load('chat');
	b.load('chat');
	await a.save(
		'chat',
		freezeSend({ text: 'Frozen' }, () => key)
	);
	await expect(b.save('chat', { text: 'Stale' })).rejects.toThrow(
		'another tab'
	);
	expect(a.load('chat').pending?.text).toBe('Frozen');
});
