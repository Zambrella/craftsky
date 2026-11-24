import { label } from '#lib/public.ts';
import { craftKey } from './craft-feeds';
import { craftIconMask } from './craft-icons';

export interface IdentityCraft {
	id: string;
	name: string;
	icon?: string;
}

export function identityCrafts(
	crafts?: readonly string[] | null
): IdentityCraft[] {
	const entries = new Map<string, string>();
	for (const craft of crafts ?? []) {
		const id = craftKey(craft);
		if (id) entries.set(id, label(id));
	}
	return [...entries].map(([id, name]) => ({
		id,
		name,
		icon: craftIconMask(id)
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

export function identityCraftLabel(entries: readonly IdentityCraft[]) {
	return 'Crafts: ' + entries.map((entry) => entry.name).join(', ');
}
