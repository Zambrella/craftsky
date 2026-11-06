import { InputError, UserFacingError } from '#lib/user-facing-error.ts';
import type {
	BlobReference,
	JsonObject,
	PostEditSeed,
	StrongRef
} from '#lib/api/client.ts';

export interface DraftMedia {
	id: string;
	name: string;
	blob?: Blob;
	mime: string;
	alt: string;
	width?: number;
	height?: number;
	existingIndex?: number;
	remoteUrl?: string;
	uploaded?: BlobReference;
	scheduledId?: string;
}
export interface PendingWrite {
	key: string;
	body: JsonObject;
	did?: string;
	rkey?: string;
	cid?: string;
	kind?:
		| 'create'
		| 'edit'
		| 'schedule-create'
		| 'schedule-update'
		| 'schedule-publish';
}
export interface Draft {
	revision?: string;
	id: string;
	owner: string;
	updatedAt: string;
	kind: 'standard' | 'project';
	text: string;
	langs: string[];
	sponsored: boolean;
	project?: JsonObject;
	images: DraftMedia[];
	video?: DraftMedia;
	videoRecovery?: number;
	reply?: { root: StrongRef; parent: StrongRef };
	quote?: StrongRef;
	quoteImageCid?: string;
	external?: {
		uri: string;
		title: string;
		description: string;
		thumb?: BlobReference;
	};
	externalImage?: DraftMedia;
	externalDismissed?: boolean;
	externalSource?: string;
	externalResolvedURI?: string;
	embedChanges?: JsonObject | null;
	edit?: PostEditSeed;
	scheduleId?: string;
	/** Device-private scheduling choice; never part of a public post payload. */
	scheduleLater?: boolean;
	scheduledAt?: string;
	pending?: PendingWrite;
}
export function newDraft(
	owner: string,
	kind: Draft['kind'] = 'standard'
): Draft {
	return {
		id: crypto.randomUUID(),
		owner,
		updatedAt: new Date().toISOString(),
		kind,
		text: '',
		langs: ['en'],
		sponsored: false,
		images: []
	};
}
export function assertOwner(owner: string, draft: Draft): void {
	if (!owner.startsWith('did:') || owner !== draft.owner)
		throw new UserFacingError('This draft belongs to another account.');
}
let database: Promise<IDBDatabase> | undefined;
function open(): Promise<IDBDatabase> {
	if (database) return database;
	database = new Promise<IDBDatabase>((resolve, reject) => {
		if (typeof indexedDB === 'undefined') {
			reject(
				new UserFacingError('This browser does not support local drafts.')
			);
			return;
		}
		const request = indexedDB.open('craftsky-private-drafts', 1);
		request.onupgradeneeded = () => {
			const store = request.result.createObjectStore('drafts', {
				keyPath: ['owner', 'id']
			});
			store.createIndex('owner', 'owner');
		};
		request.onblocked = () =>
			reject(
				new UserFacingError('Close older CraftSky tabs to open local drafts.')
			);
		request.onerror = () =>
			reject(
				new UserFacingError('Local draft storage could not be opened.', {
					cause: request.error
				})
			);
		request.onsuccess = () => {
			const db = request.result;
			db.onversionchange = () => {
				db.close();
				database = undefined;
			};
			resolve(db);
		};
	}).catch((error) => {
		database = undefined;
		throw error;
	});
	return database;
}
/** Resolve writes only after the transaction commits, including the media blobs. */
async function transaction<T>(
	mode: IDBTransactionMode,
	action: (store: IDBObjectStore) => IDBRequest<T>
): Promise<T> {
	const db = await open();
	return new Promise((resolve, reject) => {
		const tx = db.transaction('drafts', mode);
		let result: T;
		const request = action(tx.objectStore('drafts'));
		request.onsuccess = () => {
			result = request.result;
		};
		tx.oncomplete = () => resolve(result);
		tx.onabort = () =>
			reject(
				new UserFacingError(
					'Could not save local drafts. Check your browser storage.',
					{ cause: tx.error }
				)
			);
		tx.onerror = () =>
			reject(
				new UserFacingError(
					'Could not save local drafts. Check your browser storage.',
					{ cause: tx.error }
				)
			);
	});
}
export const drafts = {
	/** Account-deletion cleanup only. All owner rows and media disappear in one transaction. */
	async clear(owner: string): Promise<void> {
		if (!owner.startsWith('did:'))
			throw new UserFacingError('A draft owner is required.');
		await transaction('readwrite', (store) => {
			const request = store.index('owner').getAllKeys(owner);
			request.addEventListener('success', () => {
				for (const key of request.result) store.delete(key);
			});
			return request;
		});
	},
	async list(owner: string): Promise<Draft[]> {
		return (
			(await transaction('readonly', (store) =>
				store.index('owner').getAll(owner)
			)) as Draft[]
		)
			.filter(
				(draft) =>
					!(draft as Draft & { deleted?: boolean }).deleted &&
					draft.owner === owner
			)
			.map((draft) => decodeDraft(draft, owner))
			.sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
	},
	async get(owner: string, id: string): Promise<Draft | undefined> {
		const draft = (await transaction('readonly', (store) =>
			store.get([owner, id])
		)) as Draft | undefined;
		if ((draft as (Draft & { deleted?: boolean }) | undefined)?.deleted) return;
		return draft ? decodeDraft(draft, owner) : undefined;
	},
	async save(
		owner: string,
		draft: Draft,
		expected: string | null = draft.revision ?? null
	): Promise<string> {
		assertOwner(owner, draft);
		const copy = structuredClone(draft);
		copy.updatedAt = new Date().toISOString();
		copy.revision = crypto.randomUUID();
		await change(owner, draft.id, expected, copy);
		return copy.revision;
	},
	async remove(
		owner: string,
		id: string,
		expected: string | null
	): Promise<string> {
		const revision = crypto.randomUUID();
		await change(owner, id, expected, { owner, id, deleted: true, revision });
		return revision;
	}
};

