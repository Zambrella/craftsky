import { acceptedHashtags, hashtagsInText } from './facet-input';
import { UserFacingError } from '#lib/user-facing-error.ts';
import { graphemeCount as textLength } from '#lib/text.ts';
import {
	newOperationKey,
	type Facet,
	type JsonObject,
	type PostEditChanges,
	type PostPayload,
	type Post,
	type Project
} from '#lib/api/client.ts';
import type { Draft, PendingWrite } from '../drafts/store.ts';
import { craftChoice } from './project-options';
export { label, options } from './project-options';
export const defaultProject = (crafts: readonly string[] = []): JsonObject => {
	const craft = crafts
		.map(craftChoice)
		.find((choice) => choice && choice.offered !== false)?.id;
	return {
		common: {
			...(craft ? { craftType: craft } : {}),
			status: 'social.craftsky.feed.defs#finished'
		}
	};
};
export function record(value: unknown): JsonObject {
	return value && typeof value === 'object' && !Array.isArray(value)
		? (value as JsonObject)
		: {};
}
export function str(value: unknown): string {
	return typeof value === 'string' ? value : '';
}
export function effectivePatterns<T>(patterns?: T[], legacy?: T): T[] {
	return patterns ?? (legacy ? [legacy] : []);
}
export function patternRows(common: JsonObject): JsonObject[] {
	return effectivePatterns(
		Array.isArray(common.patterns) ? common.patterns.map(record) : undefined,
		common.pattern ? record(common.pattern) : undefined
	);
}
export function projectPatternLabels(craft: string) {
	const kind = craftChoice(craft)?.profileId;
	const label =
		kind === 'weaving'
			? 'Pattern or draft'
			: kind === 'felting'
				? 'Pattern or template'
				: 'Pattern';
	return {
		title: label === 'Pattern' ? 'Pattern details' : label,
		name: `${label} name`,
		description: `Add ${label.toLowerCase()} names, credits, links and difficulty`
	};
}
function retainedRow(original: JsonObject[], row: JsonObject) {
	return typeof row.existingIndex === 'number'
		? original.find((entry) => entry.existingIndex === row.existingIndex)
		: undefined;
}
export function effectiveSizes(
	detail: Record<string, unknown>,
	tool: 'needle' | 'hook'
): string[] {
	const sizes = detail[`${tool}SizesMm`];
	const legacy = str(detail[`${tool}SizeMm`]);
	return Array.isArray(sizes)
		? sizes.filter((size): size is string => typeof size === 'string')
		: legacy
			? [legacy]
			: [];
}
export function projectWrite(project: JsonObject): JsonObject {
	const common = { ...record(project.common) };
	if (common.patterns !== undefined || common.pattern) {
		common.patterns = patternRows(common)
			.filter(
				(pattern) =>
					typeof pattern.existingIndex === 'number' ||
					hasPatternContent(pattern)
			)
			.map((object) => {
				const pattern = { ...object };
				delete pattern.existingIndex;
				return pattern;
			});
		delete common.pattern;
	}
	if (!project.details) return { ...project, common };
	const details = { ...record(project.details) };
	if (typeof details.projectSubtype === 'string') {
		const subtype = details.projectSubtype.trim();
		if (subtype) details.projectSubtype = subtype;
		else delete details.projectSubtype;
	}
	const tool = str(details.$type).endsWith('.knitting#details')
		? 'needle'
		: str(details.$type).endsWith('.crochet#details')
			? 'hook'
			: undefined;
	if (
		tool &&
		(details[`${tool}SizesMm`] !== undefined || details[`${tool}SizeMm`])
	) {
		details[`${tool}SizesMm`] = effectiveSizes(details, tool);
		delete details[`${tool}SizeMm`];
	}
	return { ...project, common, details };
}
export const patternFields = ['name', 'designer', 'publisher'] as const;
export const hasPatternContent = (pattern: JsonObject) =>
	Object.entries(pattern).some(
		([key, value]) =>
			key !== 'existingIndex' &&
			(typeof value === 'string'
				? !!value.trim()
				: value === true ||
					(Array.isArray(value)
						? value.length > 0
						: value != null && value !== false))
	);
