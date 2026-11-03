import labels from './languages.json';

/** Keep the picker aligned with Flutter's v1 language catalogue. */
export const languageName = (tag: string): string =>
	(labels as Record<string, string>)[tag] || tag;
export function languageChoices(
	query: string,
	selected: readonly string[]
): [string, string][] {
	const search = query.trim().toLowerCase();
	return Object.entries(labels)
		.filter(
			([tag, name]) =>
				!selected.includes(tag) &&
				(tag.includes(search) || name.toLowerCase().includes(search))
		)
		.sort((a, b) => a[1].localeCompare(b[1]));
}
export function addLanguage(
	selected: readonly string[],
	tag: string
): string[] {
	return selected.length >= 3 ||
		selected.includes(tag) ||
		!Object.hasOwn(labels, tag)
		? [...selected]
		: [...selected, tag];
}
export function removeLanguage(
	selected: readonly string[],
	tag: string
): string[] {
	return selected.length < 2
		? [...selected]
		: selected.filter((value) => value !== tag);
}
