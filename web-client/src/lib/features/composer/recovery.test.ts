import 'fake-indexeddb/auto';
import { describe, it, expect, vi } from 'vitest';
import { ComposerRecovery } from './recovery';
import { drafts, newDraft, type Draft } from '../drafts/store';
function persistence() {
	const values = new Map<string, Draft>(),
		key = (owner: string, id: string) => owner + '/' + id;
	const save = vi.fn(async (owner: string, value: Draft) => {
		values.set(key(owner, value.id), structuredClone(value));
	});
	return {
		values,
		key,
		store: {
			save,
			restore: vi.fn(save),
			remove: vi.fn(async (owner: string, id: string) => {
				values.delete(key(owner, id));
			})
		}
	};
}
describe('activation-owned composer recovery', () => {
	it('queues explicit discard behind an in-flight autosave and prevents later autosaves from recreating a new draft', async () => {
		const initial = newDraft('did:plc:alice'),
			current = { ...initial, text: 'Private progress' },
			{ store, values, key } = persistence();
		let finish!: () => void;
		store.save.mockImplementationOnce(async (owner, value) => {
			await new Promise<void>((resolve) => (finish = resolve));
			values.set(key(owner, value.id), structuredClone(value));
		});
		const lifecycle = new ComposerRecovery(
			initial.owner,
			initial.id,
			initial,
			false,
			store
		);
		const saving = lifecycle.checkpoint(current);
		await vi.waitFor(() => expect(store.save).toHaveBeenCalledOnce());
		const discarding = lifecycle.discard(current);
		await expect(lifecycle.checkpoint(current)).rejects.toThrow(
			'already closed'
		);
		expect(store.remove).not.toHaveBeenCalled();
		finish();
		await saving;
		await discarding;
		expect(values.size).toBe(0);
		expect(store.remove).toHaveBeenCalledWith(initial.owner, initial.id);
	});
	it('restores the original saved text, timestamp and media bytes after autosaved edits without touching another owner', async () => {
		const owner = `did:plc:${crypto.randomUUID()}`,
			other = `did:plc:${crypto.randomUUID()}`,
			initial = newDraft(owner);
		initial.text = 'Original';
		initial.images = [
			{
				id: 'image',
				name: 'private.jpg',
				blob: new Blob(['original bytes']),
				mime: 'image/jpeg',
				alt: 'Original alt'
			}
		];
		await drafts.restore(owner, initial);
		await drafts.restore(other, {
			...initial,
			owner: other,
			text: 'Other account'
		});
		const lifecycle = new ComposerRecovery(owner, initial.id, initial, true);
		const changed = {
			...initial,
			text: 'Autosaved edit',
			images: [
				{
					...initial.images[0],
					blob: new Blob(['edited bytes']),
					alt: 'Edited alt'
				}
			]
		};
		await lifecycle.checkpoint(changed);
		expect((await drafts.get(owner, initial.id))?.text).toBe(changed.text);
		await lifecycle.discard(changed);
		const restored = await drafts.get(owner, initial.id);
		expect(restored).toMatchObject({
			text: 'Original',
			updatedAt: initial.updatedAt,
			images: [{ alt: 'Original alt' }]
		});
		expect(await restored?.images[0].blob?.text()).toBe('original bytes');
		expect((await drafts.get(other, initial.id))?.text).toBe('Other account');
	});
	it('keeps recovery and allows retry when an explicit discard cannot commit', async () => {
		const initial = newDraft('did:plc:alice'),
			current = { ...initial, text: 'Keep me' },
			{ store, values, key } = persistence();
		const lifecycle = new ComposerRecovery(
			initial.owner,
			initial.id,
			initial,
			false,
			store
		);
		await lifecycle.checkpoint(current);
		store.remove.mockRejectedValueOnce(new Error('Storage unavailable'));
		await expect(lifecycle.discard(current)).rejects.toThrow(
			'Storage unavailable'
		);
		expect(lifecycle.active).toBe(true);
		expect(values.get(key(initial.owner, initial.id))?.text).toBe('Keep me');
		await lifecycle.discard(current);
		expect(values.size).toBe(0);
	});
	it('never discards an uncertain publication or accepts another account/draft snapshot', async () => {
		const initial = newDraft('did:plc:alice'),
			{ store } = persistence(),
			lifecycle = new ComposerRecovery(
				initial.owner,
				initial.id,
				initial,
				false,
				store
			);
		const pending = {
			...initial,
			text: 'Frozen',
			pending: { key: crypto.randomUUID(), body: { text: 'Frozen' } }
		};
		await lifecycle.checkpoint(pending);
		await expect(lifecycle.discard(pending)).rejects.toThrow('original action');
		expect(store.remove).not.toHaveBeenCalled();
		expect(() =>
			lifecycle.checkpoint({ ...initial, owner: 'did:plc:bob' })
		).toThrow('another account');
		expect(() => lifecycle.discard({ ...initial, id: 'another' })).toThrow(
			'another draft'
		);
	});
	it('leaves queued recovery under its original owner after account invalidation and suppresses future writes', async () => {
		const initial = newDraft('did:plc:alice'),
			{ store, values, key } = persistence(),
			lifecycle = new ComposerRecovery(
				initial.owner,
				initial.id,
				initial,
				false,
				store
			);
		const saving = lifecycle.checkpoint({
			...initial,
			text: 'Last Alice edit'
		});
		lifecycle.abandon();
		await saving;
		expect(values.get(key(initial.owner, initial.id))?.text).toBe(
			'Last Alice edit'
		);
		expect(values.get(key('did:plc:bob', initial.id))).toBeUndefined();
		await expect(lifecycle.checkpoint(initial)).rejects.toThrow(
			'already closed'
		);
	});
	it('closes the checkpoint queue before confirmed publication cleanup, including cleanup failure', async () => {
		const initial = newDraft('did:plc:alice'),
			{ store } = persistence(),
			lifecycle = new ComposerRecovery(
				initial.owner,
				initial.id,
				initial,
				false,
				store
			);
		await lifecycle.checkpoint({
			...initial,
			pending: { key: crypto.randomUUID(), body: { text: 'Confirmed' } }
		});
		store.remove.mockRejectedValueOnce(new Error('quota'));
		await expect(lifecycle.complete()).rejects.toThrow('quota');
		await expect(lifecycle.checkpoint(initial)).rejects.toThrow(
			'already closed'
		);
		expect(store.save).toHaveBeenCalledOnce();
	});
	it('keeps a live media baseline clean after persisting cloned recovery snapshots', async () => {
		const initial = newDraft('did:plc:alice');
		initial.images = [
			{
				id: 'image',
				name: 'saved.jpg',
				blob: new Blob(['saved bytes']),
				mime: 'image/jpeg',
				alt: 'Saved image'
			}
		];
		const { store } = persistence();
		const lifecycle = new ComposerRecovery(
			initial.owner,
			initial.id,
			initial,
			true,
			store
		);
		const snapshot = structuredClone(initial);
		expect(snapshot.images[0].blob).not.toBe(initial.images[0].blob);
		await lifecycle.checkpoint(snapshot);
		expect(lifecycle.hasChanges(initial)).toBe(false);
		expect(
			lifecycle.hasChanges({
				...initial,
				images: [{ ...initial.images[0], alt: 'Changed description' }]
			})
		).toBe(true);
	});
	it('tracks user metadata and replacement bytes while ignoring timestamps and completed upload references', () => {
		const initial = newDraft('did:plc:alice');
		initial.images = [
			{
				id: 'image',
				name: 'x.jpg',
				blob: new Blob(['before']),
				mime: 'image/jpeg',
				alt: ''
			}
		];
		const lifecycle = new ComposerRecovery(
			initial.owner,
			initial.id,
			initial,
			true
		);
		expect(lifecycle.hasChanges(initial)).toBe(false);
		expect(lifecycle.hasChanges({ ...initial, updatedAt: 'later' })).toBe(
			false
		);
		expect(lifecycle.hasChanges({ ...initial, sponsored: true })).toBe(true);
		expect(lifecycle.hasChanges({ ...initial, langs: ['fr'] })).toBe(true);
		expect(
			lifecycle.hasChanges({
				...initial,
				scheduleLater: true,
				scheduledAt: '2026-11-04T12:00:00Z'
			})
		).toBe(true);
		expect(
			lifecycle.hasChanges({
				...initial,
				images: [{ ...initial.images[0], blob: new Blob(['after']) }]
			})
		).toBe(true);
		expect(
			lifecycle.hasChanges({
				...initial,
				images: [
					{
						...initial.images[0],
						uploaded: {
							$type: 'blob',
							ref: { $link: 'uploaded' },
							mimeType: 'image/jpeg',
							size: 5
						}
					}
				]
			})
		).toBe(false);
	});
});