/** Resolve a carousel selection against fresh target assets, never an arbitrary query CID. */
export function selectedQuoteImage(
	post: Pick<Post, 'images'>,
	cid: string | null | undefined
): string | undefined {
	return cid && post.images?.some((image) => image.cid === cid)
		? cid
		: undefined;
}
export function responseLabel(
	draft: Pick<Draft, 'reply'>
): 'Comment' | 'Reply' {
	return draft.reply?.parent.uri === draft.reply?.root.uri
		? 'Comment'
		: 'Reply';
}
export function scheduledLocalDate(value: string | undefined): string {
	if (!value) return '';
	const date = new Date(value);
	return Number.isFinite(date.getTime())
		? new Date(date.getTime() - date.getTimezoneOffset() * 60000)
				.toISOString()
				.slice(0, 16)
		: '';
}
export function scheduledUTCDate(value: string): string | undefined {
	const date = new Date(value);
	return value && Number.isFinite(date.getTime())
		? date.toISOString()
		: undefined;
}
/** Flutter ScheduleTimeWindow: whole minutes, five minutes through 28 days. */
export function scheduleWindow(now = Date.now()): {
	earliest: number;
	latest: number;
} {
	return {
		earliest: Math.ceil((now + 5 * 60_000) / 60_000) * 60_000,
		latest: Math.floor((now + 28 * 86_400_000) / 60_000) * 60_000
	};
}
export { graphemeCount as textLength } from '#lib/text.ts';
export function validateText(
	text: string,
	max: number,
	bytes = max * 10
): boolean {
	return (
		textLength(text) <= max && new TextEncoder().encode(text).length <= bytes
	);
}
export function validateDraft(draft: Draft, scheduling = false): string[] {
	const errors: string[] = [];
	if (
		draft.instagramImport &&
		(scheduling || draft.scheduleId || draft.scheduleLater)
	)
		errors.push('Imported Instagram posts cannot be scheduled.');
	if (draft.instagramImport && (draft.edit || draft.reply || draft.quote))
		errors.push('An Instagram import must create a new post or project.');
	if (!draft.text.trim()) errors.push('Add some text to your post.');
	if (!validateText(draft.text, 2000, 20000))
		errors.push('Posts can contain up to 2,000 characters.');
	if (draft.images.length > 8) errors.push('Choose up to eight images.');
	if (draft.images.some((image) => !validateText(image.alt, 1000, 10000)))
		errors.push('Image descriptions can contain up to 1,000 characters.');
	if (draft.video && !validateText(draft.video.alt, 1000, 10000))
		errors.push('Video descriptions can contain up to 1,000 characters.');
	if (
		!draft.langs.length ||
		draft.langs.length > 3 ||
		new Set(draft.langs.map((lang) => lang.toLowerCase())).size !==
			draft.langs.length ||
		draft.langs.some((lang) => !validLanguage(lang))
	)
		errors.push('Choose one to three distinct valid post languages.');
	if (draft.reply && draft.sponsored)
		errors.push('Replies cannot be sponsored.');
	if (draft.external) {
		if (
			!draft.external.title.trim() ||
			!validateText(draft.external.title, 200, 2000) ||
			!validateText(draft.external.description, 300, 3000)
		)
			errors.push(
				'Link previews need a title up to 200 characters and a description up to 300 characters.'
			);
		try {
			const url = new URL(draft.external.uri);
			if (
				!['https:', 'http:'].includes(url.protocol) ||
				url.username ||
				url.password
			)
				throw 0;
		} catch {
			errors.push('Use an HTTP or HTTPS link preview URL.');
		}
	}
	if ((draft.videoRecovery || 0) > 1)
		errors.push(
			'The video service could not restore this video. Remove it and select the file again.'
		);
	if (
		draft.video &&
		(draft.images.length || draft.reply || draft.quote || draft.external)
	)
		errors.push(
			'Videos cannot be combined with images, replies, quotes or link previews.'
		);
	if (
		draft.quote &&
		(draft.reply || draft.kind === 'project' || draft.external)
	)
		errors.push(
			'Quotes cannot be combined with projects, replies or link previews.'
		);
	if (scheduling && (draft.reply || draft.quote || draft.video || draft.edit))
		errors.push(
			'Only new posts and projects with images or links can be scheduled.'
		);
	if (draft.kind === 'project') {
		if (draft.external)
			errors.push(
				'Project posts cannot contain link previews. Remove the preview or use a standard post.'
			);
		const common = record(draft.project?.common);
		const detail = record(draft.project?.details);
		const patterns = patternRows(common);
		const patternName = projectPatternLabels(str(common.craftType)).name;
		if (patterns.length > 5) errors.push('Use up to five patterns.');
		const materials = Array.isArray(common.materials) ? common.materials : [];
		if (
			materials.length > 10 ||
			materials.some(
				(value) => !validateText(str(record(value).text), 100, 1000)
			)
		)
			errors.push('Use up to ten materials, with up to 100 characters each.');
		if (!common.craftType) errors.push('Choose a primary craft.');
		if (!str(common.status).trim()) errors.push('Choose a project status.');
		if (!validateText(str(detail.projectSubtype).trim(), 100, 100))
			errors.push('Project subtype is too long.');
		for (const [value, max, bytes, name] of [
			[common.title, 200, 500, 'Project title'],
			[common.duration, 100, 500, 'Duration'],
			...patterns.flatMap((pattern) =>
				patternFields.map(
					(field) =>
						[
							pattern[field],
							200,
							500,
							{
								name: patternName,
								designer: 'Designer',
								publisher: 'Publisher'
							}[field]
						] as const
				)
			),
			[detail.fitNotes, 600, 6000, 'Fit notes']
		] as const)
			if (typeof value === 'string' && !validateText(value, max, bytes))
				errors.push(`${name} is too long.`);
		const gauge = record(detail.gauge);
		if (
			Object.keys(gauge).length &&
			(!positive(gauge.stitches) ||
				!positive(gauge.measurement) ||
				!gauge.unit ||
				(gauge.rows !== undefined && !positive(gauge.rows)))
		)
			errors.push(
				'Gauge needs positive stitches, measurement and a unit. Rows are optional.'
			);
		errors.push(
			...validateCraftDetails(
				detail,
				record(record(draft.edit?.editable.project).details)
			)
		);
	}
	return errors;
}
function validateCraftDetails(
	detail: JsonObject,
	previous: JsonObject
): string[] {
	const errors: string[] = [];
	const yarn = record(detail.finishedYarn);
	const priorYarn = record(previous.finishedYarn);
	const kind = str(detail.$type);
	const changed = (value: unknown, before: unknown) =>
		JSON.stringify(value) !== JSON.stringify(before);
	const spinning = kind === 'social.craftsky.project.spinning#details';
	const measurements = [
		...(spinning
			? [
					[yarn.mass, priorYarn.mass, 'Finished yarn mass'],
					[yarn.length, priorYarn.length, 'Finished yarn length']
				]
			: []),
		...(kind === 'social.craftsky.project.punchNeedle#details'
			? [[detail.loopHeight, previous.loopHeight, 'Loop height']]
			: []),
		...(kind === 'social.craftsky.project.weaving#details'
			? [[detail.sett, previous.sett, 'Sett']]
			: [])
	];
	for (const [value, before, name] of measurements) {
		const measurement = record(value);
		if (
			value &&
			changed(value, before) &&
			(!decimal(measurement.amount) || !str(measurement.unit))
		)
			errors.push(`${name} needs a positive decimal amount and a unit.`);
	}
	if (spinning) {
		for (const [value, before, name] of [
			[detail.plyCount, previous.plyCount, 'Ply count'],
			[yarn.wrapsPerInch, priorYarn.wrapsPerInch, 'Wraps per inch']
		] as const) {
			if (value !== undefined && changed(value, before) && !positive(value))
				errors.push(`${name} must be a positive whole number.`);
		}
	}
	if (
		kind === 'social.craftsky.project.knitting#details' &&
		detail.machineGaugeMm !== undefined &&
		changed(detail.machineGaugeMm, previous.machineGaugeMm) &&
		!decimal(detail.machineGaugeMm)
	)
		errors.push('Machine needle spacing needs a positive decimal amount.');
	if (
		kind === 'social.craftsky.project.macrame#details' &&
		Array.isArray(detail.cords)
	) {
		const original =
			previous.$type === kind && Array.isArray(previous.cords)
				? previous.cords.map(record)
				: [];
		if (
			detail.cords
				.map(record)
				.some(
					(cord) =>
						cord.diameterMm !== undefined &&
						cord.diameterMm !== retainedRow(original, cord)?.diameterMm &&
						!decimal(cord.diameterMm)
				)
		)
			errors.push('Cord diameter needs a positive decimal amount.');
	}
	if (
		kind === 'social.craftsky.project.weaving#details' &&
		changed(detail.materials, previous.materials) &&
		Array.isArray(detail.materials) &&
		detail.materials.some(
			(row) => !str(record(row).role) || !str(record(row).text).trim()
		)
	)
		errors.push('Each weaving material needs a name and a role.');
	return errors;
}

