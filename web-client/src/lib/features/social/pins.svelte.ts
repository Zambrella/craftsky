import type { CraftskyAPI, ProfilePins, Post } from '#lib/api/client.ts';
import { pinSlot } from './model';
class ProfilePinState {
	value = $state<ProfilePins>();
	busy = $state(false);
	private loading?: Promise<void>;
	constructor(private readonly api: CraftskyAPI) {}
	async load() {
		if (this.value) return;
		if (this.loading) return this.loading;
		this.loading = this.api.posts
			.pins()
			.then((value) => {
				this.value = value;
			})
			.finally(() => {
				this.loading = undefined;
			});
		return this.loading;
	}
	isPinned(post: Post) {
		const slot = pinSlot(post);
		return slot === 'standard'
			? this.value?.standardPostUri === post.uri
			: slot === 'project'
				? this.value?.projectPostUri === post.uri
				: false;
	}
	async set(post: Post, pinned: boolean) {
		if (this.busy || !pinSlot(post) || !this.value) return;
		this.busy = true;
		try {
			this.value = await (pinned
				? this.api.posts.pin(post.author.did, post.rkey)
				: this.api.posts.unpin(post.author.did, post.rkey));
		} finally {
			this.busy = false;
		}
	}
}
const stores = new WeakMap<CraftskyAPI, ProfilePinState>();
export function profilePins(api: CraftskyAPI) {
	let value = stores.get(api);
	if (!value) {
		value = new ProfilePinState(api);
		stores.set(api, value);
	}
	return value;
}
