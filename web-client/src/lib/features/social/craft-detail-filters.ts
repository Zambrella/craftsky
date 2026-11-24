import {
	projectDetailFilterKeys,
	type ProjectDetailFilterKey
} from '#lib/api/project-detail-filter-keys.ts';
import type { JsonObject, ProjectFilters } from '#lib/api/types.ts';
import { effectiveSizes, record } from '../composer/model';
import {
	craftChoice,
	projectDetailOptionsFor,
	projectMethodValues,
	projectUnits
} from '../composer/project-options';
import toolSizes from '../composer/tool-sizes.json';

type Key = ProjectDetailFilterKey | 'knittingMethod' | 'feltingMethod';
type Source = string | readonly string[];
type Definition = {
	label: string;
	crafts: readonly string[];
	options: Source;
	path: readonly string[];
};
const textiles = ['knitting', 'crochet'];

const filter = (
	label: string,
	craft: string | readonly string[],
	source: Source,
	...path: string[]
): Definition => ({
	label,
	crafts: typeof craft === 'string' ? [craft] : craft,
	options: source,
	path
});

const definitions: Record<Key, Definition> = {
	knittingMethod: filter('Knitting method', 'knitting', 'method', 'methods[]'),
	crochetMethod: filter('Crochet method', 'crochet', 'method', 'methods[]'),
	feltingMethod: filter('Felting method', 'felting', 'method', 'method'),
	technique: filter(
		'Techniques',
		[...textiles, 'sewing'],
		'technique',
		'techniques[]',
		'technique'
	),
	techniqueSubtype: filter(
		'Technique subtypes',
		'knitting',
		'techniqueSubtype',
		'techniques[]',
		'subtypes[]',
		'subtype'
	),
	embroideryMethod: filter(
		'Embroidery method',
		'embroidery',
		'method',
		'methods[]',
		'method'
	),
	embroideryStyle: filter(
		'Embroidery style',
		'embroidery',
		'style',
		'methods[]',
		'styles[]',
		'style'
	),
	spinningTool: filter('Spinning tool', 'spinning', 'tool', 'tool'),
	fibrePreparation: filter(
		'Fibre preparation',
		'spinning',
		'fibrePreparation',
		'fibrePreparation'
	),
	draftingTechnique: filter(
		'Drafting techniques',
		'spinning',
		'draftingTechnique',
		'draftingTechniques[]',
		'technique'
	),
	plyingTechnique: filter(
		'Plying techniques',
		'spinning',
		'plyingTechnique',
		'plyingTechniques[]',
		'technique'
	),
	finishedYarnMassUnit: filter(
		'Finished yarn mass unit',
		'spinning',
		projectUnits.mass,
		'finishedYarn',
		'mass',
		'unit'
	),
	finishedYarnLengthUnit: filter(
		'Finished yarn length unit',
		'spinning',
		projectUnits.length,
		'finishedYarn',
		'length',
		'unit'
	),
	materialForm: filter(
		'Material form',
		'dyeing',
		'materialForm',
		'materialForm'
	),
	dyeType: filter('Dye type', 'dyeing', 'dyeType', 'dyeType'),
	applicationMethod: filter(
		'Application method',
		'dyeing',
		'applicationMethod',
		'applicationMethod'
	),
	colorEffect: filter('Colour effect', 'dyeing', 'colorEffect', 'colorEffect'),
	loopHeightUnit: filter(
		'Loop height unit',
		'punch-needle',
		projectUnits.length,
		'loopHeight',
		'unit'
	),
	cordConstruction: filter(
		'Cord construction',
		'macrame',
		'construction',
		'cords[]',
		'construction'
	),
	knot: filter('Knots', 'macrame', 'knot', 'knots[]', 'knot'),
	loomType: filter('Loom type', 'weaving', 'loomType', 'loomType'),
	weaveStructure: filter(
		'Weave structure',
		'weaving',
		'structure',
		'structure'
	),
	materialRole: filter(
		'Material role',
		'weaving',
		'materialRole',
		'materials[]',
		'role'
	),
	settUnit: filter('Sett unit', 'weaving', projectUnits.sett, 'sett', 'unit'),
	feltingForm: filter('Project form', 'felting', 'form', 'form'),
	needleGauge: filter(
		'Needle gauges',
		'felting',
		'needleGauge',
		'needleGauges[]',
		'gauge'
	),
	needleSizeMm: filter('Needle sizes', 'knitting', toolSizes.needle),
	hookSizeMm: filter('Hook sizes', 'crochet', toolSizes.hook),
	gaugeUnit: filter('Gauge unit', textiles, projectUnits.gauge, 'gauge', 'unit')
};

