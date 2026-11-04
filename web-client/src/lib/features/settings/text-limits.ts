import { graphemeCount as characterCount } from '#lib/text.ts';
export { graphemeCount as characterCount } from '#lib/text.ts';
const encoder = new TextEncoder();
/** Keep both Lexicon limits; HTML maxlength counts UTF-16 code units instead. */
export function requireRecordText(
	value: string,
	label: string,
	characters: number,
	bytes: number
) {
	if (
		characterCount(value) > characters ||
		encoder.encode(value).length > bytes
	)
		throw new Error(`${label} is too long.`);
}
