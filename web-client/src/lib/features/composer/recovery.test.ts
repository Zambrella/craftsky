import 'fake-indexeddb/auto';
import { describe, it, expect } from 'vitest';
import { ComposerRecovery } from './recovery';
import { drafts, newDraft } from '../drafts/store';
describe('explicit draft saves and publication recovery', () => {
	it('does not persist edits on discard or account disposal', async () => {
		const initial = newDraft('did:plc:alice');
		const editor = new ComposerRecovery(
			initial.owner,
			initial.id,
			initial,
			false
		);
		await editor.discard({ ...initial, text: 'Unsaved' });
		expect(await drafts.get(initial.owner, initial.id)).toBeUndefined();
		const another = new ComposerRecovery(
			initial.owner,
			initial.id,
			initial,
			false
		);
		another.abandon();
		expect(await drafts.get(initial.owner, initial.id)).toBeUndefined();
	});
	it('does not overwrite another tab when discarding unsaved edits', async () => {
		const initial = newDraft('did:plc:alice');
		initial.revision = await drafts.save(initial.owner, initial);
		const a = new ComposerRecovery(initial.owner, initial.id, initial, true);
		const b = new ComposerRecovery(initial.owner, initial.id, initial, true);
		await a.checkpoint({ ...initial, text: 'New explicit save' });
		await b.discard({ ...initial, text: 'Discard me' });
		expect((await drafts.get(initial.owner, initial.id))?.text).toBe(
			'New explicit save'
		);
	});
	it('atomically rejects stale explicit saves, deletes and publication checkpoints', async () => {
		const initial = newDraft('did:plc:alice');
		const a = new ComposerRecovery(initial.owner, initial.id, initial, false);
		const b = new ComposerRecovery(initial.owner, initial.id, initial, false);
		const pending = {
			...initial,
			text: 'Frozen',
			pending: { key: crypto.randomUUID(), body: { text: 'Frozen' } }
		};
		await a.checkpoint(pending);
		await expect(
			b.checkpoint({ ...initial, text: 'Stale edit' })
		).rejects.toThrow('another tab');
		await expect(
			drafts.remove(initial.owner, initial.id, null)
		).rejects.toThrow('another tab');
		await expect(a.discard(pending)).rejects.toThrow('original action');
		await a.complete();
		await expect(b.checkpoint(pending)).rejects.toThrow('another tab');
		expect(await drafts.get(initial.owner, initial.id)).toBeUndefined();
		expect(await drafts.list(initial.owner)).not.toContainEqual(
			expect.objectContaining({ id: initial.id })
		);
	});
	it('keeps only an explicit save after definitive rejection and allows another deliberate attempt', async () => {
		for (const existing of [true, false]) {
			const initial = newDraft('did:plc:alice');
			initial.text = 'Explicit save';
			if (existing)
				initial.revision = await drafts.save(initial.owner, initial);
			const editor = new ComposerRecovery(
				initial.owner,
				initial.id,
				initial,
				existing
			);
			await editor.checkpoint({
				...initial,
				text: 'Submitted',
				pending: { key: crypto.randomUUID(), body: { text: 'Submitted' } }
			});
			await editor.rejected();
			expect((await drafts.get(initial.owner, initial.id))?.text).toBe(
				existing ? 'Explicit save' : undefined
			);
			await editor.checkpoint({ ...initial, text: 'Explicit retry save' });
			expect((await drafts.get(initial.owner, initial.id))?.text).toBe(
				'Explicit retry save'
			);
		}
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
