import { CraftskyAPI } from './client';
import { publicReadPath } from './public-path';
import { publicReportPath } from './public-report-path';
import type { Fetcher } from './transport';
import type {
	Profile as PublicProfile,
	Post as PublicPost
} from '#lib/public.ts';
import type { Post, Profile, CommentSection, ReplyPage, Author } from './types';

type ObjectValue = Record<string, unknown>;
const object = (value: unknown): value is ObjectValue =>
	!!value && typeof value === 'object' && !Array.isArray(value);
const denied = () =>
	Response.json(
		{
			error: 'authentication_required',
			message: 'Sign in to use this feature.'
		},
		{ status: 403 }
	);

/** Display-only defaults exist here, never in the anonymous wire contract. */
export function publicPostForDisplay(raw: ObjectValue | PublicPost): Post {
	const value = raw as ObjectValue;
	const quote = object(value.quoteView) ? value.quoteView : undefined;
	return {
		...value,
		author: publicAuthorForDisplay(value.author as ObjectValue),
		indexedAt: value.createdAt,
		viewerHasLiked: false,
		viewerHasReposted: false,
		viewerHasReplied: false,
		viewerHasSaved: false,
		quoteView: quote
			? {
					state: quote.state,
					revealable: false,
					post: object(quote.post)
						? publicPostForDisplay(quote.post)
						: undefined
				}
			: undefined
	} as unknown as Post;
}
function publicAuthorForDisplay(value: ObjectValue): Author {
	const { business, ...author } = value;
	return {
		...author,
		accountType: value.accountType || (business ? 'business' : 'regular')
	} as Author;
}
function adapt(value: unknown): unknown {
	if (Array.isArray(value)) return value.map(adapt);
	if (!object(value)) return value;
	if (typeof value.text === 'string' && object(value.author))
		return publicPostForDisplay(value);
	if (typeof value.did === 'string' && typeof value.handle === 'string') {
		const author = publicAuthorForDisplay(value);
		if (Array.isArray(value.links))
			return {
				...author,
				viewerIsFollowing: false,
				isCraftskyProfile: true,
				business: value.businessProfile
			} as Profile;
		return author;
	}
	if (Array.isArray(value.items))
		return {
			...value,
			items: value.items.map(adapt),
			totalCount: value.totalCount ?? value.items.length
		};
	return value;
}

export function publicProfileForDisplay(value: PublicProfile): Profile {
	return adapt(value) as Profile;
}

function publicFetcher(fetcher: Fetcher): Fetcher {
	return async (input, init) => {
		const raw = String(input);
		if (!raw.startsWith('/api/public/v1/')) return denied();
		const url = new URL(raw, 'https://app.craftsky.social');
		let path = url.pathname.slice('/api/public/v1/'.length);
		const query = new URLSearchParams(url.searchParams);
		if (init?.method === 'POST') {
			let canonical: string;
			try {
				canonical = publicReportPath(path, query);
			} catch {
				return denied();
			}
			return fetcher('/api/public/' + canonical, {
				method: 'POST',
				body: init.body,
				headers: {
					accept: 'application/json',
					'content-type': 'application/json'
				},
				credentials: 'omit',
				redirect: 'manual',
				cache: 'no-store',
				signal: init.signal
			});
		}
		if ((init?.method || 'GET') !== 'GET') return denied();
		let section: 'comments' | 'replies' | undefined;
		if (/^posts\/[^/]+\/[^/]+\/(comments|replies)$/.test(path)) {
			section = path.endsWith('/comments') ? 'comments' : 'replies';
			path = path.replace(/\/(comments|replies)$/, '/thread');
			query.delete('sort');
			query.delete('focus');
		}
		let canonical: string;
		try {
			canonical = publicReadPath(path, query);
		} catch {
			return denied();
		}
		const response = await fetcher('/api/public/' + canonical, {
			method: 'GET',
			headers: { accept: 'application/json' },
			credentials: 'omit',
			redirect: 'manual',
			cache: 'no-store',
			signal: init?.signal
		});
		if (!response.ok || /\/video-captions\/[^/]+$/.test(path)) return response;
		const value: unknown = await response.json();
		let display = adapt(value);
		if (
			section &&
			object(value) &&
			object(value.replies) &&
			Array.isArray(value.replies.items)
		) {
			const replies = {
				items: value.replies.items.map((item) => ({ post: adapt(item) })),
				cursor: value.replies.cursor,
				loaded: true
			} as ReplyPage;
			display =
				section === 'replies'
					? replies
					: ({
							post: adapt(value.post),
							sort: 'oldest',
							ancestors: Array.isArray(value.ancestors)
								? value.ancestors.map((item) =>
										object(item)
											? {
													state: item.state,
													post: object(item.post) ? adapt(item.post) : undefined
												}
											: undefined
									)
								: [],
							comments: {
								items: replies.items.map((item) => ({
									post: item.post,
									replies: { items: [], loaded: false },
									placement: 'normal'
								})),
								cursor: replies.cursor
							}
						} as CommentSection);
		}
		return Response.json(display, {
			status: response.status,
			headers: { 'cache-control': 'no-store' }
		});
	};
}

/** Reuses the existing client interface for shared UI. Its transport is a
 * closed read/report map; other private/mutation methods fail before IO. */
export class PublicCraftskyAPI extends CraftskyAPI {
	override readonly audience = 'guest' as const;
	constructor(fetcher: Fetcher = fetch) {
		super(publicFetcher(fetcher), '/api/public');
	}
}
