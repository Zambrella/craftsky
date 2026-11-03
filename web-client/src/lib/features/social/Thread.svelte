<script lang="ts">
	import { onMount, onDestroy, tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import {
		CraftskyAPI,
		type PostPresentation,
		type CommentItem,
		type Author
	} from '#lib/api/client.ts';
	import MemberPost from './MemberPost.svelte';
	import { visible, postAddress } from './model';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import Icon from '../shell/Icon.svelte';
	import Avatar from '#lib/Avatar.svelte';
	import EmptyState from './EmptyState.svelte';
	import CompactIdentity from './CompactIdentity.svelte';
	import ChoiceMenu from './ChoiceMenu.svelte';
	let {
		api,
		session,
		did,
		rkey,
		section = ''
	}: {
		api: CraftskyAPI;
		session: { id: string; did: string };
		did: string;
		rkey: string;
		section?: string;
	} = $props();
	const work = new Work();
	let post = $state<PostPresentation>(),
		comments = $state<CommentItem[]>([]),
		people = $state<Author[]>([]),
		quotes = $state<PostPresentation[]>([]),
		cursor = $state<string | null>();
	let sort = $state<'oldest' | 'newest' | 'follows'>('oldest');
	let focusMessage = $state('');
	let collapsed = $state<string[]>([]);
	const focused = page.url.searchParams.get('focus') || undefined;
	onMount(() => {
		void load();
	});
	let dead = false;
	onDestroy(() => {
		dead = true;
		work.dispose();
	});
	function load(more = false) {
		const nextCursor = more ? cursor : undefined;
		const ordering = sort;
		void work.run(
			async () => {
				const root = more ? post : await api.posts.get(did, rkey);
				if (!section && root && visible(root) && root.reply) {
					const address = postAddress(root.reply.root.uri);
					if (address)
						return {
							kind: 'redirect' as const,
							href: `/app/posts/${encodeURIComponent(address.did)}/${encodeURIComponent(address.rkey)}?focus=${encodeURIComponent(root.uri)}`
						};
				}
				if (section === 'likes' || section === 'reposts')
					return {
						root,
						kind: 'people' as const,
						page: await api.posts[section](did, rkey, { cursor: nextCursor })
					};
				if (section === 'quotes')
					return {
						root,
						kind: 'quotes' as const,
						page: await api.posts.quotes(did, rkey, { cursor: nextCursor })
					};
				return {
					root,
					kind: 'comments' as const,
					page: await api.posts.comments(did, rkey, {
						sort: ordering,
						focus: more ? undefined : focused,
						cursor: nextCursor
					})
				};
			},
			(result) => {
				if (result.kind === 'redirect') {
					void goto(result.href, { replaceState: true });
					return;
				}
				post = result.root;
				if (result.kind === 'people') {
					people = more ? [...people, ...result.page.items] : result.page.items;
					cursor = result.page.cursor;
				} else if (result.kind === 'quotes') {
					quotes = more ? [...quotes, ...result.page.items] : result.page.items;
					cursor = result.page.cursor;
				} else {
					comments = more
						? [...comments, ...result.page.comments.items]
						: result.page.comments.items;
					cursor = result.page.comments.cursor;
					focusMessage =
						result.page.focus && result.page.focus.status !== 'included'
							? 'The linked comment is unavailable in this thread.'
							: '';
					if (result.page.focus?.status === 'included')
						void tick().then(() => {
							if (!dead)
								document.querySelector('.comment.focused')?.scrollIntoView({
									block: 'center',
									behavior: matchMedia('(prefers-reduced-motion: reduce)')
										.matches
										? 'instant'
										: 'smooth'
								});
						});
				}
			}
		);
	}
	function replies(comment: CommentItem) {
		if (!visible(comment.post)) return;
		const more = comment.replies.loaded && !!comment.replies.cursor;
		const address = comment.post;
		void work.run(
			() =>
				api.posts.replies(address.author.did, address.rkey, {
					cursor: comment.replies.cursor
				}),
			(result) => {
				comment.replies = {
					...result,
					items: more
						? [...comment.replies.items, ...result.items]
						: result.items
				};
				comments = [...comments];
				collapsed = collapsed.filter((uri) => uri !== address.uri);
			}
		);
	}
</script>

<div class="workspace-header">
	<a
		class="icon-button thread-back"
		href={section
			? `/app/posts/${encodeURIComponent(did)}/${encodeURIComponent(rkey)}`
			: '/app/home'}
		aria-label="Back"><Icon name="back" /></a
	>
	<h1>
		{section
			? section[0].toUpperCase() +
				section.slice(1) +
				(section === 'likes' && post && visible(post)
					? ` (${post.likeCount})`
					: '')
			: 'Post'}
	</h1>
</div>
<Status
	{work}
	skeleton={section === 'likes' || section === 'reposts' ? 'people' : 'posts'}
	reload={() => load()}
/>{#if focusMessage}<p class="notice">{focusMessage}</p>{/if}
{#if post && !section}<MemberPost
		value={post}
		{api}
		did={session.did}
		detail
		initialImageCid={page.url.searchParams.get('imageCid') || undefined}
		onRemove={() => {
			location.href = '/app/home';
		}}
	/>{/if}
{#if !section && post && visible(post) && (post.likeCount || post.repostCount || post.quoteCount)}<nav
		class="thread-interactions"
		aria-label="Post interactions"
	>
		{#if post.likeCount}<a
				href={`/app/posts/${encodeURIComponent(did)}/${encodeURIComponent(rkey)}/likes`}
				>{post.likeCount} {post.likeCount === 1 ? 'Like' : 'Likes'}</a
			>{/if}{#if post.repostCount}<a
				href={`/app/posts/${encodeURIComponent(did)}/${encodeURIComponent(rkey)}/reposts`}
				>{post.repostCount} {post.repostCount === 1 ? 'Repost' : 'Reposts'}</a
			>{/if}{#if post.quoteCount}<a
				href={`/app/posts/${encodeURIComponent(did)}/${encodeURIComponent(rkey)}/quotes`}
				>{post.quoteCount} {post.quoteCount === 1 ? 'Quote' : 'Quotes'}</a
			>{/if}
	</nav>{/if}
{#if section === 'likes' || section === 'reposts'}<section class="stack">
		{#each people as author}<a
				class="thread-person"
				href={'/app/profile/' + encodeURIComponent(author.did)}
				><Avatar
					author={{
						did: author.did,
						handle: author.handle,
						displayName: author.displayName || undefined,
						avatar: author.avatar || undefined
					}}
				/><CompactIdentity {author} /></a
			>{:else}{#if !work.busy && !work.error}<EmptyState
					title={section === 'likes' ? 'Likes' : 'Reposts'}
					detail={`No ${section} yet.`}
					icon={section === 'likes' ? 'heart' : 'repost'}
				/>{/if}{/each}
	</section>{:else if section === 'quotes'}{#each quotes as quote}<MemberPost
			value={quote}
			{api}
			did={session.did}
		/>{:else}{#if !work.busy}<p class="empty">
				No quotes yet.
			</p>{/if}{/each}{:else}
	<div class="comments-heading">
		<h2 class="visually-hidden">Comments</h2>
		<ChoiceMenu
			label="Comment order"
			value={sort}
			disabled={work.busy}
			options={[
				{ value: 'newest', label: 'Newest', icon: 'sort' },
				{ value: 'oldest', label: 'Oldest', icon: 'oldest' },
				{ value: 'follows', label: 'Follows', icon: 'following' }
			]}
			onchange={(value) => {
				sort = value as typeof sort;
				load();
			}}
		/>
	</div>
	{#if post && visible(post)}<a
			class="comment-compose"
			href={'/app/compose/reply?did=' +
				encodeURIComponent(did) +
				'&rkey=' +
				encodeURIComponent(rkey)}
			aria-label="Add a comment"><Icon name="comment" size={20} />Comment</a
		>{/if}
	{#each comments as comment, i (`${'uri' in comment.post ? comment.post.uri : i}`)}<section
			class="comment"
			class:focused={comment.placement === 'focused'}
		>
			<MemberPost
				value={comment.post}
				{api}
				did={session.did}
				onRemove={() => (comments = comments.filter((old) => old !== comment))}
			/>
			{#if visible(comment.post) && ((comment.replies.loaded && comment.replies.items.length) || comment.post.replyCount > 0)}<div
					class="reply-controls"
				>
					{#if !comment.replies.loaded || collapsed.includes(comment.post.uri)}<button
							disabled={work.busy}
							onclick={() => {
								if (comment.replies.loaded && visible(comment.post))
									collapsed = collapsed.filter(
										(uri) => uri !== comment.post.uri
									);
								else replies(comment);
							}}
							>Show {comment.post.replyCount}
							{comment.post.replyCount === 1 ? 'reply' : 'replies'}</button
						>{:else}<button
							onclick={() => {
								if (visible(comment.post))
									collapsed = [...collapsed, comment.post.uri];
							}}>Hide replies</button
						>{/if}
				</div>{/if}
			{#if comment.replies.loaded && (!visible(comment.post) || !collapsed.includes(comment.post.uri))}
				<div class="replies">
					{#each comment.replies.items as reply, j (`${'uri' in reply.post ? reply.post.uri : j}`)}<MemberPost
							value={reply.post}
							{api}
							did={session.did}
							onRemove={() => {
								comment.replies.items = comment.replies.items.filter(
									(old) => old !== reply
								);
								comments = [...comments];
							}}
						/>{/each}{#if comment.replies.cursor}<button
							onclick={() => replies(comment)}
							disabled={work.busy}>Load more replies</button
						>{/if}
				</div>{/if}
		</section>{:else}{#if !work.busy && !work.error}<EmptyState
				title="Comments"
				detail="Start the conversation with a comment."
				icon="comment"
			/>{/if}{/each}{/if}
{#if cursor}<button class="next" disabled={work.busy} onclick={() => load(true)}
		>Load more</button
	>{/if}

<style>
	.workspace-header {
		justify-content: flex-start;
		gap: 8px;
	}
	.workspace-header h1 {
		flex: 1;
	}
	.thread-back {
		display: grid;
		place-items: center;
		width: 40px;
		height: 44px;
	}
	.comments-heading {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 12px;
		padding: 8px 16px;
	}
	.comments-heading h2 {
		margin: 0;
		font-size: 18px;
	}
	.comment-compose {
		margin: 0 16px 16px;
		padding: 12px 16px;
		min-height: 48px;
		display: flex;
		align-items: center;
		gap: 10px;
		border: 1px solid var(--hairline);
		border-radius: 12px;
		color: var(--ink-2);
		background: var(--surface);
		text-decoration: none;
	}
	.thread-person {
		display: flex;
		gap: 12px;
		align-items: center;
		padding: 12px 16px;
		color: var(--ink);
		text-decoration: none;
		border-bottom: 1px solid var(--hairline);
	}

	.comment {
		padding: 12px 0;
		border-bottom: 1px dashed var(--hairline);
	}
	.comment :global(.member-post) {
		border: 0;
		box-shadow: none;
		border-radius: 0;
		background: transparent;
		margin-bottom: 0;
	}
	.comment :global(.post-card) {
		background: transparent;
		border-radius: 0;
	}
	.comment.focused {
		border-left: 4px solid var(--primary);
		background: var(--primary-soft);
	}
	.thread-interactions {
		display: flex;
		gap: 16px;
		flex-wrap: wrap;
		margin: 12px 16px;
	}
	.thread-interactions a {
		color: var(--ink);
		font-size: 14px;
		font-weight: 600;
		text-decoration: none;
	}
	.reply-controls {
		padding: 4px 28px;
	}
	.reply-controls button,
	.replies > button {
		background: transparent;
		border: 0;
		box-shadow: none;
		color: var(--primary);
		font-size: 13px;
		padding: 8px 0;
	}
	@media (max-width: 600px) {
		.comment-compose {
			position: fixed;
			z-index: 8;
			bottom: max(16px, env(safe-area-inset-bottom));
			left: 16px;
			right: 16px;
			margin: 0;
			border: 1.5px solid var(--ink);
			border-radius: 999px;
			box-shadow: 3px 3px 0 var(--shadow);
			background: var(--primary);
			color: white;
			justify-content: center;
			font-weight: 600;
			min-height: 52px;
		}
		.comment:last-child {
			margin-bottom: 64px;
		}
	}
	.replies {
		margin: 8px 16px 0 32px;
		padding-left: 4px;
		border-left: 2px solid var(--hairline);
	}
	.replies :global(.member-post) {
		margin-left: 0;
		margin-right: 0;
	}
</style>
