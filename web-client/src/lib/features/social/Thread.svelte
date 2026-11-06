<script lang="ts">
	import { onMount, onDestroy, tick, getContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import {
		CraftskyAPI,
		type PostPresentation,
		type CommentItem,
		type Author
	} from '#lib/api/client.ts';
	import MemberPost from './MemberPost.svelte';
	import { Publications, publicationsContext } from './publications.svelte';
	import { visible, postAddress } from './model';
	import { rememberPage } from '../shell/page-history';
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
	const publications = getContext<Publications | undefined>(
		publicationsContext
	);
	const back = getContext<(event: MouseEvent) => void>('craftsky-back');
	const historyView = rememberPage<{
		pages: number;
		sort: 'oldest' | 'newest' | 'follows';
		collapsed: string[];
	}>('thread');
	let loadedPages = $state(0);
	const work = new Work('social.Thread.action');
	let post = $state<PostPresentation>(),
		comments = $state<CommentItem[]>([]),
		people = $state<Author[]>([]),
		quotes = $state<PostPresentation[]>([]),
		cursor = $state<string | null>();
	let sort = $state<'oldest' | 'newest' | 'follows'>(
		historyView?.value?.sort || 'oldest'
	);
	let focusMessage = $state('');
	let collapsed = $state<string[]>(historyView?.value?.collapsed || []);
	$effect(() => {
		historyView?.save({ pages: loadedPages, sort, collapsed: [...collapsed] });
	});
	const focused = page.url.searchParams.get('focus') || undefined;
	const showComments = $derived(
		!section && page.url.searchParams.get('show-comments') === 'true'
	);
	let commentsSection = $state<HTMLElement>();
	onMount(() => {
		void (async () => {
			const returning = !!historyView?.value?.pages;
			const pages = historyView?.value?.pages || 1;
			await load();
			while (!dead && cursor && !work.error && loadedPages < pages)
				await load(true);
			await tick();
			if (
				!dead &&
				showComments &&
				!focused &&
				!returning &&
				post &&
				!work.error
			)
				commentsSection?.scrollIntoView({
					block: 'start',
					behavior: 'instant'
				});
			historyView?.ready();
		})();
	});
	let dead = false;
	onDestroy(() => {
		dead = true;
		work.dispose();
	});
	function load(more = false) {
		const nextCursor = more ? cursor : undefined;
		const ordering = sort;
		return work.run(
			async () => {
				const root = more ? post : await api.posts.get(did, rkey);
				if (!section && root && visible(root) && root.reply) {
					const address = postAddress(root.reply.root.uri);
					if (address)
						return {
							kind: 'redirect' as const,
							href: `/posts/${encodeURIComponent(address.did)}/${encodeURIComponent(address.rkey)}?focus=${encodeURIComponent(root.uri)}`
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
				loadedPages = more ? loadedPages + 1 : 1;
				if (result.root) publications?.observe([result.root]);
				post = result.root;
				if (result.kind === 'people') {
					people = more ? [...people, ...result.page.items] : result.page.items;
					cursor = result.page.cursor;
				} else if (result.kind === 'quotes') {
					quotes = more ? [...quotes, ...result.page.items] : result.page.items;
					cursor = result.page.cursor;
				} else {
					const values = more
						? [...comments, ...result.page.comments.items]
						: result.page.comments.items;
					comments =
						publications?.comments(values, result.root?.uri || '') || values;
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
			},
			undefined,
			'social.Thread.load'
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
				comments = comments.map((item) =>
					item.post.uri === comment.post.uri ? comment : item
				);
				comments =
					publications?.comments(comments, post?.uri || '') || comments;
				collapsed = collapsed.filter((uri) => uri !== address.uri);
			},
			undefined,
			'social.Thread.replies'
		);
	}
</script>

<div class="workspace-header">
	<a
		class="icon-button thread-back"
		href={section
			? `/posts/${encodeURIComponent(did)}/${encodeURIComponent(rkey)}`
			: '/feed'}
		aria-label="Back"
		onclick={back}><Icon name="back" /></a
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
			void goto('/feed');
		}}
	/>{/if}
{#if !section && post && visible(post) && (post.likeCount || post.repostCount || post.quoteCount)}<nav
		class="thread-interactions"
		aria-label="Post interactions"
	>
		{#if post.likeCount}<a
				href={`/posts/${encodeURIComponent(did)}/${encodeURIComponent(rkey)}/likes`}
				>{post.likeCount} {post.likeCount === 1 ? 'Like' : 'Likes'}</a
			>{/if}{#if post.repostCount}<a
				href={`/posts/${encodeURIComponent(did)}/${encodeURIComponent(rkey)}/reposts`}
				>{post.repostCount} {post.repostCount === 1 ? 'Repost' : 'Reposts'}</a
			>{/if}{#if post.quoteCount}<a
				href={`/posts/${encodeURIComponent(did)}/${encodeURIComponent(rkey)}/quotes`}
				>{post.quoteCount} {post.quoteCount === 1 ? 'Quote' : 'Quotes'}</a
			>{/if}
	</nav>{/if}
{#if section === 'likes' || section === 'reposts'}<section
		class="thread-people"
	>
		{#each people as author}<a
				class="thread-person"
				href={'/profiles/' + encodeURIComponent(author.did)}
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
	<section
		class="thread-comments"
		class:open-comments={showComments}
		bind:this={commentsSection}
		aria-label="Comments"
	>
		<div class="comments-heading">
			<h2 class="visually-hidden">Comments</h2>
			<ChoiceMenu
				label="Comment order"
				value={sort}
				disabled={work.busy}
				options={[
					{ value: 'newest', label: 'Newest', icon: 'sort' },
					{ value: 'oldest', label: 'Oldest', icon: 'filter' },
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
				href={'/compose/reply?did=' +
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
					detail
					{api}
					did={session.did}
					onRemove={() =>
						(comments = comments.filter((old) => old !== comment))}
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
				{#if (comment.replies.loaded || comment.replies.items.length > 0) && (!visible(comment.post) || !collapsed.includes(comment.post.uri))}
					<div class="replies">
						{#each comment.replies.items as reply, j (`${'uri' in reply.post ? reply.post.uri : j}`)}<MemberPost
								value={reply.post}
								detail
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
				/>{/if}{/each}
	</section>{/if}
{#if cursor}<button class="next" disabled={work.busy} onclick={() => load(true)}
		>Load more</button
	>{/if}

<style>
	.open-comments {
		min-height: calc(100dvh - var(--app-bar-height));
	}

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
		margin: 16px 16px 32px;
		padding: 8px 16px;
		min-height: 40px;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		border: 0;
		border-radius: 999px;
		color: var(--on-primary);
		background: var(--primary);
		font-weight: 600;
		text-decoration: none;
	}
	.comment-compose:hover {
		background: var(--primary-hover);
		color: var(--on-primary);
	}
	.thread-person {
		display: flex;
		gap: 12px;
		align-items: center;
		padding: 4px 16px;
		min-height: 56px;
		color: var(--ink);
		text-decoration: none;
	}

	.comment {
		padding: 0 16px;
	}
	.comment :global(.member-post) {
		border: 0;
		box-shadow: none;
		border-radius: 0;
		background: transparent;
		margin: 0 0 8px;
	}
	.comment :global(.post-card) {
		background: transparent;
		border-radius: 0;
	}
	.comment :global(.post-card) {
		padding: 8px 12px;
	}
	.comment :global(.post-actions) {
		border-top: 0;
		margin: 0 8px;
		padding: 0 0 8px;
	}
	.comment :global(.author small) {
		color: var(--ink-3);
		font-weight: 400;
	}
	.replies :global(.author small) {
		color: var(--ink-2);
	}
	.comment :global(.author) {
		margin-bottom: 12px;
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
		color: var(--primary);
		font-size: 14px;
		font-weight: 600;
		text-decoration: none;
	}
	.reply-controls {
		padding: 0 16px 8px 72px;
	}
	.reply-controls button,
	.replies > button {
		background: transparent;
		border: 0;
		box-shadow: none;
		color: light-dark(#597231, #9ebc68);
		font-size: 14px;
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
			border: 0;
			border-radius: 999px;
			box-shadow: none;
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
		margin: 0 16px 8px 32px;
		padding: 8px 0;
		border: 1px solid var(--ink-4);
		border-radius: 14px;
		background: var(--paper-2);
	}
	.replies :global(.member-post) {
		margin-left: 0;
		margin-right: 0;
	}
</style>
