<script lang="ts">
	import { onMount, onDestroy, tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import {
		CraftskyAPI,
		type PostPresentation,
		type CommentItem,
		type Author,
		type ReplyPage
	} from '#lib/api/client.ts';
	import MemberPost from './MemberPost.svelte';
	import { visible, name, postAddress } from './model';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
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
								document
									.querySelector('.comment.focused')
									?.scrollIntoView({
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
		if (!visible(comment.post) || !comment.replies.cursor) return;
		const address = comment.post;
		void work.run(
			() =>
				api.posts.replies(address.author.did, address.rkey, {
					cursor: comment.replies.cursor
				}),
			(result) => {
				comment.replies = {
					...result,
					items: [...comment.replies.items, ...result.items]
				};
				comments = [...comments];
			}
		);
	}
</script>

<div class="workspace-header">
	<h1>{section ? section[0].toUpperCase() + section.slice(1) : 'Post'}</h1>
	<a href="/app/home">Home</a>
</div>
<Status {work} />{#if focusMessage}<p class="notice">{focusMessage}</p>{/if}
{#if post}<MemberPost
		value={post}
		{api}
		did={session.did}
		detail
		onRemove={() => {
			location.href = '/app/home';
		}}
	/>{/if}
{#if section === 'likes' || section === 'reposts'}<section class="stack">
		{#each people as author}<a
				class="person-row panel"
				href={'/app/profile/' + encodeURIComponent(author.did)}
				><strong>{name(author)}</strong><span>@{author.handle}</span></a
			>{:else}{#if !work.busy}<p class="empty">No {section} yet.</p>{/if}{/each}
	</section>{:else if section === 'quotes'}{#each quotes as quote}<MemberPost
			value={quote}
			{api}
			did={session.did}
		/>{:else}{#if !work.busy}<p class="empty">
				No quotes yet.
			</p>{/if}{/each}{:else}
	<div class="workspace-header">
		<h2>Comments</h2>
		<label class="visually-hidden" for="comment-sort">Comment order</label
		><select
			id="comment-sort"
			bind:value={sort}
			disabled={work.busy}
			onchange={() => load()}
			><option value="oldest">Oldest first</option><option value="newest"
				>Newest first</option
			><option value="follows">People I follow</option></select
		>
	</div>
	{#if post && visible(post)}<a
			class="button next"
			href={'/app/compose/reply?did=' +
				encodeURIComponent(did) +
				'&rkey=' +
				encodeURIComponent(rkey)}>Add a comment</a
		>{/if}
	{#each comments as comment, i (`${'uri' in comment.post ? comment.post.uri : i}`)}<section
			class="comment"
			class:focused={comment.placement === 'focused'}
		>
			<MemberPost
				value={comment.post}
				{api}
				did={session.did}
			/>{#if visible(comment.post)}<a
					class="reply-link"
					href={'/app/compose/reply?did=' +
						encodeURIComponent(comment.post.author.did) +
						'&rkey=' +
						encodeURIComponent(comment.post.rkey)}>Reply to comment</a
				>{/if}
			<div class="replies">
				{#each comment.replies.items as reply, j (`${'uri' in reply.post ? reply.post.uri : j}`)}<MemberPost
						value={reply.post}
						{api}
						did={session.did}
					/>{/each}{#if comment.replies.cursor}<button
						onclick={() => replies(comment)}
						disabled={work.busy}>Show more replies</button
					>{/if}
			</div>
		</section>{:else}{#if !work.busy}<p class="empty">
				Start the conversation.
			</p>{/if}{/each}{/if}
{#if cursor}<button class="next" disabled={work.busy} onclick={() => load(true)}
		>Load more</button
	>{/if}
