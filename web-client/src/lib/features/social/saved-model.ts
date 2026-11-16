import type { CraftskyAPI, SavedFolder } from '#lib/api/client.ts';
import { FieldValidationError } from '#lib/field-validation.ts';

/** Match the folder-name contract used by the mobile client and AppView. */
export function normalizeSavedFolderName(value: string): string {
	const name = value.trim();
	if (!name)
		throw new FieldValidationError('folderName', 'Enter a folder name.');
	if (Array.from(name).length > 100)
		throw new FieldValidationError(
			'folderName',
			'Use 100 characters or fewer.'
		);
	if (/[\\/\u0000-\u001f\u007f-\u009f]/u.test(name))
		throw new FieldValidationError(
			'folderName',
			'Folder names cannot contain slashes or control characters.'
		);
	return name;
}

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
