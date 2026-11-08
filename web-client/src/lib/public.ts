import type { Facet } from './api/types';
export const origin = 'https://app.craftsky.social';
export type Author = {
	did: string;
	handle: string;
	displayName?: string;
	avatar?: string;
	crafts?: string[];
	business?: boolean;
};
export type Profile = Author & {
	businessProfile?: import('./api/types').BusinessProfile;
	hasUpcomingEvents?: boolean;
	description?: string;
	pronouns?: string;
	banner?: string;
	crafts: string[];
	links: { url: string; title: string }[];
	business: boolean;
};
export type Image = {
	alt: string;
	thumb?: string;
	fullsize?: string;
	aspectRatio?: { width: number; height: number };
};
export type Project = {
	common: {
		title?: string;
		craftType: string;
		status?: string;
		duration?: string;
		additionalCrafts?: { craftType: string }[];
		pattern?: {
			name?: string;
			url?: string;
			designer?: string;
			publisher?: string;
			difficulty?: string;
			selfDrafted?: boolean;
			nameFacets?: Facet[];
			designerFacets?: Facet[];
			publisherFacets?: Facet[];
		};
		materials?: { text: string; facets?: Facet[] }[];
		colors?: string[];
		designTags?: string[];
		tags?: string[];
	};
	details?: Record<string, unknown>;
};
export type Post = {
	rkey: string;
	text: string;
	author: Author;
	createdAt: string;
	editedAt?: string;
	images: Image[];
	sponsored: boolean;
	project?: Project;
	video?: { playlist: string; thumbnail?: string; alt: string };
};
export type PostPage = { items: Post[]; cursor?: string };
export const profilePath = (did: string) =>
	`/profiles/${encodeURIComponent(did)}`;
export const postPath = (post: Post) =>
	`/posts/${encodeURIComponent(post.author.did)}/${encodeURIComponent(post.rkey)}`;
export const authorName = (author: Author) =>
	author.displayName?.trim() || author.handle || author.did;
export function safeURL(value?: string): string | undefined {
	if (!value) return;
	try {
		const url = new URL(value);
		if (url.protocol === 'https:' && !url.username && !url.password)
			return url.href;
	} catch {
		/* Omit malformed public URLs. */
	}
}
export function label(value: string): string {
	const token = value.split('#').at(-1) || value;
	const names: Record<string, string> = {
		wip: 'Work in progress',
		finished: 'Finished',
		homeGoods: 'Home goods',
		dk: 'DK',
		selfDrafted: 'Self drafted'
	};
	return (
		names[token] ||
		token
			.replace(/([a-z])([A-Z])/g, '$1 $2')
			.replace(/[-_]/g, ' ')
			.replace(/^./, (c) => c.toUpperCase())
	);
}
const dateFormatter = new Intl.DateTimeFormat('en-GB', {
	day: 'numeric',
	month: 'long',
	year: 'numeric',
	timeZone: 'UTC'
});
export const date = (value: string) => dateFormatter.format(new Date(value));
export const description = (text: string) =>
	Array.from(text.replace(/\s+/g, ' ').trim()).slice(0, 200).join('');
export function postTitle(post: Post): string {
	return (
		post.project?.common.title ||
		`${post.project ? 'Project' : 'Post'} by ${authorName(post.author)}`
	);
}

/** Known structured gauge fields only; unknown union fields remain unrendered. */
export function gaugeText(value: unknown): string | undefined {
	if (!value || typeof value !== 'object') return;
	const gauge = value as Record<string, unknown>;
	if (
		typeof gauge.stitches !== 'number' ||
		typeof gauge.measurement !== 'number' ||
		!['cm', 'in'].includes(String(gauge.unit))
	)
		return;
	return `${gauge.stitches} stitches${typeof gauge.rows === 'number' ? ` × ${gauge.rows} rows` : ''} over ${gauge.measurement} ${gauge.unit}`;
}

const compactDateFormatter = new Intl.DateTimeFormat('en-GB', {
	day: 'numeric',
	month: 'short',
	year: '2-digit',
	timeZone: 'UTC'
});

// Match Flutter's RelativeTimeText: exactly seven days is still relative.
export function compactDate(value: string, now = Date.now()): string {
	const delta = now - new Date(value).getTime();
	if (delta > 7 * 86400000) {
		// Intl/CLDR versions differ on punctuation around a two-digit year.
		// Flutter uses a literal d MMM yy pattern, so keep only its date parts.
		return compactDateFormatter
			.formatToParts(new Date(value))
			.filter((part) => ['day', 'month', 'year'].includes(part.type))
			.map((part) => part.value)
			.join(' ');
	}

	if (delta < 60000) return 'now';
	if (delta < 3600000) return `${Math.floor(delta / 60000)}m`;
	if (delta < 86400000) return `${Math.floor(delta / 3600000)}h`;
	return `${Math.floor(delta / 86400000)}d`;
}
const timestampFormatter = new Intl.DateTimeFormat('en-GB', {
	dateStyle: 'full',
	timeStyle: 'short',
	timeZone: 'UTC'
});
export const fullTimestamp = (value: string) =>
	timestampFormatter.format(new Date(value)) + ' UTC';
