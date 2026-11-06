import { label } from '#lib/public.ts';

const branded = new Set([
	'sewing',
	'knitting',
	'crochet',
	'quilting',
	'embroidery'
]);

export function identityCrafts(crafts?: readonly string[] | null) {
	const entries = new Map<string, string>();
	for (const craft of crafts ?? []) {
		const id = craft.split('#').at(-1)?.trim().toLowerCase() || '';
		if (id) entries.set(id, label(id));
	}
	return [...entries].map(([id, name]) => ({
		id,
		name,
		icon: branded.has(id)
	}));
}

export function fittingIdentityCrafts(
	width: number,
	nameWidth: number,
	entries: ReturnType<typeof identityCrafts>,
	countWidth: (count: number) => number
) {
	const budget = Math.min(
		width,
		Math.max(width - nameWidth, 8 + countWidth(entries.length))
	);
	let visible = entries.filter((entry) => entry.icon).length;
	while (visible > 0) {
		const hidden = entries.length - visible;
		if (
			8 +
				visible * 16 +
				(visible - 1) * 4 +
				(hidden ? 4 + countWidth(hidden) : 0) <=
			budget
		)
			break;
		visible--;
	}
	return visible;
}
