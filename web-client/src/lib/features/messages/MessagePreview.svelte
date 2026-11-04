<script lang="ts">
	import { excerpt } from '#lib/text.ts';
	import type { CraftskyAPI } from '#lib/api/client.ts';
	import { craftskyMessageLink, safeAvatar } from './model';
	let {
		api,
		text,
		accepted
	}: { api: CraftskyAPI; text: string; accepted: boolean } = $props();
	let preview = $state<{
		title: string;
		excerpt: string;
		href: string;
		image?: string;
		startsAt?: string;
	}>();
	$effect(() => {
		const target = accepted ? craftskyMessageLink(text) : undefined;
		const client = api;
		let canceled = false;
		preview = undefined;
		async function load() {
			if (!target) return;
			try {
				let value: typeof preview;
				if (target.kind === 'profile') {
					const profile = await client.profiles.get(target.did);
					if (
						profile.did !== target.did ||
						profile.muted ||
						profile.blocking ||
						profile.blockedBy ||
						profile.moderation
					)
						return;
					value = {
						title: profile.displayName || profile.handle,
						excerpt: excerpt(profile.description || '', 160),
						href: target.href,
						image: safeAvatar(profile.avatar)
					};
				} else if (target.kind === 'event') {
					const event = await client.business.event(target.did, target.rkey!);
					if (
						event.did !== target.did ||
						event.rkey !== target.rkey ||
						event.publicSuppressionReasons.length
					)
						return;
					value = {
						title: event.name,
						excerpt: excerpt(event.summary || event.venueName || '', 160),
						href: target.href,
						image: safeAvatar(event.image?.thumb),
						startsAt: event.startsAt
					};
				} else {
					let did = target.did,
						rkey = target.rkey!;
					if (target.focus) {
						const focus = target.focus.slice(5).split('/');
						did = focus[0];
						rkey = focus[2];
					}
					const post = await client.posts.get(did, rkey);
					if (
						!('author' in post) ||
						post.author.did !== did ||
						post.rkey !== rkey ||
						post.moderation ||
						post.availability
					)
						return;
					value = {
						title:
							post.project?.common.title ||
							post.author.displayName ||
							post.author.handle,
						excerpt: excerpt(post.text, 160),
						href: target.href,
						image: safeAvatar(post.images?.[0]?.thumb || post.video?.thumbnail)
					};
				}
				if (!canceled) preview = value;
			} catch {
				/* Preview failures leave the original message link usable. */
			}
		}
		void load();
		return () => {
			canceled = true;
		};
	});
</script>

{#if preview}<a class="message-preview" href={preview.href}
		>{#if preview.image}<img
				src={preview.image}
				alt=""
				loading="lazy"
				referrerpolicy="no-referrer"
			/>{/if}<span
			><strong>{preview.title}</strong>{#if preview.excerpt}<span
					class="excerpt">{preview.excerpt}</span
				>{/if}{#if preview.startsAt}<time datetime={preview.startsAt}
					>{new Date(preview.startsAt).toLocaleString()}</time
				>{/if}</span
		></a
	>{/if}

<style>
	.message-preview {
		display: flex;
		align-items: start;
		gap: 12px;
		padding: 12px;
		border: 1px solid var(--hairline);
		border-radius: 10px;
		text-decoration: none;
		color: var(--ink);
		background: var(--paper-2);
		width: 100%;
		max-width: 380px;
	}
	.message-preview > span {
		display: grid;
		gap: 4px;
		min-width: 0;
	}
	.message-preview img {
		width: 64px;
		height: 64px;
		object-fit: cover;
		border-radius: 6px;
		flex: none;
	}
	.message-preview strong {
		font-size: 14px;
	}
	.excerpt,
	time {
		font-size: 13px;
		color: var(--ink-3);
	}
	.excerpt {
		display: -webkit-box;
		-webkit-line-clamp: 3;
		line-clamp: 3;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
</style>