/** Compare and replace in one IndexedDB transaction, including deletion tombstones.
 * Tombstones contain no draft content and prevent an old, never-saved editor resurrecting a completed item.
 */
async function change(
	owner: string,
	id: string,
	expected: string | null,
	value: unknown
): Promise<void> {
	let conflict: Error | undefined;
	try {
		await transaction('readwrite', (store) => {
			const read = store.get([owner, id]);
			read.addEventListener('success', () => {
				const current = read.result;
				if (
					(current ? (current.revision ?? current.updatedAt) : null) !==
					expected
				) {
					conflict = new InputError(
						'This draft changed in another tab. Your edits are still here. Open the latest draft before saving or publishing.'
					);
					store.transaction.abort();
				} else store.put(value);
			});
			return read;
		});
	} catch (error) {
		throw conflict ?? error;
	}
}

/** Validate durable recovery before it can drive a write. Never infer a missing operation key. */
export function decodeDraft(value: unknown, owner: string): Draft {
	const draft = value as Draft;
	const object = (v: unknown): v is Record<string, unknown> =>
		!!v && typeof v === 'object' && !Array.isArray(v);
	const media = (v: unknown) =>
		object(v) &&
		typeof v.id === 'string' &&
		typeof v.mime === 'string' &&
		typeof v.alt === 'string' &&
		(!v.blob || v.blob instanceof Blob);
	if (
		!object(value) ||
		draft.owner !== owner ||
		typeof draft.id !== 'string' ||
		typeof draft.updatedAt !== 'string' ||
		typeof draft.text !== 'string' ||
		!['standard', 'project'].includes(draft.kind) ||
		typeof draft.sponsored !== 'boolean' ||
		!Array.isArray(draft.langs) ||
		!draft.langs.every((v) => typeof v === 'string') ||
		!Array.isArray(draft.images) ||
		!draft.images.every(media) ||
		(draft.video !== undefined && !media(draft.video)) ||
		(draft.externalImage !== undefined && !media(draft.externalImage)) ||
		(draft.revision !== undefined && typeof draft.revision !== 'string')
	)
		throw new UserFacingError('This saved draft could not be read.');
	if (draft.pending) {
		const p = draft.pending;
		if (
			!object(p) ||
			!/^[0-9a-f-]{36}$/.test(p.key) ||
			!object(p.body) ||
			(p.kind !== undefined &&
				![
					'create',
					'edit',
					'schedule-create',
					'schedule-update',
					'schedule-publish'
				].includes(p.kind)) ||
			(p.kind === 'edit' && (!p.did || !p.rkey || !p.cid)) ||
			(['schedule-update', 'schedule-publish'].includes(p.kind ?? '') &&
				!p.rkey)
		)
			throw new UserFacingError(
				'This saved publication could not be read. Keep it until its outcome can be checked.'
			);
	}
	return draft;
}
