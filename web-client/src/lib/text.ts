/** Shared Unicode segmentation: limits count user-perceived characters, not UTF-16 units. */
const segmenter = new Intl.Segmenter(undefined, { granularity: 'grapheme' });
export const graphemes = (text: string) => segmenter.segment(text);

export function graphemeCount(text: string): number {
	let count = 0;
	for (const _ of graphemes(text)) count++;
	return count;
}

/** UTF-16 offset after at most `limit` graphemes, without allocating the entire input. */
export function graphemeBoundary(text: string, limit: number): number {
	let count = 0;
	for (const { index } of graphemes(text)) {
		if (count++ >= limit) return index;
	}
	return text.length;
}

export function excerpt(text: string, limit: number): string {
	const end = graphemeBoundary(text, limit);
	return end < text.length ? text.slice(0, end) + '…' : text;
}