const retainedFilters = [
	'knittingMethod',
	'crochetMethod',
	'embroideryMethod',
	'embroideryStyle',
	'feltingMethod',
	'loomType'
] as const;

export const craftFilterFamilies = retainedFilters.map(
	(key) => [key, definitions[key].label] as const
);
const detailKeys = new Set<string>([
	...projectDetailFilterKeys,
	'knittingMethod',
	'feltingMethod'
]);
export const isNewCraftFilter = (key: string): key is Key =>
	detailKeys.has(key);

export type FilterSelections = Record<string, string[]>;

function craftFilterAvailable(key: Key, craft: string) {
	const choice = craftChoice(craft);
	return (
		!!choice &&
		choice.offered !== false &&
		definitions[key].crafts.some((id) => craftChoice(id)?.id === choice.id)
	);
}

export function craftDetailOptions(key: string, craft: string) {
	const active = retainedFilters.find((value) => value === key);
	return active
		? craftFilterAvailable(active, craft)
			? projectDetailOptionsFor(active)
			: []
		: undefined;
}

function valuesAt(source: unknown, path: readonly string[]) {
	let values: unknown[] = [source];
	for (const field of path) {
		if (field.endsWith('[]'))
			values = values.flatMap((entry) => {
				const children = record(entry)[field.slice(0, -2)];
				return Array.isArray(children) ? children : [];
			});
		else values = values.map((entry) => record(entry)[field]);
	}
	return values;
}

/** Keep locally confirmed posts under the same structured constraints as indexed results. */
export function matchesCraftDetailFilters(
	details: JsonObject | undefined,
	craft: string,
	filters: ProjectFilters
) {
	const active = Object.keys(filters).filter(isNewCraftFilter);
	if (!active.length) return true;
	const choice = craftChoice(craft);
	if (!details || !choice || details.$type !== choice.detailsType) return false;
	const namespace = choice.detailsType?.split('#')[0]?.toLowerCase() ?? '';
	const prefix = ['knitting', 'crochet', 'sewing', 'embroidery'].includes(
		choice.profileId ?? ''
	)
		? `${namespace}.defs#`
		: `${namespace}#`;
	const matches = (values: unknown[], key: Key) => {
		const raw = filters[key];
		const selected = Array.isArray(raw) ? raw : [raw];
		const taxonomyToken =
			typeof definitions[key].options === 'string' && key !== 'needleGauge';
		const normalized = (input: string) => {
			const value = input.toLowerCase();
			return taxonomyToken && value.startsWith(prefix)
				? value.slice(prefix.length)
				: value;
		};
		return values.some(
			(candidate) =>
				typeof candidate === 'string' &&
				selected.some(
					(value) =>
						typeof value === 'string' &&
						normalized(candidate) === normalized(value.trim())
				)
		);
	};
	for (const [parent, child] of [
		['technique', 'techniqueSubtype'],
		['embroideryMethod', 'embroideryStyle']
	] as const) {
		if (!filters[parent] || !filters[child]) continue;
		if (
			parent === 'embroideryMethod' &&
			(Object.hasOwn(details, 'method') || Object.hasOwn(details, 'style'))
		)
			continue;
		const [group, ...parentPath] = definitions[parent].path;
		const childPath = definitions[child].path.slice(1);
		if (
			!valuesAt(details, [group]).some(
				(entry) =>
					matches(valuesAt(entry, parentPath), parent) &&
					matches(valuesAt(entry, childPath), child)
			)
		)
			return false;
	}
	return active.every((key) => {
		if (!craftFilterAvailable(key, craft)) return false;
		if (key === 'needleSizeMm' || key === 'hookSizeMm') {
			const tool = key === 'needleSizeMm' ? 'needle' : 'hook';
			const pluralKey = `${tool}SizesMm`;
			return (
				(!Object.hasOwn(details, pluralKey) ||
					Array.isArray(details[pluralKey])) &&
				matches(effectiveSizes(details, tool), key)
			);
		}
		if (
			key === 'knittingMethod' ||
			key === 'crochetMethod' ||
			key === 'feltingMethod'
		)
			return matches(projectMethodValues(craft, details), key);
		if (key === 'embroideryMethod' && Object.hasOwn(details, 'method'))
			return matches([details.method], key);
		if (key === 'embroideryStyle' && Object.hasOwn(details, 'style'))
			return matches([details.style], key);
		return matches(valuesAt(details, definitions[key].path), key);
	});
}
