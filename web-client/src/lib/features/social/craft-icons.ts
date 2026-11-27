import { craftKey } from './craft-feeds';

export const craftIconIds = [
	'sewing',
	'knitting',
	'crochet',
	'quilting',
	'embroidery',
	'spinning',
	'dyeing',
	'punch-needle',
	'macrame',
	'weaving',
	'felting'
] as const;
const craftIcons = new Set<string>(craftIconIds);
export const craftBackgroundIds = craftIconIds.map((craft) => `craft-${craft}`);

export function craftIconPath(value: string) {
	const key = craftKey(value);
	return craftIcons.has(key) ? `/craft_icons/${key}.svg` : undefined;
}

export function craftIconMask(value: string) {
	const path = craftIconPath(value);
	return path ? `url('${path}')` : undefined;
}
