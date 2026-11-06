import 'fake-indexeddb/auto';
import { describe, expect, it, vi } from 'vitest';
import { assertOwner, drafts, newDraft } from './store.ts';

describe('private draft storage', () => {
	it('atomically clears only the deleted account, retaining other owners and rolling back an aborted cleanup', async () => {
		const owner = `did:plc:${crypto.randomUUID()}`,
			other = `did:plc:${crypto.randomUUID()}`;
		const first = newDraft(owner),
			second = newDraft(owner),
			third = newDraft(other);
		first.text = 'First';
		second.text = 'Second';
		third.text = 'Other';
		await Promise.all([
			drafts.save(owner, first),
			drafts.save(owner, second),
			drafts.save(other, third)
		]);
		const remove = IDBObjectStore.prototype.delete;
		const spy = vi
			.spyOn(IDBObjectStore.prototype, 'delete')
			.mockImplementationOnce(function (
				this: IDBObjectStore,
				key: IDBValidKey | IDBKeyRange
			) {
				const request = remove.call(this, key);
				request.addEventListener('success', () => this.transaction.abort());
				return request;
			});
		try {
			await expect(drafts.clear(owner)).rejects.toThrow('browser storage');
			expect(await drafts.list(owner)).toHaveLength(2);
		} finally {
			spy.mockRestore();
		}
		await drafts.clear(owner);
		expect(await drafts.list(owner)).toEqual([]);
		expect((await drafts.list(other)).map((item) => item.text)).toEqual([
			'Other'
		]);
	});
	it('round-trips actual image/video bytes and a frozen pending mutation', async () => {
		const owner = `did:plc:${crypto.randomUUID()}`;
		const draft = newDraft(owner);
		draft.text = 'A private work in progress';
		draft.images = [
			{
				id: 'image',
				name: 'quilt.jpg',
				blob: new Blob([new Uint8Array([1, 2, 3])], { type: 'image/jpeg' }),
				mime: 'image/jpeg',
				alt: 'Blue quilt'
			}
		];
		draft.video = {
			id: 'video',
			name: 'loom.mp4',
			blob: new Blob([new Uint8Array([4, 5, 6])], { type: 'video/mp4' }),
			mime: 'video/mp4',
			alt: 'Weaving'
		};
		draft.pending = {
			key: crypto.randomUUID(),
			body: { text: draft.text, sponsored: false }
		};
		await drafts.save(owner, draft);
		draft.text = 'Unsaved changes';
		const recovered = await drafts.get(owner, draft.id);
		expect(recovered?.text).toBe('A private work in progress');
		expect(recovered?.pending).toEqual(draft.pending);
		expect(await recovered?.images[0].blob?.arrayBuffer()).toEqual(
			new Uint8Array([1, 2, 3]).buffer
		);
		expect(await recovered?.video?.blob?.arrayBuffer()).toEqual(
			new Uint8Array([4, 5, 6]).buffer
		);
	});
	it('partitions reads, lists and deletion by owner even with identical draft IDs', async () => {
		const owner = `did:plc:${crypto.randomUUID()}`,
			other = `did:plc:${crypto.randomUUID()}`;
		const first = newDraft(owner),
			second = newDraft(other);
		second.id = first.id;
		first.text = 'First owner';
		second.text = 'Second owner';
		await Promise.all([drafts.save(owner, first), drafts.save(other, second)]);
		expect((await drafts.list(owner)).map((item) => item.text)).toEqual([
			'First owner'
		]);
		await drafts.remove(
			owner,
			first.id,
			(await drafts.get(owner, first.id))!.revision!
		);
		expect(await drafts.get(owner, first.id)).toBeUndefined();
		expect((await drafts.get(other, first.id))?.text).toBe('Second owner');
		await expect(drafts.save(other, first)).rejects.toThrow('another account');
		expect(() => assertOwner(other, first)).toThrow('another account');
	});
	it('does not claim a successful save when its IndexedDB transaction later aborts', async () => {
		const put = IDBObjectStore.prototype.put;
		const spy = vi
			.spyOn(IDBObjectStore.prototype, 'put')
			.mockImplementationOnce(function (
				this: IDBObjectStore,
				value: unknown,
				key?: IDBValidKey
			) {
				const request = put.call(this, value, key);
				request.addEventListener('success', () => this.transaction.abort());
				return request;
			});
		const owner = `did:plc:${crypto.randomUUID()}`;
		const draft = newDraft(owner);
		try {
			await expect(drafts.save(owner, draft)).rejects.toThrow(
				'browser storage'
			);
			expect(await drafts.get(owner, draft.id)).toBeUndefined();
		} finally {
			spy.mockRestore();
		}
	});
});

it('restores a private draft scheduling choice and exact instant without leaking it into another owner', async () => {
	const owner = `did:plc:${crypto.randomUUID()}`,
		other = `did:plc:${crypto.randomUUID()}`,
		draft = newDraft(owner);
	draft.scheduleLater = true;
	draft.scheduledAt = '2027-05-04T15:32:00.000Z';
	draft.langs = ['en', 'fr'];
	draft.sponsored = true;
	await drafts.save(owner, draft);
	expect(await drafts.get(owner, draft.id)).toMatchObject({
		scheduleLater: true,
		scheduledAt: draft.scheduledAt,
		langs: ['en', 'fr'],
		sponsored: true
	});
	expect(await drafts.get(other, draft.id)).toBeUndefined();
});

it('rejects malformed recovery before it can issue a fresh publication', async () => {
	const { decodeDraft } = await import('./store');
	const draft = newDraft('did:plc:alice');
	for (const pending of [
		{ body: { text: 'PRIVATE_DRAFT_CANARY' } },
		{ key: crypto.randomUUID(), body: 'PRIVATE_DRAFT_CANARY' },
		{ key: crypto.randomUUID(), body: {}, kind: 'edit' },
		{ key: crypto.randomUUID(), body: {}, kind: 'unknown' }
	]) {
		expect(() => decodeDraft({ ...draft, pending }, draft.owner)).toThrow(
			'saved publication'
		);
	}
	expect(() => decodeDraft({ ...draft, images: [null] }, draft.owner)).toThrow(
		'saved draft'
	);
});
