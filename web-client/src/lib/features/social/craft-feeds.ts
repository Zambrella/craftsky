import { craftChoice, primaryCrafts } from '../composer/project-options';

export const craftKey = (value: string) =>
	craftChoice(value.trim())?.profileId ??
	value.split('#').at(-1)?.trim().toLowerCase() ??
	'';
const crafts = primaryCrafts;

/** Native Home puts preferred crafts first, then every other supported craft. */
export function craftFeeds(
	preferences: readonly string[] = [],
	preferredOnly = false
) {
	const preferred = [...new Set(preferences.map(craftKey))].flatMap((key) =>
		crafts.filter((entry) => craftKey(entry.id) === key)
	);
	return preferredOnly
		? preferred
		: [...preferred, ...crafts.filter((entry) => !preferred.includes(entry))];
}
