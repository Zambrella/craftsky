import type {
	Post,
	PostPresentation,
	CommentItem,
	PostEditResult
} from '#lib/api/types.ts';
import { visible } from './model';
import { craftKey } from './craft-feeds';
import type { ProjectFilters } from '#lib/api/types.ts';
export const publicationsContext = Symbol('confirmed-publications');

/** PDS-confirmed writes bridge the delay before list reads observe the new record. */
export class Publications {
	posts = $state<Post[]>([]);
	removed = $state<string[]>([]);
	private counters = $state<
		Record<string, { replyCount: number; quoteCount: number }>
	>({});
	private acceptedAt = new Map<string, number>();
	private edits = $state<Record<string, Post>>({});
	private editChecks = new Map<string, () => void>();

	/** Follow the accepted CID across navigation; stale index reads are not success. */
	watchEdit(
		rkey: string,
		result: PostEditResult,
		load: () => Promise<PostPresentation>
	) {
		const uri = `at://${this.did}/social.craftsky.feed.post/${rkey}`;
		this.editChecks.get(uri)?.();
		let cancelled = false;
		let timer: ReturnType<typeof setTimeout> | undefined;
		const cancel = () => {
			cancelled = true;
			clearTimeout(timer);
		};
		this.editChecks.set(uri, cancel);
		const finish = () => {
			cancel();
			if (this.editChecks.get(uri) === cancel) this.editChecks.delete(uri);
		};
		const check = async (attempt = 0) => {
			try {
				const post = await load();
				if (cancelled || this.removed.includes(uri)) return;
				if (
					visible(post) &&
					post.uri === uri &&
					post.author.did === this.did &&
					post.cid === result.acceptedCid
				) {
					this.edits[uri] = post;
					const oldest = Object.keys(this.edits);
					if (oldest.length > 50) delete this.edits[oldest[0]];
					this.observe([post]);
					finish();
					return;
				}
			} catch {
				// A background read failure does not change the accepted write outcome.
			}
			if (cancelled) return;
			if (attempt >= 30) finish();
			else
				timer = setTimeout(
					() => void check(attempt + 1),
					attempt === 0 ? 1000 : 2000
				);
		};
		void check();
	}
	dispose() {
		for (const cancel of this.editChecks.values()) cancel();
		this.editChecks.clear();
	}
	constructor(readonly did: string) {}
	private recent() {
		return this.posts.filter(
			(post) => Date.now() - (this.acceptedAt.get(post.uri) || 0) < 300000
		);
	}
	observe(values: PostPresentation[]) {
		const latest = new Map(
			values.filter(visible).map((post) => [post.uri, post])
		);
		for (const [uri, observed] of latest) {
			const edit = this.edits[uri];
			if (
				edit &&
				observed.cid !== edit.cid &&
				(observed.editedAt || '') > (edit.editedAt || '')
			)
				this.edits[uri] = observed;
		}
		if (
			this.posts.some((post) => {
				const observed = latest.get(post.uri);
				return (
					observed &&
					observed.cid !== post.cid &&
					(observed.editedAt || '') >= (post.editedAt || '')
				);
			})
		)
			this.posts = this.posts.map((post) => {
				const observed = latest.get(post.uri);
				return observed && (observed.editedAt || '') >= (post.editedAt || '')
					? observed
					: post;
			});
	}
	accept(post: Post, context?: Post) {
		if (
			post.author.did !== this.did ||
			this.posts.some((item) => item.uri === post.uri)
		)
			return;
		this.acceptedAt.set(post.uri, Date.now());
		this.posts = [post, ...this.posts].slice(0, 50);
		for (const uri of this.acceptedAt.keys())
			if (!this.posts.some((post) => post.uri === uri))
				this.acceptedAt.delete(uri);
		if (
			context &&
			(post.reply?.parent.uri === context.uri ||
				post.quote?.uri === context.uri)
		) {
			const base = this.counters[context.uri] || {
				replyCount: context.replyCount,
				quoteCount: context.quoteCount
			};
			this.counters = {
				...this.counters,
				[context.uri]: {
					replyCount:
						Math.max(base.replyCount, context.replyCount) +
						(post.reply ? 1 : 0),
					quoteCount:
						Math.max(base.quoteCount, context.quoteCount) + (post.quote ? 1 : 0)
				}
			};
		}
	}
	remove(uri: string) {
		this.editChecks.get(uri)?.();
		this.editChecks.delete(uri);
		delete this.edits[uri];
		this.acceptedAt.delete(uri);
		this.posts = this.posts.filter((item) => item.uri !== uri);
		this.removed = [...this.removed, uri].slice(-100);
	}
	overlay(value: PostPresentation): PostPresentation {
		if (!visible(value)) return value; // Authoritative privacy/moderation always wins.
		const edit = this.edits[value.uri];
		// Keep live counters/viewer interaction state; replace only stale record content.
		const post =
			edit &&
			value.cid !== edit.cid &&
			(value.editedAt || '') <= (edit.editedAt || '')
				? {
						...value,
						...edit,
						likeCount: value.likeCount,
						replyCount: value.replyCount,
						repostCount: value.repostCount,
						quoteCount: value.quoteCount,
						viewerHasLiked: value.viewerHasLiked,
						viewerHasSaved: value.viewerHasSaved,
						viewerHasReplied: value.viewerHasReplied,
						viewerHasReposted: value.viewerHasReposted
					}
				: value;
		const counts = this.counters[post.uri];
		return counts
			? {
					...post,
					replyCount: Math.max(post.replyCount, counts.replyCount),
					quoteCount: Math.max(post.quoteCount, counts.quoteCount)
				}
			: post;
	}
	merge(
		values: PostPresentation[],
		include: (post: Post) => boolean
	): PostPresentation[] {
		const uris = new Set(values.map((item) => item.uri));
		return [
			...this.recent().filter((item) => include(item) && !uris.has(item.uri)),
			...values
		]
			.filter((item) => !item.uri || !this.removed.includes(item.uri))
			.map((item) => this.overlay(item));
	}
	comments(values: CommentItem[], root: string): CommentItem[] {
		const present = new Set(values.map((item) => item.post.uri));
		const fresh: CommentItem[] = this.recent()
			.filter(
				(item) => item.reply?.parent.uri === root && !present.has(item.uri)
			)
			.map((post) => ({
				post,
				placement: 'viewerAuthored',
				replies: { items: [], loaded: false }
			}));
		return [...fresh, ...values]
			.filter((item) => !this.removed.includes(item.post.uri || ''))
			.map((item) => {
				const parents = new Set([
					item.post.uri,
					...item.replies.items.map((reply) => reply.post.uri)
				]);
				const replies = this.merge(
					item.replies.items.map((reply) => reply.post),
					(post) => !!post.reply && parents.has(post.reply.parent.uri)
				);
				return {
					...item,
					post: this.overlay(item.post),
					replies: {
						...item.replies,
						loaded: item.replies.loaded,
						items: replies.map((post) => ({
							...item.replies.items.find(
								(reply) => reply.post.uri === post.uri
							),
							post
						}))
					}
				};
			});
	}
}

/** Only the primary craft controls craft-feed membership, matching AppView. */
export function matchesProject(
	post: Post,
	craft: string,
	filters: ProjectFilters
) {
	const project = post.project;
	if (
		!project ||
		post.reply ||
		craftKey(project.common.craftType) !== craftKey(craft)
	)
		return false;
	const fields: Record<string, unknown> = {
		...project.details,
		status: project.common.status,
		patternDifficulty: project.common.pattern?.difficulty,
		color: project.common.colors,
		designTag: project.common.designTags,
		selfDrafted: project.common.pattern?.selfDrafted ?? false
	};
	return Object.entries(filters).every(([key, selected]) => {
		if (key === 'selfDrafted')
			return fields[key] === (selected === true || selected === 'true');
		const desired = Array.isArray(selected) ? selected : [selected];
		const actual = Array.isArray(fields[key])
			? (fields[key] as unknown[])
			: [fields[key]];
		return desired.some(
			(value) =>
				typeof value === 'string' &&
				actual.some(
					(candidate) =>
						typeof candidate === 'string' &&
						craftKey(candidate) === craftKey(value)
				)
		);
	});
}
