import catalog from './catalog.json';
import type { ProjectFilters } from '#lib/api/client.ts';
interface Option {
	id: string;
	family: string;
	parents: string[];
	labels: { en: string };
	craft?: string;
}
const entries = catalog as Option[];
export const filterFamilies = [
	['status', 'Status'],
	['projectType', 'Project type'],
	['projectSubtype', 'Project subtype'],
	['patternDifficulty', 'Pattern difficulty'],
	['color', 'Colour'],
	['designTag', 'Design detail'],
	['yarnWeight', 'Yarn weight'],
	['piecingTechnique', 'Piecing technique'],
	['quiltingMethod', 'Quilting method']
] as const;
export function filterOptions(
	family: string,
	craft = '',
	projectType: string | readonly string[] = ''
): Option[] {
	const projectTypes =
		typeof projectType === 'string'
			? projectType
				? [projectType]
				: []
			: projectType;
	const selected =
		craft && !craft.includes('#')
			? 'social.craftsky.feed.defs#' + craft
			: craft;
	if (
		selected &&
		family === 'yarnWeight' &&
		![
			'social.craftsky.feed.defs#knitting',
			'social.craftsky.feed.defs#crochet'
		].includes(selected)
	)
		return [];
	if (
		selected &&
		['piecingTechnique', 'quiltingMethod'].includes(family) &&
		selected !== 'social.craftsky.feed.defs#quilting'
	)
		return [];
	return entries.filter(
		(entry) =>
			entry.family ===
				(family === 'patternDifficulty' ? 'difficulty' : family) &&
			(!selected || !entry.craft || entry.craft === selected) &&
			(!projectTypes.length ||
				family !== 'projectSubtype' ||
				entry.parents.some((parent) => projectTypes.includes(parent)))
	);
}
/** Empty HTML select values are not valid AppView filters. Keep repeated values. */
export function browseFilters(
	params: Pick<URLSearchParams, 'get' | 'getAll'>
): ProjectFilters {
	const out: ProjectFilters = {};
	for (const [family] of filterFamilies) {
		const values = [
			...new Set(params.getAll(family).filter((value) => value.trim()))
		];
		if (values.length) out[family] = values;
	}
	if (params.get('selfDrafted') === 'true') out.selfDrafted = true;
	return out;
}