function positive(value: unknown) {
	return typeof value === 'number' && Number.isInteger(value) && value > 0;
}
function decimal(value: unknown) {
	return (
		typeof value === 'string' &&
		/^\d+(?:\.\d+)?$/.test(value) &&
		Number(value) > 0
	);
}
export function validLanguage(value: string): boolean {
	try {
		return (
			new Intl.Locale(value).baseName === value || !!new Intl.Locale(value)
		);
	} catch {
		return false;
	}
}

/** UTF-8 byte offsets required by AT Protocol, including text before emoji. */
export async function buildFacets(
	text: string,
	resolve: (handle: string) => Promise<string>,
	precedingHashtags: string[] = []
): Promise<Facet[]> {
	const facets: Facet[] = [];
	const accepted = acceptedHashtags([
		...precedingHashtags,
		...hashtagsInText(text)
	]);
	const encoder = new TextEncoder();
	const re =
		/(https?:\/\/[^\s<>]+)|(?:^|[\s(])(@[a-zA-Z0-9][a-zA-Z0-9.-]*\.[a-zA-Z]{2,})|(?:^|[\s(])(#[\p{L}\p{N}_]+)/gu;
	for (const match of text.matchAll(re)) {
		const token = match[1] || match[2] || match[3];
		const start = match.index! + match[0].indexOf(token);
		let feature: JsonObject;
		if (match[1])
			feature = { $type: 'app.bsky.richtext.facet#link', uri: token };
		else if (match[2]) {
			let did: string;
			try {
				did = await resolve(token.slice(1));
			} catch {
				continue;
			}
			feature = { $type: 'app.bsky.richtext.facet#mention', did };
		} else {
			const tag = token.slice(1);
			if (!validateText(tag, 64, 640) || !accepted.has(tag.toLowerCase()))
				continue;
			feature = { $type: 'app.bsky.richtext.facet#tag', tag };
		}
		facets.push({
			index: {
				byteStart: encoder.encode(text.slice(0, start)).length,
				byteEnd: encoder.encode(text.slice(0, start + token.length)).length
			},
			features: [feature]
		});
	}
	return facets;
}

/** Count only tag features on the byte ranges AppView can index. */
export function facetHashtags(text: string, facets: unknown): string[] {
	if (!Array.isArray(facets)) return [];
	const bytes = new TextEncoder().encode(text).length;
	return facets.flatMap((value) => {
		const facet = record(value);
		const { byteStart, byteEnd } = record(facet.index);
		if (
			typeof byteStart !== 'number' ||
			!Number.isInteger(byteStart) ||
			typeof byteEnd !== 'number' ||
			!Number.isInteger(byteEnd) ||
			byteStart < 0 ||
			byteEnd <= byteStart ||
			byteEnd > bytes ||
			!Array.isArray(facet.features)
		)
			return [];
		return facet.features.flatMap((value) => {
			const feature = record(value);
			return feature.$type === 'app.bsky.richtext.facet#tag' &&
				typeof feature.tag === 'string'
				? [feature.tag]
				: [];
		});
	});
}

/** Regenerate annotations only for new or edited project text leaves. */
export async function projectFacets(
	project: JsonObject,
	before: JsonObject | undefined,
	resolve: (handle: string) => Promise<string>,
	precedingHashtags: string[] = []
): Promise<JsonObject> {
	const next = JSON.parse(JSON.stringify(project)) as JsonObject;
	const common = record(next.common);
	const previous = record(before?.common);
	const patterns = patternRows(common);
	const oldPatterns = patternRows(previous);
	const preservePatterns =
		!!before && JSON.stringify(patterns) === JSON.stringify(oldPatterns);
	const prefix = [...precedingHashtags];
	if (Array.isArray(common.tags)) {
		const accepted = acceptedHashtags([
			...prefix,
			...common.tags.filter((tag): tag is string => typeof tag === 'string')
		]);
		common.tags = common.tags.filter(
			(tag) => typeof tag === 'string' && accepted.has(tag.trim().toLowerCase())
		);
		prefix.push(...(common.tags as string[]));
	}
	async function fieldFacets(
		text: string,
		existing: unknown,
		unchanged: boolean
	) {
		const generated = await buildFacets(
			text,
			unchanged ? async () => '' : resolve,
			prefix
		);
		prefix.push(...hashtagsInText(text));
		if (!unchanged || !Array.isArray(existing)) return generated;
		// Retain unchanged mention/link/extension identities and refresh only tags.
		const retained = (existing as Facet[]).flatMap((facet) => {
			const features = facet.features.filter(
				(feature) => feature.$type !== 'app.bsky.richtext.facet#tag'
			);
			return features.length ? [{ ...facet, features }] : [];
		});
		return [
			...retained,
			...generated.filter(
				(facet) =>
					facet.features.some(
						(feature) => feature.$type === 'app.bsky.richtext.facet#tag'
					) &&
					!retained.some(
						(other) =>
							facet.index.byteStart < other.index.byteEnd &&
							facet.index.byteEnd > other.index.byteStart
					)
			)
		].sort((a, b) => a.index.byteStart - b.index.byteStart);
	}
	for (const pattern of patterns) {
		const oldPattern = retainedRow(oldPatterns, pattern);
		for (const field of patternFields) {
			const text = str(pattern[field]);
			const key = `${field}Facets`;
			if (oldPattern && text === str(oldPattern[field])) {
				if (oldPattern[key] !== undefined) pattern[key] = oldPattern[key];
				else delete pattern[key];
			} else if (!preservePatterns) {
				const facets = await buildFacets(text, resolve, prefix);
				if (facets.length) pattern[key] = facets as unknown as JsonObject[];
				else delete pattern[key];
			}
			// Keep authored facets on unchanged leaves. AppView applies the discovery
			// limit to every saved tag, including tags this client would not generate.
			prefix.push(...facetHashtags(text, pattern[key]));
		}
	}
	const oldMaterials = Array.isArray(previous.materials)
		? previous.materials.map(record)
		: [];
	if (Array.isArray(common.materials))
		for (let index = 0; index < common.materials.length; index++) {
			const item = record(common.materials[index]);
			const source =
				oldMaterials[
					typeof item.existingIndex === 'number' ? item.existingIndex : index
				];
			const facets = await fieldFacets(
				str(item.text),
				item.facets,
				!!before && str(item.text) === str(source?.text)
			);
			if (facets.length) item.facets = facets as unknown as JsonObject[];
			else delete item.facets;
		}
	return next;
}
export function postPayload(draft: Draft, facets: Facet[]): PostPayload {
	const payload: PostPayload = {
		text: draft.text,
		langs: [...draft.langs],
		sponsored: draft.sponsored
	};
	if (facets.length) payload.facets = facets;
	if (draft.instagramImport)
		payload.instagramImport = { ...draft.instagramImport };
	if (draft.kind === 'project' && draft.project)
		payload.project = projectWrite(draft.project) as unknown as Project;
	if (draft.reply) payload.reply = draft.reply;
	if (draft.images.length)
		payload.images = draft.images.map((image) => {
			if (!image.uploaded)
				throw new UserFacingError('An image has not finished uploading.');
			return {
				image: image.uploaded,
				alt: image.alt,
				...(image.width && image.height
					? { aspectRatio: { width: image.width, height: image.height } }
					: {})
			};
		});
	if (draft.quote)
		payload.embed = {
			quote: draft.quote,
			...(draft.quoteImageCid ? { imageCid: draft.quoteImageCid } : {})
		};
	else if (draft.external) payload.embed = { external: draft.external };
	return payload;
}
/** Edits contain changed known fields, leaving unedited authored extensions intact. */
export function editChanges(draft: Draft, facets: Facet[]): PostEditChanges {
	const before = draft.edit?.editable;
	if (!before) throw new UserFacingError('The post edit context is missing.');
	const changes: JsonObject = {};
	if (draft.text !== before.text) {
		changes.text = draft.text;
		changes.facets = facets.length ? (facets as unknown as JsonObject[]) : null;
	}
	if (draft.sponsored !== before.sponsored) changes.sponsored = draft.sponsored;
	if (JSON.stringify(draft.langs) !== JSON.stringify(before.langs || []))
		changes.langs = draft.langs;
	const images = draft.images.map((image) =>
		image.existingIndex !== undefined
			? { existingIndex: image.existingIndex, alt: image.alt }
			: {
					image: image.uploaded,
					alt: image.alt,
					...(image.width && image.height
						? { aspectRatio: { width: image.width, height: image.height } }
						: {})
				}
	);
	const oldImages = Array.isArray(before.images) ? before.images : [];
	const unchanged =
		images.length === oldImages.length &&
		draft.images.every(
			(image, i) =>
				image.existingIndex === i && image.alt === record(oldImages[i]).alt
		);
	if (!unchanged)
		changes.images = images.length ? (images as unknown as JsonObject[]) : null;
	const originalEmbed = record(before.embed);
	if (draft.embedChanges !== undefined) changes.embed = draft.embedChanges;
	if (draft.external) {
		const original = record(originalEmbed.external);
		const external: JsonObject = {};
		for (const key of ['uri', 'title', 'description', 'thumb'] as const) {
			if (JSON.stringify(original[key]) !== JSON.stringify(draft.external[key]))
				external[key] = (draft.external[key] as unknown as JsonObject) ?? null;
		}
		if (Object.keys(external).length)
			changes.embed = {
				...(!originalEmbed.$type ? { $type: 'app.bsky.embed.external' } : {}),
				external
			};
	} else if (originalEmbed.external && draft.embedChanges === undefined)
		changes.embed = null;
	if (draft.kind === 'project' && draft.project) {
		const projectDiff = diffProject(record(before.project), draft.project);
		if (Object.keys(projectDiff).length) changes.project = projectDiff;
	}
	return changes as unknown as PostEditChanges;
}
const commonEntryFields: Record<string, string[]> = {
	patterns: [
		'url',
		'name',
		'nameFacets',
		'difficulty',
		'designer',
		'designerFacets',
		'publisher',
		'publisherFacets',
		'selfDrafted'
	],
	materials: ['text', 'facets'],
	additionalCrafts: ['craftType']
};
const detailEntryFields: Record<string, string[]> = {
	techniques: ['technique', 'subtypes'],
	subtypes: ['subtype'],
	methods: ['method', 'styles'],
	styles: ['style'],
	draftingTechniques: ['technique'],
	plyingTechniques: ['technique'],
	cords: ['material', 'construction', 'diameterMm'],
	knots: ['knot'],
	materials: ['role', 'text'],
	needleGauges: ['gauge']
};
const detailObjectFields: Record<string, string[]> = {
	gauge: ['stitches', 'rows', 'measurement', 'unit'],
	finishedYarn: ['mass', 'length', 'wrapsPerInch'],
	mass: ['amount', 'unit'],
	length: ['amount', 'unit'],
	loopHeight: ['amount', 'unit'],
	sett: ['amount', 'unit']
};

function detailObjectPatch(
	key: string,
	before: unknown,
	after: JsonObject
): JsonObject {
	const base = record(before);
	const patch: JsonObject = {};
	for (const field of detailObjectFields[key]) {
		const value = after[field];
		if (JSON.stringify(base[field]) === JSON.stringify(value)) continue;
		patch[field] =
			detailObjectFields[field] &&
			value &&
			typeof value === 'object' &&
			!Array.isArray(value)
				? detailObjectPatch(field, base[field], value)
				: (value ?? null);
	}
	return patch;
}

function entryPatch(
	before: unknown,
	after: unknown[],
	fields: string[]
): JsonObject[] {
	const original = Array.isArray(before) ? before.map(record) : [];
	return after.map((value) => {
		const item = record(value);
		const index = item.existingIndex;
		const base = typeof index === 'number' ? (original[index] ?? {}) : {};
		const patch: JsonObject =
			typeof index === 'number' ? { existingIndex: index } : {};
		for (const field of fields) {
			if (JSON.stringify(base[field]) === JSON.stringify(item[field])) continue;
			const value = item[field];
			if (
				(field === 'subtypes' || field === 'styles') &&
				Array.isArray(value)
			) {
				patch[field] = entryPatch(base[field], value, detailEntryFields[field]);
			} else patch[field] = value ?? null;
		}
		return patch;
	});
}

function diffProject(before: JsonObject, after: JsonObject): JsonObject {
	const changes: JsonObject = {};
	for (const key of ['common', 'details']) {
		const right = record(after[key]);
		const original = record(before[key]);
		if (key === 'details' && before.details && !after.details) {
			changes.details = null;
			continue;
		}
		const left =
			key === 'details' && original.$type !== right.$type ? {} : original;
		const patch: JsonObject = {};
		for (const leaf of new Set([...Object.keys(left), ...Object.keys(right)])) {
			if (leaf === 'pattern' && Array.isArray(right.patterns)) continue;
			if (leaf === 'needleSizeMm' && Array.isArray(right.needleSizesMm))
				continue;
			if (leaf === 'hookSizeMm' && Array.isArray(right.hookSizesMm)) continue;
			if (JSON.stringify(left[leaf]) === JSON.stringify(right[leaf])) continue;
			if (leaf === 'patterns') {
				patch[leaf] = entryPatch(
					patternRows(left),
					patternRows(right).filter(
						(pattern) =>
							typeof pattern.existingIndex === 'number' ||
							hasPatternContent(pattern)
					),
					commonEntryFields[leaf]
				);
			} else {
				const fields = (
					key === 'common' ? commonEntryFields : detailEntryFields
				)[leaf];
				const value =
					key === 'details' && leaf === 'projectSubtype'
						? str(right[leaf]).trim() || undefined
						: right[leaf];
				if (
					key === 'details' &&
					detailObjectFields[leaf] &&
					value &&
					typeof value === 'object' &&
					!Array.isArray(value)
				) {
					patch[leaf] = detailObjectPatch(leaf, left[leaf], value);
				} else if (
					fields &&
					Array.isArray(value) &&
					value.every((entry) => entry && typeof entry === 'object')
				) {
					patch[leaf] = entryPatch(left[leaf], value, fields);
				} else patch[leaf] = value ?? null;
			}
		}
		if (Object.keys(patch).length) changes[key] = patch;
	}
	return changes;
}
export function freezeWrite(body: JsonObject, draft: Draft): PendingWrite {
	return JSON.parse(
		JSON.stringify({
			key: newOperationKey(),
			body,
			kind: draft.edit ? 'edit' : 'create',
			...(draft.edit
				? {
						did: draft.edit.post.author.did,
						rkey: draft.edit.post.rkey,
						cid: draft.edit.baseCid
					}
				: {})
		})
	) as PendingWrite;
}
