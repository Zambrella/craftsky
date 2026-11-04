import { expect, type Page } from '@playwright/test';
import type { Post, Profile } from '../../src/lib/api/types';

/** Fresh reads through the real, account-bound browser BFF; never obtains PDS credentials. */
export class LiveAPI {
	constructor(
		readonly page: Page,
		readonly account: string,
		readonly did: string
	) {}
	async read<T>(path: string): Promise<{ status: number; value?: T }> {
		const response = await this.page.request.get('/api/appview' + path, {
			headers: {
				'X-Craftsky-Account-Id': this.account,
				'Cache-Control': 'no-store'
			}
		});
		const status = response.status();
		const value = status === 200 ? ((await response.json()) as T) : undefined;
		await response.dispose();
		return { status, value };
	}
	async bio(bio: string) {
		await expect
			.poll(
				async () =>
					(await this.read<Profile>('/v1/profiles/me')).value?.description,
				{ timeout: 60_000, intervals: [500, 1000] }
			)
			.toBe(bio);
	}
	async find(text: string): Promise<Post | undefined> {
		const result = await this.read<{ items: Post[] }>(
			`/v1/profiles/@${encodeURIComponent(this.did)}/posts?limit=100`
		);
		return result.value?.items.find(
			(post) => post.author?.did === this.did && post.text === text
		);
	}
	async published(text: string): Promise<Post> {
		let post: Post | undefined;
		await expect
			.poll(
				async () => {
					post = await this.find(text);
					return !!post;
				},
				{ timeout: 60_000, intervals: [500, 1000] }
			)
			.toBe(true);
		return post!;
	}
	path(post: Post) {
		if (
			post.author.did !== this.did ||
			post.uri !== `at://${this.did}/social.craftsky.feed.post/${post.rkey}`
		)
			throw new Error('Test post ownership mismatch.');
		return `/v1/posts/${encodeURIComponent(this.did)}/${encodeURIComponent(post.rkey)}`;
	}
	async liked(post: Post) {
		await expect
			.poll(
				async () => {
					const current = (await this.read<Post>(this.path(post))).value;
					return !!current?.viewerHasLiked && current.likeCount >= 1;
				},
				{ timeout: 60_000, intervals: [500, 1000] }
			)
			.toBe(true);
	}
	async edited(post: Post, text: string) {
		await expect
			.poll(
				async () => {
					const current = (await this.read<Post>(this.path(post))).value;
					return (
						!!current &&
						current.uri === post.uri &&
						current.cid !== post.cid &&
						!!current.editedAt &&
						current.text === text &&
						current.images?.length === 1 &&
						current.images[0].cid === post.images![0].cid &&
						current.viewerHasLiked &&
						current.likeCount >= 1
					);
				},
				{ timeout: 60_000, intervals: [500, 1000] }
			)
			.toBe(true);
	}
	async deleted(post: Post) {
		await expect
			.poll(async () => (await this.read(this.path(post))).status, {
				timeout: 60_000,
				intervals: [500, 1000]
			})
			.toBe(404);
	}
}
