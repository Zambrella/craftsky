import { graphemeCount } from '#lib/text.ts';
export interface ActiveFacetToken {
	kind: 'mention' | 'hashtag';
	start: number;
	end: number;
	query: string;
}
const characters = (kind: ActiveFacetToken['kind']) =>
	kind === 'mention' ? /^[A-Za-z0-9._-]$/ : /^[\p{L}\p{N}_]$/u;
/** UTF-16 caret indices match browser selectionStart, independently of published UTF-8 offsets. */
export function activeFacetToken(
	text: string,
	caret: number,
	end = caret
): ActiveFacetToken | undefined {
	if (caret !== end || caret < 0 || caret > text.length) return;
	const match = /(?:^|[\s([{])([@#])([^\s@#]*)$/u.exec(text.slice(0, caret));
	if (!match) return;
	const kind = match[1] === '@' ? 'mention' : 'hashtag';
	const query = match[2];
	const allowed = characters(kind);
	if ([...query].some((char) => !allowed.test(char))) return;
	let tokenEnd = caret;
	for (const char of text.slice(caret)) {
		if (!allowed.test(char)) break;
		tokenEnd += char.length;
	}
	return { kind, start: caret - query.length - 1, end: tokenEnd, query };
}
export function replaceFacetToken(
	text: string,
	token: ActiveFacetToken,
	value: string
): { text: string; caret: number } {
	const clean = value.replace(/^[@#]/, '');
	if (!clean || [...clean].some((char) => !characters(token.kind).test(char)))
		throw new Error('Invalid suggestion');
	const replacement = (token.kind === 'mention' ? '@' : '#') + clean;
	const suffix = text.slice(token.end);
	const space = suffix.length === 0 ? ' ' : '';
	return {
		text: text.slice(0, token.start) + replacement + space + suffix,
		caret: token.start + replacement.length + space.length
	};
}

/** Display ranges only; mention identity is still resolved when publishing. */
export function facetSegments(
	text: string,
	precedingHashtags: string[] = []
): { text: string; facet: boolean }[] {
	const parts: { text: string; facet: boolean }[] = [];
	const accepted = acceptedHashtags([
		...precedingHashtags,
		...hashtagsInText(text)
	]);
	const pattern =
		/(https?:\/\/[^\s<>]+)|(?:^|[\s(])(@[a-zA-Z0-9][a-zA-Z0-9.-]*\.[a-zA-Z]{2,})|(?:^|[\s(])(#[\p{L}\p{N}_]+)/gu;
	let offset = 0;
	for (const match of text.matchAll(pattern)) {
		const token = match[1] || match[2] || match[3];
		const start = match.index! + match[0].indexOf(token);
		if (start > offset)
			parts.push({ text: text.slice(offset, start), facet: false });
		parts.push({
			text: token,
			facet: !match[3] || accepted.has(token.slice(1).toLowerCase())
		});
		offset = start + token.length;
	}
	if (offset < text.length)
		parts.push({ text: text.slice(offset), facet: false });
	return parts;
}

export const maxPostHashtags = 10;
export function acceptedHashtags(tags: Iterable<string>): Set<string> {
	const accepted = new Set<string>();
	for (const raw of tags) {
		const tag = raw.trim().toLowerCase();
		if (tag && accepted.size < maxPostHashtags) accepted.add(tag);
	}
	return accepted;
}
export function hashtagsInText(text: string): string[] {
	const pattern =
		/(https?:\/\/[^\s<>]+)|(?:^|[\s(])(@[a-zA-Z0-9][a-zA-Z0-9.-]*\.[a-zA-Z]{2,})|(?:^|[\s(])(#[\p{L}\p{N}_]+)/gu;
	return [...text.matchAll(pattern)].flatMap((match) => {
		const tag = match[3]?.slice(1);
		return tag &&
			graphemeCount(tag) <= 64 &&
			new TextEncoder().encode(tag).length <= 640
			? [tag]
			: [];
	});
}
