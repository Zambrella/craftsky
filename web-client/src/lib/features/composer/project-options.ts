import catalogue from './catalogue.json';
import taxonomy from './project-taxonomy.json';
import itemOptions from './project-item-options.json';
import type { SelectOption } from '#lib/select-options.ts';

interface ProjectOption {
	id: string;
	family: string;
	label: string;
	craft?: string | null;
	parents?: string[];
	parent?: string;
	aliases?: string[];
	group?: string;
	crafts?: string[];
	profileId?: string;
	detailsType?: string;
	phase?: number;
	offered?: boolean;
}

export const projectUnits = {
	gauge: ['cm', 'in'],
	length: ['mm', 'cm', 'm', 'in', 'ft', 'yd'],
	mass: ['g', 'kg', 'oz', 'lb'],
	sett: ['ends-per-inch', 'ends-per-centimetre']
} as const;
export const unitOption = (unit: string) => ({
	id: unit,
	label: unit.replaceAll('-', ' ')
});

export const projectTaxonomy: ProjectOption[] = taxonomy;
const entries: ProjectOption[] = [
	...projectTaxonomy,
	...catalogue.filter(
		(item) =>
			!projectTaxonomy.some(
				({ id, family }) => id === item.id && family === item.family
			)
	)
];
const entriesByID = new Map(entries.map((entry) => [entry.id, entry]));
const itemsByCraft: Record<string, { type: string; subtypes: string[] }[]> =
	itemOptions.projectItems;
const detailChoices: Record<string, string[]> =
	itemOptions.projectDetailOptions;
export const otherProjectType = 'social.craftsky.project.defs#other';
export const projectOptionFor = (value: string) => entriesByID.get(value);

function compareOptionLabels(left: string, right: string): number {
	const a = left.toLowerCase();
	const b = right.toLowerCase();
	return a < b ? -1 : a > b ? 1 : 0;
}

export function sortedProjectOptions<T extends { id: string; label: string }>(
	choices: readonly T[]
): T[] {
	const isOther = (id: string) => {
		const value = id.trim().toLowerCase();
		return value === otherProjectType || value === 'other';
	};
	return [...choices].sort(
		(a, b) =>
			Number(isOther(a.id)) - Number(isOther(b.id)) ||
			compareOptionLabels(a.label, b.label)
	);
}

export function projectItemAliasDescription(
	label: string,
	aliases: readonly string[] = []
): string | undefined {
	const key = (name: string) => name.trim().toLowerCase();
	const seen = new Set([
		key(label),
		...label.split(/\s+(?:or|and)\s+|\//i).map(key)
	]);
	const names: string[] = [];
	for (const alias of aliases) {
		for (const part of alias.split(/\s+or\s+|\//i)) {
			const name = part.trim();
			const normalized = key(name);
			if (!normalized || seen.has(normalized)) continue;
			seen.add(normalized);
			names.push(name[0].toUpperCase() + name.slice(1));
		}
	}
	return names.length ? names.sort(compareOptionLabels).join(' · ') : undefined;
}

export function projectItemConfig(craft: string) {
	return craft
		? (itemsByCraft[craftChoice(craft)?.id ?? craft] ?? [])
		: Object.values(itemsByCraft).flat();
}
function catalogueChoices(ids: string[]): ProjectOption[] {
	return [...new Set(ids)].flatMap((id) => {
		const entry = projectOptionFor(id);
		return entry ? [entry] : [];
	});
}
export const projectTypesFor = (craft: string) =>
	catalogueChoices(projectItemConfig(craft).map((item) => item.type));
export function projectSubtypesFor(
	craft: string,
	type: string | readonly string[] = ''
) {
	const selected = typeof type === 'string' ? (type ? [type] : []) : type;
	return catalogueChoices(
		projectItemConfig(craft)
			.filter((item) => !selected.length || selected.includes(item.type))
			.flatMap((item) => item.subtypes)
	);
}
export function projectItemChoices(craft: string): SelectOption[] {
	const choice = craftChoice(craft);
	if (craft && (!choice || choice.offered === false)) return [];
	const all = (choice ? [choice] : primaryCrafts).flatMap(({ id }) =>
		projectItemConfig(id)
	);
	const types = sortedProjectOptions(
		catalogueChoices(all.map(({ type }) => type))
	);
	const option = ({ id, label, aliases }: ProjectOption): SelectOption => ({
		id,
		label,
		aliases,
		description: projectItemAliasDescription(label, aliases)
	});
	return types.flatMap((type) => [
		option(type),
		...sortedProjectOptions(
			catalogueChoices(
				all
					.filter((item) => item.type === type.id)
					.flatMap(({ subtypes }) => subtypes)
			)
		).map((subtype) => ({
			...option(subtype),
			parentId: type.id,
			selectionLabel: `${type.label} · ${subtype.label}`
		}))
	]);
}
export const projectDetailOptionsFor = (field: string) =>
	catalogueChoices(detailChoices[field] ?? []);

export function projectMethodValues(
	craft: string,
	detail: Record<string, unknown>
): string[] {
	if (Array.isArray(detail.methods))
		return detail.methods.filter(
			(value): value is string => typeof value === 'string'
		);
	const kind = craftChoice(craft)?.profileId;
	const matchesLegacy = (value: unknown, id: string) =>
		typeof value === 'string' &&
		(value.toLowerCase() === id.toLowerCase() ||
			value.toLowerCase() === id.split('#').at(-1)?.toLowerCase());
	if (kind === 'felting' && typeof detail.method === 'string') {
		const choice = projectDetailOptionsFor('feltingMethod').find(({ id }) =>
			matchesLegacy(detail.method, id)
		);
		return [choice?.id ?? detail.method];
	}
	if (kind === 'crochet') {
		const tunisian = 'social.craftsky.project.crochet.defs#tunisianCrochet';
		if (
			Array.isArray(detail.techniques) &&
			detail.techniques.some((entry) =>
				matchesLegacy(entry?.technique, tunisian)
			)
		)
			return [tunisian];
	}
	return [];
}
const knownCrafts = projectTaxonomy.filter(({ family }) => family === 'craft');
export const primaryCrafts = knownCrafts.filter(
	({ offered }) => offered !== false
);
export const craftChoice = (value: string) =>
	knownCrafts.find(
		({ id, profileId }) =>
			profileId === value ||
			id === value ||
			id.split('#').at(-1)?.toLowerCase() === value.toLowerCase()
	);
export const options = (family: string, craft?: string, parent?: string) =>
	entries.filter(
		(entry) =>
			entry.family === family &&
			(family !== 'craft' || entry.offered !== false) &&
			(!entry.craft || !craft || entry.craft === craft) &&
			(!entry.crafts || !craft || entry.crafts.includes(craft)) &&
			(!parent ||
				(entry.parent
					? parent === entry.parent
					: !entry.parents?.length || entry.parents.includes(parent)))
	);
export const label = (value: string) =>
	projectOptionFor(value)?.label ||
	craftChoice(value)?.label ||
	(value.startsWith('social.craftsky.') ? value.split('#').at(-1) : value) ||
	value;
