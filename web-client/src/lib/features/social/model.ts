import type {
	Author,
	Post,
	PostPresentation,
	Facet,
	ProfileCustomisation
} from '#lib/api/types.ts';
import type { Post as PublicPost } from '#lib/public.ts';
import palettes from './palette.json';
export function visible(value: PostPresentation): value is Post {
	return (
		'author' in value && !!value.author && !!value.uri && !value.availability
	);
}
export function postAddress(uri: string) {
	const match =
		/^at:\/\/([^/]+)\/social\.craftsky\.feed\.post\/([^/?#]+)$/.exec(uri);
	return match ? { did: match[1], rkey: match[2] } : undefined;
}
export function postRoute(
	post: Pick<Post, 'uri' | 'author'> & { rkey?: string }
) {
	const address = postAddress(post.uri);
	return `/app/posts/${encodeURIComponent(address?.did || post.author.did)}/${encodeURIComponent(address?.rkey || post.rkey || '')}`;
}
export function publicPost(post: Post): PublicPost {
	return {
		rkey: post.rkey,
		text: post.text,
		author: {
			did: post.author.did,
			handle: post.author.handle,
			displayName: post.author.displayName || undefined,
			avatar: post.author.avatar || undefined,
			crafts: post.author.crafts,
			business: post.author.accountType === 'business'
		},
		createdAt: post.createdAt,
		editedAt: post.editedAt,
		images: post.images || [],
		project: post.project,
		video: post.video,
		sponsored: post.sponsored
	};
}
export function safeLink(raw: string) {
	try {
		const url = new URL(raw);
		if (url.protocol === 'https:' && !url.username && !url.password)
			return url.href;
	} catch {}
	return undefined;
}
export interface TextPart {
	text: string;
	href?: string;
	external?: boolean;
}
export function facetParts(
	text: string,
	facets: Facet[] | null | undefined
): TextPart[] {
	const offsets = new Map<number, number>([[0, 0]]);
	let bytes = 0,
		index = 0;
	for (const point of text) {
		bytes += new TextEncoder().encode(point).length;
		index += point.length;
		offsets.set(bytes, index);
	}
	const out: TextPart[] = [];
	let end = 0;
	const annotations = Array.isArray(facets)
		? facets.filter(
				(facet) =>
					facet &&
					facet.index &&
					Number.isSafeInteger(facet.index.byteStart) &&
					Number.isSafeInteger(facet.index.byteEnd) &&
					Array.isArray(facet.features)
			)
		: [];
	for (const facet of [...annotations].sort(
		(a, b) => a.index.byteStart - b.index.byteStart
	)) {
		const start = offsets.get(facet.index.byteStart),
			stop = offsets.get(facet.index.byteEnd);
		if (
			start === undefined ||
			stop === undefined ||
			start < end ||
			stop <= start
		)
			continue;
		let href: string | undefined,
			external = false;
		for (const feature of facet.features) {
			if (!feature || typeof feature !== 'object') continue;
			const type = String(feature.$type || '');
			if (type.endsWith('#link') && typeof feature.uri === 'string') {
				href = safeLink(feature.uri);
				external = !!href;
			}
			if (
				type.endsWith('#mention') &&
				typeof feature.did === 'string' &&
				/^did:[a-z]+:[^/\s]+$/.test(feature.did)
			)
				href = '/app/profile/' + encodeURIComponent(feature.did);
			if (type.endsWith('#tag') && typeof feature.tag === 'string')
				href = '/app/tags/' + encodeURIComponent(feature.tag);
			if (href) break;
		}
		if (!href) continue;
		if (start > end) out.push({ text: text.slice(end, start) });
		out.push({ text: text.slice(start, stop), href, external });
		end = stop;
	}
	if (end < text.length || !out.length) out.push({ text: text.slice(end) });
	return out;
}
export function profileStyle(value?: ProfileCustomisation) {
	const key = value?.colour as keyof typeof palettes;
	const p = Object.hasOwn(palettes, key) ? palettes[key] : palettes.cobalt;
	return p
		? `--profile-accent:${p.base};--profile-soft:${p.softContainer};--profile-foreground:${p.foreground};--profile-dark:${p.darkAccent};--profile-texture-tint:${p.textureTint}`
		: '';
}
export function texture(value?: ProfileCustomisation) {
	if (
		['knitting', 'crochet', 'sewing', 'embroidery', 'quilting'].some(
			(craft) => value?.profileBackground === 'craft-' + craft
		)
	)
		return `url('/profile_backgrounds/${value!.profileBackground}.svg')`;
	return [
		'bayerdark',
		'cubedark',
		'dotcrossdark',
		'scallopdark',
		'skewdark',
		'x2'
	].includes(value?.profileBackground || '')
		? `url('/profile_backgrounds/${value!.profileBackground}.png')`
		: undefined;
}
export const name = (author: Author) =>
	author.displayName || author.handle || author.did;

export function pinSlot(
	post: Pick<Post, 'reply' | 'project' | 'quote'>
): 'standard' | 'project' | undefined {
	if (post.reply || (post.project && post.quote)) return;
	return post.project ? 'project' : 'standard';
}
export function quoteCanRender(post: Post): boolean {
	const view = post.quoteView;
	return (
		!!view &&
		view.state === 'visible' &&
		!!view.post &&
		!!view.post.author?.did &&
		!view.post.availability &&
		!view.post.moderation
	);
}
