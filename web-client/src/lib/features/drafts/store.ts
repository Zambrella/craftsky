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
		throw new Error('This draft belongs to another account.');
}
let database: Promise<IDBDatabase> | undefined;
function open(): Promise<IDBDatabase> {
	if (database) return database;
	database = new Promise<IDBDatabase>((resolve, reject) => {
		if (typeof indexedDB === 'undefined') {
			reject(new Error('This browser does not support local drafts.'));
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
			reject(new Error('Close older CraftSky tabs to open local drafts.'));
		request.onerror = () =>
			reject(new Error('Local draft storage could not be opened.'));
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
				new Error('Could not save local drafts. Check your browser storage.')
			);
		tx.onerror = () =>
			reject(
				new Error('Could not save local drafts. Check your browser storage.')
			);
	});
}
export const drafts = {
	/** Account-deletion cleanup only. All owner rows and media disappear in one transaction. */
	async clear(owner: string): Promise<void> {
		if (!owner.startsWith('did:'))
			throw new Error('A draft owner is required.');
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
			.filter((draft) => draft.owner === owner)
			.sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
	},
	async get(owner: string, id: string): Promise<Draft | undefined> {
		const draft = (await transaction('readonly', (store) =>
			store.get([owner, id])
		)) as Draft | undefined;
		if (draft) assertOwner(owner, draft);
		return draft;
	},
	async save(owner: string, draft: Draft): Promise<void> {
		assertOwner(owner, draft);
		const copy = structuredClone(draft);
		copy.updatedAt = new Date().toISOString();
		await transaction('readwrite', (store) => store.put(copy));
	},
	/** Explicit Discard restores the activation's original private snapshot without changing its timestamp. */
	async restore(owner: string, draft: Draft): Promise<void> {
		assertOwner(owner, draft);
		await transaction('readwrite', (store) =>
			store.put(structuredClone(draft))
		);
	},
	async remove(owner: string, id: string): Promise<void> {
		await transaction('readwrite', (store) => store.delete([owner, id]));
	}
};
