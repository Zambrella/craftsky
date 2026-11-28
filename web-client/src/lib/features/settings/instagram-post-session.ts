import type { InstagramArchive } from '../instagram-import';
import type { InstagramPublishedImport } from '#lib/api/types.ts';

export interface InstagramPostSession {
	key: string;
	signal: AbortSignal;
	archive?: InstagramArchive;
	fileName: string;
	visible: number;
	posted: Record<string, InstagramPublishedImport>;
	previews: Record<string, string>;
}
let current: InstagramPostSession | undefined;
let lifetime: AbortController | undefined;

/** Browser memory only: leaving for the composer retains the ZIP; leaving the account does not. */
export function clearInstagramPostSession(): void {
	lifetime?.abort();
	lifetime = undefined;
	current?.archive?.close();
	for (const url of Object.values(current?.previews ?? {}))
		URL.revokeObjectURL(url);
	current = undefined;
}
export function useInstagramPostSession(key: string): InstagramPostSession {
	if (current?.key !== key) {
		clearInstagramPostSession();
		lifetime = new AbortController();
		current = {
			key,
			signal: lifetime.signal,
			fileName: '',
			visible: 20,
			posted: {},
			previews: {}
		};
	}
	return current;
}
export function importedPostPath(uri: string): string | undefined {
	const match =
		/^at:\/\/(did:[^/]+)\/social\.craftsky\.feed\.post\/([^/]+)$/.exec(uri);
	return match
		? `/posts/${encodeURIComponent(match[1])}/${encodeURIComponent(match[2])}`
		: undefined;
}
