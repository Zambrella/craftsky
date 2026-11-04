import { describe, it, expect, vi } from 'vitest';
import type { Post } from '#lib/api/types.ts';
import { Publications, matchesProject } from './publications.svelte.ts';
const owner = 'did:plc:alice';
const post = (id = 'new', extra: Partial<Post> = {}): Post => ({
	uri: `at://${owner}/social.craftsky.feed.post/${id}`,
	cid: 'cid-' + id,
	rkey: id,
	author: { did: owner, handle: 'alice.test' },
	text: 'Confirmed',
	createdAt: new Date().toISOString(),
	indexedAt: new Date().toISOString(),
	tags: [],
	langs: ['en'],
	sponsored: false,
	likeCount: 0,
	repostCount: 0,
	quoteCount: 0,
	replyCount: 0,
	viewerHasLiked: false,
	viewerHasReposted: false,
	viewerHasReplied: false,
	viewerHasSaved: false,
	...extra
});
describe('confirmed publications', () => {
	it('bridges stale list reads without duplicate insertion after indexing', () => {
		const state = new Publications(owner),
			accepted = post();
		expect(state.merge([], () => true)).toEqual([]);
		state.accept(accepted);
		state.accept(accepted);
		expect(state.merge([], () => true)).toEqual([accepted]);
		expect(state.merge([accepted], () => true)).toEqual([accepted]);
	});
	it('trusts authoritative protected rows and newer record versions', () => {
		const state = new Publications(owner),
			accepted = post();
		state.accept(accepted);
		const protectedRow = {
			uri: accepted.uri,
			availability: 'blocked' as const,
			relationship: { state: 'blocked', revealable: false }
		};
		expect(state.merge([protectedRow], () => true)).toEqual([protectedRow]);
		const updated = {
			...accepted,
			cid: 'edited-cid',
			text: 'Edited',
			editedAt: new Date().toISOString()
		};
		expect(state.merge([updated], () => true)[0]).toEqual(updated);
		state.observe([updated]);
		expect(state.merge([], () => true)[0]).toEqual(updated);
	});
	it('keeps account ownership and expiry bounded without persisting private state', () => {
		const state = new Publications('did:plc:bob');
		state.accept(post());
		expect(state.posts).toEqual([]);
		const owned = new Publications(owner);
		owned.accept(post());
		vi.spyOn(Date, 'now').mockReturnValue(Date.now() + 300001);
		try {
			expect(owned.merge([], () => true)).toEqual([]);
		} finally {
			vi.restoreAllMocks();
		}
	});
	it('does not resurrect deleted confirmed records from a stale list', () => {
		const state = new Publications(owner),
			accepted = post();
		state.accept(accepted);
		state.remove(accepted.uri);
		expect(state.merge([accepted], () => true)).toEqual([]);
	});
	it('adds confirmed comments/replies and counters once without claiming unloaded replies are complete', () => {
		const state = new Publications(owner),
			root = post('root'),
			comment = post('comment', { reply: { root, parent: root } }),
			reply = post('reply', { reply: { root, parent: comment } });
		state.accept(comment, root);
		state.accept(reply, comment);
		state.accept(reply, comment);
		const comments = state.comments([], root.uri);
		expect(comments).toHaveLength(1);
		expect(comments[0].replies.items.map((item) => item.post.uri)).toEqual([
			reply.uri
		]);
		expect(comments[0].replies.loaded).toBe(false);
		expect(state.overlay(root)).toMatchObject({ replyCount: 1 });
		expect(state.overlay(comment)).toMatchObject({ replyCount: 1 });
	});
	it('honours primary craft and active filters for new project rows', () => {
		const project = post('project', {
			project: {
				common: {
					craftType: 'social.craftsky.feed.defs#knitting',
					additionalCrafts: [
						{ craftType: 'social.craftsky.feed.defs#crochet' }
					],
					status: 'defs#finished',
					colors: ['defs#blue'],
					pattern: { difficulty: 'defs#easy', selfDrafted: true }
				},
				details: { projectType: 'defs#sweater', yarnWeight: 'defs#dk' }
			}
		});
		expect(
			matchesProject(project, 'knitting', {
				status: ['defs#finished'],
				color: ['defs#blue'],
				patternDifficulty: ['defs#easy'],
				selfDrafted: true,
				projectType: ['defs#sweater']
			})
		).toBe(true);
		expect(matchesProject(project, 'crochet', {})).toBe(false);
		expect(
			matchesProject(project, 'knitting', { yarnWeight: ['defs#aran'] })
		).toBe(false);
	});
});
