import catalog from './catalog.json';

export const craftKey = (value: string) =>
	value.split('#').at(-1)?.trim().toLowerCase() || '';
const crafts = catalog.filter((entry) => entry.family === 'craft');

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
