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
