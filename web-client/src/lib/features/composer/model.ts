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
import catalogue from './catalogue.json';
export interface CatalogueEntry {
	id: string;
	family: string;
	label: string;
	craft?: string | null;
	parents: string[];
}
const entries: CatalogueEntry[] = catalogue;
export const options = (family: string, craft?: string, parent?: string) =>
	entries.filter(
		(entry) =>
			entry.family === family &&
			(!entry.craft || !craft || entry.craft === craft) &&
			(!parent || !entry.parents.length || entry.parents.includes(parent))
	);
export const label = (token: string) =>
	catalogue.find((entry) => entry.id === token)?.label ||
	token.split('#').at(-1) ||
	token;
export const defaultProject = (crafts: readonly string[] = []): JsonObject => {
	const craft = crafts
		.map(
			(value) =>
				options('craft').find((item) => item.id.split('#').at(-1) === value)?.id
		)
		.find(Boolean);
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
		const pattern = record(common.pattern);
		const materials = Array.isArray(common.materials) ? common.materials : [];
		if (
			materials.length > 10 ||
			materials.some(
				(value) => !validateText(str(record(value).text), 100, 1000)
			)
		)
			errors.push('Use up to ten materials, with up to 100 characters each.');
		if (!common.craftType) errors.push('Choose a primary craft.');
		for (const [value, max, bytes, name] of [
			[common.title, 200, 500, 'Project title'],
			[common.duration, 100, 500, 'Duration'],
			[pattern.name, 200, 500, 'Pattern name'],
			[pattern.designer, 200, 500, 'Designer'],
			[pattern.publisher, 200, 500, 'Publisher'],
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
	}
	return errors;
}
function positive(value: unknown) {
	return typeof value === 'number' && Number.isInteger(value) && value > 0;
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
	const pattern = record(common.pattern);
	const oldPattern = record(previous.pattern);
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
	for (const field of ['name', 'designer', 'publisher']) {
		const text = str(pattern[field]);
		const facets = await fieldFacets(
			text,
			pattern[`${field}Facets`],
			!!before && text === str(oldPattern[field])
		);
		if (facets.length)
			pattern[`${field}Facets`] = facets as unknown as JsonObject[];
		else delete pattern[`${field}Facets`];
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
	if (draft.kind === 'project')
		payload.project = draft.project as unknown as Project;
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
function diffProject(before: JsonObject, after: JsonObject): JsonObject {
	const changes: JsonObject = {};
	for (const key of ['common', 'details']) {
		const left = record(before[key]),
			right = record(after[key]);
		const patch: JsonObject = {};
		for (const leaf of new Set([...Object.keys(left), ...Object.keys(right)])) {
			if (JSON.stringify(left[leaf]) === JSON.stringify(right[leaf])) continue;
			if (leaf === 'materials' || leaf === 'additionalCrafts') {
				const values = Array.isArray(right[leaf]) ? right[leaf] : [];
				patch[leaf] = values.map((item) => record(item));
			} else patch[leaf] = right[leaf] ?? null;
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
