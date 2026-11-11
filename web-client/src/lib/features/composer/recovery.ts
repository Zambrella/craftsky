import { UserFacingError } from '#lib/user-facing-error.ts';
import {
	assertOwner,
	drafts,
	type Draft,
	type DraftMedia
} from '../drafts/store';

type Persistence = Pick<typeof drafts, 'save' | 'remove'>;
/** One editor activation owns its durable queue and the snapshot explicit Discard restores. */
export class ComposerRecovery {
	private tail: Promise<void> = Promise.resolve();
	private phase: 'active' | 'discarding' | 'closed' = 'active';
	private original?: Draft;
	private revision: string | null;
	private readonly baseline: string;
	private blobs = new WeakMap<Blob, number>();
	private nextBlob = 1;
	constructor(
		readonly owner: string,
		readonly id: string,
		initial: Draft,
		existing: boolean,
		private store: Persistence = drafts
	) {
		this.assert(initial);
		this.baseline = this.fingerprint(initial);
		this.original =
			existing && !initial.pending ? structuredClone(initial) : undefined;
		this.revision = existing ? (initial.revision ?? initial.updatedAt) : null;
	}
	get active() {
		return this.phase === 'active';
	}
	/** Compare the live editor value; persistence snapshots clone otherwise immutable Blob objects. */
	hasChanges(value: Draft): boolean {
		this.assert(value);
		return this.fingerprint(value) !== this.baseline;
	}
	private fingerprint(value: Draft): string {
		const media = (item: DraftMedia | undefined) =>
			item && {
				...item,
				uploaded: undefined,
				blob: item.blob ? this.blobIdentity(item.blob) : undefined
			};
		return JSON.stringify({
			text: value.text,
			langs: value.langs,
			sponsored: value.sponsored,
			kind: value.kind,
			project: value.project,
			instagramImport: value.instagramImport,
			images: value.images.map(media),
			video: media(value.video),
			external: value.external,
			externalImage: media(value.externalImage),
			externalDismissed: !!value.externalDismissed,
			embedChanges: value.embedChanges,
			scheduleLater: !!value.scheduleLater,
			scheduledAt: value.scheduledAt
		});
	}
	private blobIdentity(blob: Blob): number {
		let identity = this.blobs.get(blob);
		if (!identity) {
			identity = this.nextBlob++;
			this.blobs.set(blob, identity);
		}
		return identity;
	}
	private assert(value: Draft) {
		assertOwner(this.owner, value);
		if (value.id !== this.id)
			throw new UserFacingError('This editor belongs to another draft.');
	}
	private enqueue(action: () => Promise<void>): Promise<void> {
		const result = this.tail.then(action, action);
		this.tail = result.catch(() => {});
		return result;
	}
	checkpoint(value: Draft): Promise<void> {
		this.assert(value);
		if (!this.active)
			return Promise.reject(
				new UserFacingError('This editor has already closed.')
			);
		const frozen = structuredClone(value);
		return this.enqueue(async () => {
			this.revision = await this.store.save(this.owner, frozen, this.revision);
			if (!frozen.pending)
				this.original = { ...frozen, revision: this.revision };
		});
	}
	/** Unsaved edits live only in memory. Discard must never restore a stale row. */
	discard(value: Draft): Promise<void> {
		this.assert(value);
		if (value.pending)
			return Promise.reject(
				new UserFacingError(
					'Check the original action before discarding this draft.'
				)
			);
		if (!this.active)
			return Promise.reject(
				new UserFacingError('This editor has already closed.')
			);
		this.phase = 'closed';
		return this.tail;
	}
	/** A definitive rejection removes only recovery, restoring the last explicit save. */
	rejected(): Promise<void> {
		return this.enqueue(async () => {
			if (this.original)
				this.revision = await this.store.save(
					this.owner,
					this.original,
					this.revision
				);
			else {
				this.revision = await this.store.remove(
					this.owner,
					this.id,
					this.revision
				);
			}
		});
	}
	/** Confirmed publication may clear its recovery row; stale editor writes must never recreate it. */
	complete(): Promise<void> {
		this.phase = 'closed';
		return this.enqueue(async () => {
			await this.store.remove(this.owner, this.id, this.revision);
		});
	}
	abandon(): void {
		this.phase = 'closed';
	}
}
