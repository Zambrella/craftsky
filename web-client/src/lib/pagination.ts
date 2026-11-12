/** Append a page in server order, keeping the first item for each stable key. */
export function appendPage<T, K>(
	current: readonly T[],
	incoming: readonly T[],
	key: (item: T) => K
): T[] {
	const seen = new Set(current.map(key));
	const result = [...current];
	for (const item of incoming) {
		const id = key(item);
		if (!seen.has(id)) {
			seen.add(id);
			result.push(item);
		}
	}
	return result;
}

/** Build a refreshed range before publishing it, preserving loaded page depth. */
export async function reloadPages<T, K>(
	fetch: (
		cursor?: string | null
	) => Promise<{ items: T[]; cursor?: string | null }>,
	pages: number,
	key: (item: T) => K,
	minimumItems = 0
): Promise<{ items: T[]; cursor: string | null; pages: number }> {
	let page = await fetch();
	let items = appendPage([], page.items, key);
	let loaded = 1;
	const visited = new Set<string>();
	while (
		(loaded < pages || items.length < minimumItems) &&
		page.cursor &&
		!visited.has(page.cursor)
	) {
		visited.add(page.cursor);
		page = await fetch(page.cursor);
		items = appendPage(items, page.items, key);
		loaded++;
	}
	return { items, cursor: page.cursor || null, pages: loaded };
}
