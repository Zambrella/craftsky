import catalog from './catalog.json';
import {
	craftChoice,
	primaryCrafts,
	projectItemAliasDescription,
	projectItemChoices,
	projectSubtypesFor,
	projectTaxonomy,
	projectTypesFor
} from '../composer/project-options';
import type { ProjectFilters } from '#lib/api/client.ts';
import type { SelectOption } from '#lib/select-options.ts';
import {
	craftDetailOptions,
	craftFilterFamilies,
	type FilterSelections
} from './craft-detail-filters';

interface Option {
	id: string;
	family: string;
	parents: string[];
	labels: { en: string };
	aliases?: string[];
	craft?: string;
}

const entries = catalog as Option[];
export const filterFamilies = [
	['status', 'Status'],
	['projectItem', 'Project type'],
	['projectType', 'Project type'],
	['projectSubtype', 'Project subtype'],
	['patternDifficulty', 'Pattern difficulty'],
	...craftFilterFamilies
] as const;

export function sameFilterValue(id: string, value: string) {
	const left = id.toLowerCase();
	const right = value.trim().toLowerCase();
	return (
		left === right || (!right.includes('#') && left.split('#').at(-1) === right)
	);
}

function asOption(
	family: string,
	entry: {
		id: string;
		label: string;
		craft?: string | null;
		parent?: string;
		parents?: string[];
		aliases?: string[];
	}
): Option {
	return {
		id: entry.id,
		family,
		parents: entry.parents ?? (entry.parent ? [entry.parent] : []),
		labels: { en: entry.label },
		aliases: entry.aliases,
		craft: entry.craft ?? undefined
	};
}

export function filterOptions(family: string, craft = ''): Option[] {
	const choice = craftChoice(craft);
	if (craft && (!choice || choice.offered === false)) return [];
	if (family === 'status' || family === 'patternDifficulty')
		return entries.filter(
			(entry) =>
				entry.family === (family === 'status' ? 'status' : 'difficulty')
		);
	const craftOptions = craftDetailOptions(family, craft);
	if (craftOptions) return craftOptions.map((entry) => asOption(family, entry));
	if (family !== 'projectType' && family !== 'projectSubtype') return [];
	const crafts = choice ? [choice] : primaryCrafts;
	const byId = new Map<string, Option>();
	for (const { id } of crafts) {
		const types = projectTypesFor(id);
		const options = family === 'projectType' ? types : projectSubtypesFor(id);
		for (const option of options) byId.set(option.id, asOption(family, option));
	}
	return [...byId.values()];
}

function savedOption(
	family: string,
	craft: string,
	value: string
): Option | undefined {
	const options = filterOptions(family, craft);
	const current = options.find((option) => sameFilterValue(option.id, value));
	if (current) return current;

	// A named saved choice can outlive its picker entry. Preserve its label and
	// selection as long as the filter is still applicable to the selected craft.
	if (family !== 'projectSubtype' && family !== 'embroideryStyle') return;
	if (!options.length) return;
	const craftId = craftChoice(craft)?.id;
	const entry = projectTaxonomy.find(
		(option) =>
			option.family === (family === 'embroideryStyle' ? 'style' : family) &&
			(!craftId || option.craft === craftId) &&
			sameFilterValue(option.id, value)
	);
	return entry && asOption(family, entry);
}

export function filterValueLabel(family: string, craft: string, value: string) {
	return savedOption(family, craft, value)?.labels.en;
}

const catalogueById = new Map(
	projectTaxonomy.map((choice) => [choice.id.toLowerCase(), choice])
);

export function matchesNamedProjectItem(actual: unknown, selection: string) {
	if (typeof actual !== 'string') return false;
	const seen = new Set<string>();
	function includes(id: string): boolean {
		const key = id.toLowerCase();
		const option = catalogueById.get(key);
		if (!option || seen.has(key)) return false;
		seen.add(key);
		return (
			sameFilterValue(key, selection) ||
			(option.parents?.some(includes) ?? false)
		);
	}
	return includes(actual);
}

function savedProjectItem(craft: string, value: string) {
	return (
		savedOption('projectType', craft, value) ??
		savedOption('projectSubtype', craft, value)
	);
}

export function selectedProjectItems(
	craft: string,
	selected: FilterSelections
) {
	if (selected.projectItem?.length) return selected.projectItem;
	const family = selected.projectSubtype?.length
		? 'projectSubtype'
		: 'projectType';
	return [
		...new Set(
			(selected[family] || []).flatMap((value) => {
				const option = savedOption(family, craft, value);
				return option ? [option.id] : [];
			})
		)
	];
}

export function projectItemFilterChoices(
	craft: string,
	values: readonly string[]
): SelectOption[] {
	const options = projectItemChoices(craft);
	for (const value of values) {
		if (options.some(({ id }) => id === value)) continue;
		const item = savedProjectItem(craft, value);
		if (!item) continue;
		const parentId = item.parents.find((id) =>
			options.some((option) => option.id === id)
		);
		const parent = options.findIndex(({ id }) => id === parentId);
		const label = item.labels.en;
		options.splice(parent + 1, 0, {
			id: item.id,
			label,
			parentId,
			selectionLabel:
				parent >= 0 ? `${options[parent].label} · ${label}` : undefined,
			aliases: item.aliases,
			description: projectItemAliasDescription(label, item.aliases)
		});
	}
	return options;
}

/** Keep applicable saved choices and ignore retired filter families in client URLs. */
export function browseFilters(
	params: Pick<URLSearchParams, 'get' | 'getAll'>,
	craft = params.get('craft') || ''
): ProjectFilters {
	const out: ProjectFilters = {};
	for (const [family] of filterFamilies) {
		const values = [
			...new Set(
				params.getAll(family).flatMap((value) => {
					if (family === 'projectItem') {
						const item = savedProjectItem(craft, value);
						return item ? [item.id] : [];
					}
					return savedOption(family, craft, value) ? [value] : [];
				})
			)
		];
		if (values.length) out[family] = values;
	}
	if (params.get('selfDrafted') === 'true') out.selfDrafted = true;
	return out;
}
