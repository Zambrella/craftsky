const segmenter = new Intl.Segmenter(undefined, { granularity: 'grapheme' });
export const characterCount = (value: string) =>
	[...segmenter.segment(value)].length;
/** Keep both Lexicon limits; HTML maxlength counts UTF-16 code units instead. */
export function requireRecordText(
	value: string,
	label: string,
	characters: number,
	bytes: number
) {
	if (
		characterCount(value) > characters ||
		new TextEncoder().encode(value).length > bytes
	)
		throw new Error(`${label} is too long.`);
}
