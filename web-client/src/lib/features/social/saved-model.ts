import type { CraftskyAPI, SavedFolder } from '#lib/api/client.ts';
/** Resolve every folder page for the compact picker, retaining opaque cursors unchanged. */
export async function allFolders(api: CraftskyAPI): Promise<SavedFolder[]> {
	const folders: SavedFolder[] = [];
	const seen = new Set<string>();
	const folderIds = new Set<string>();
	let cursor: string | null | undefined;
	for (let page = 0; page < 100; page++) {
		const result = await api.saved.folders({ limit: 100, cursor });
		for (const folder of result.items)
			if (!folderIds.has(folder.id)) {
				folderIds.add(folder.id);
				folders.push(folder);
			}
		cursor = result.cursor;
		if (!cursor) return folders;
		if (seen.has(cursor))
			throw new Error('The folder list could not be completed. Try again.');
		seen.add(cursor);
	}
	throw new Error('The folder list is too large to load.');
}
