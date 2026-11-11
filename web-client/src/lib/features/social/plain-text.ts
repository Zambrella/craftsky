import { safeLink, type TextPart } from './model';
import { graphemeCount } from '#lib/text.ts';

/** Profile descriptions have no stored facets; detect the same tokens as mobile. */
export function plainTextParts(text: string): TextPart[] {
	const tokens: { start: number; end: number; part: TextPart }[] = [];
	const patterns = [
		/(^|[\s([{])@([A-Za-z0-9](?:[A-Za-z0-9.-]*[A-Za-z0-9])?\.[A-Za-z][A-Za-z0-9.-]*)/gu,
		/(^|[\s([{])((?:https?:\/\/)?(?:[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?\.)+[A-Za-z]{2,}(?::[0-9]+)?(?:\/[^\s<>]*)?)/giu,
		/(^|[^\p{L}\p{N}_])#([\p{L}\p{N}_]+)/gu
	];
	for (const [kind, pattern] of patterns.entries()) {
		for (const match of text.matchAll(pattern)) {
			const start = match.index + match[1].length;
			let token = match[2];
			let href: string | undefined;
			if (kind === 0) {
				href = '/profiles/' + encodeURIComponent(token);
				token = '@' + token;
			} else if (kind === 1) {
				token = trimLink(token);
				href = safeLink(
					/^https?:\/\//i.test(token) ? token : 'https://' + token
				);
			} else {
				if (
					graphemeCount(token) > 64 ||
					new TextEncoder().encode(token).length > 640
				)
					continue;
				href = '/search/tags?tag=' + encodeURIComponent(token);
				token = '#' + token;
			}
			if (href)
				tokens.push({
					start,
					end: start + token.length,
					part: { text: token, href, external: kind === 1 }
				});
		}
	}
	const parts: TextPart[] = [];
	let end = 0;
	for (const token of tokens.sort(
		(a, b) => a.start - b.start || b.end - a.end
	)) {
		if (token.start < end) continue;
		if (token.start > end) parts.push({ text: text.slice(end, token.start) });
		parts.push(token.part);
		end = token.end;
	}
	if (end < text.length || !parts.length) parts.push({ text: text.slice(end) });
	return parts;
}

function trimLink(value: string): string {
	let text = value.replace(/[.,!?;:]+$/, '');
	for (const [open, close] of [
		['(', ')'],
		['[', ']'],
		['{', '}']
	]) {
		while (
			text.endsWith(close) &&
			text.split(close).length > text.split(open).length
		) {
			text = text.slice(0, -1);
		}
	}
	return text;
}
