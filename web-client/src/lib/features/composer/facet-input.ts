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
