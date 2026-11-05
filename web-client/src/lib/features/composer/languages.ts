import labels from './languages.json';

const choices = Object.entries(labels)
	.sort((a, b) => a[1].localeCompare(b[1]))
	.map(([tag, name]) => ({ tag, name, searchName: name.toLowerCase() }));

/** Keep the picker aligned with Flutter's v1 language catalogue. */
export const languageName = (tag: string): string =>
	(labels as Record<string, string>)[tag] || tag;
export function languageChoices(
	query: string,
	selected: readonly string[]
): [string, string][] {
	const search = query.trim().toLowerCase();
	return choices
		.filter(
			({ tag, searchName }) =>
				!selected.includes(tag) &&
				(tag.includes(search) || searchName.includes(search))
		)
		.map(({ tag, name }) => [tag, name]);
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
