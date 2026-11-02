<script lang="ts">
	import { onDestroy, untrack, getContext } from 'svelte';
	import { profilePins } from './pins.svelte';
	import {
		CraftskyAPI,
		APIError,
		MutationOutcomeUnknownError,
		newOperationKey,
		type PostPresentation,
		type Post,
		type SavedFolder,
		type SubscriptionAccess
	} from '#lib/api/client.ts';
	import PostCard from '#lib/PostCard.svelte';
	import { allFolders } from './saved-model';
	import ReportForm from '../moderation/ReportForm.svelte';
	import type { ReportSubmission } from '#lib/api/types.ts';
	import RichText from './RichText.svelte';
	import VideoPlayer from './VideoPlayer.svelte';
	import {
		visible,
		publicPost,
		postRoute,
		postAddress,
		profileStyle,
		safeLink,
		pinSlot,
		quoteCanRender
	} from './model';
	let {
		value,
		api,
		did,
		detail = false,
		onRemove = () => {}
	}: {
		value: PostPresentation;
		api: CraftskyAPI;
		did: string;
		detail?: boolean;
		onRemove?: () => void;
	} = $props();
	const access =
		getContext<() => SubscriptionAccess | undefined>('craftsky-access');
	const plus = $derived(access?.()?.effectiveTier !== 'free' && !!access?.());
	const pins = profilePins(untrack(() => api));
	let post = $state<Post | undefined>(
		untrack(() => (visible(value) ? value : undefined))
	);
	let revealed = $state(false),
		message = $state(''),
		busy = $state(false),
		retry = $state<(() => Promise<void>) | undefined>();
	let dead = false;
	let reportOpen = $state(false);
	let folderDialog = $state<HTMLDialogElement>(null!);
	let folders = $state<SavedFolder[]>([]);
	let gallery = $state<HTMLDialogElement>(null!);
	let imageIndex = $state(0);
	let revision = 0;
	$effect(() => {
		const incoming = value;
		revision++;
		post = visible(incoming) ? incoming : undefined;
		revealed = false;
		message = '';
		busy = false;
		retry = undefined;
		reportOpen = false;
	});
	$effect(() => {
		if (plus && post?.author.did === did && pinSlot(post))
			void pins.load().catch(() => {});
	});
	onDestroy(() => {
		dead = true;
	});
	async function action<T>(
		run: () => Promise<T>,
		commit: (result: T) => void,
		continuing = false
	) {
		if (busy || (retry && !continuing)) return;
		const captured = revision;
		busy = true;
		message = '';
		try {
			const result = await run();
			if (dead || captured !== revision) return;
			commit(result);
			retry = undefined;
		} catch (error) {
			if (dead || captured !== revision) return;
			message =
				error instanceof APIError
					? error.message
					: 'This action could not be completed.';
			if (
				error instanceof MutationOutcomeUnknownError ||
				(error instanceof APIError && error.retryable)
			)
				retry = () => action(run, commit, true);
		} finally {
			if (!dead && captured === revision) busy = false;
		}
	}
	function toggle(kind: 'like' | 'repost') {
		if (!post) return;
		const p = post,
			key = newOperationKey();
		const active = kind === 'like' ? p.viewerHasLiked : p.viewerHasReposted;
		const call =
			kind === 'like'
				? active
					? () => api.posts.unlike(p.author.did, p.rkey, key)
					: () => api.posts.like(p.author.did, p.rkey, key)
				: active
					? () => api.posts.unrepost(p.author.did, p.rkey, key)
					: () => api.posts.repost(p.author.did, p.rkey, key);
		void action<unknown>(call, () => {
			if (!post) return;
			post = {
				...post,
				...(kind === 'like'
					? {
							viewerHasLiked: !active,
							likeCount: Math.max(0, p.likeCount + (active ? -1 : 1))
						}
					: {
							viewerHasReposted: !active,
							repostCount: Math.max(0, p.repostCount + (active ? -1 : 1))
						})
			};
		});
	}
	async function save(folderId?: string, move = false) {
		if (!post) return;
		const p = post;
		await action<unknown>(
			() =>
				p.viewerHasSaved && !folderId && !move
					? api.saved.unsave(p.author.did, p.rkey)
					: api.saved.save(p.author.did, p.rkey, folderId),
			() => {
				post = {
					...p,
					viewerHasSaved: folderId || move ? true : !p.viewerHasSaved,
					viewerSavedFolderId: folderId
				};
				folderDialog?.close();
			}
		);
	}
	async function chooseFolder() {
		const captured = revision;
		try {
			const result = await allFolders(api);
			if (dead || captured !== revision) return;
			folders = result;
			folderDialog.showModal();
		} catch (error) {
			if (dead || captured !== revision) return;
			message =
				error instanceof APIError ? error.message : 'Folders could not load.';
		}
	}
	async function share() {
		if (!post) return;
		const url =
			'https://app.craftsky.social/posts/' +
			encodeURIComponent(post.author.did) +
			'/' +
			encodeURIComponent(post.rkey);
		try {
			if (navigator.share)
				await navigator.share({ url, text: post.project?.common.title });
			else {
				await navigator.clipboard.writeText(url);
				message = 'Link copied.';
			}
		} catch {
			message = 'Sharing canceled or unavailable.';
		}
	}
	function remove() {
		if (!post || !confirm('Delete this post permanently?')) return;
		const p = post,
			key = newOperationKey();
		void action(
			() => api.posts.delete(p.author.did, p.rkey, p.cid, key),
			onRemove
		);
	}
	function showGallery(event: MouseEvent) {
		const target = (event.target as HTMLElement).closest<HTMLAnchorElement>(
			'.gallery a'
		);
		if (!target || !post?.images) return;
		const index = post.images.findIndex(
			(image) => safeLink(image.fullsize || '') === target.href
		);
		if (index < 0) return;
		event.preventDefault();
		imageIndex = index;
		gallery.showModal();
	}
	async function report(submission: ReportSubmission) {
		if (!post) return;
		const p = post;
		await action(
			() => api.posts.report(p.author.did, p.rkey, submission),
			() => {
				reportOpen = false;
				message = 'Report submitted.';
			}
		);
	}
	function togglePin() {
		if (!post) return;
		const p = post,
			was = pins.isPinned(p);
		void action(
			() => pins.set(p, !was),
			() => {
				message = was ? 'Unpinned.' : 'Pinned to profile.';
			}
		);
	}
	function revealPost() {
		const address = value.uri ? postAddress(value.uri) : undefined;
		if (address)
			void action(
				() => api.posts.get(address.did, address.rkey),
				(result) => {
					if (
						visible(result) &&
						result.author.did === address.did &&
						result.rkey === address.rkey
					) {
						post = result;
						revealed = true;
					}
				}
			);
	}
	function revealQuote() {
		if (!post?.quote) return;
		const p = post,
			address = postAddress(p.quote!.uri);
		if (address)
			void action(
				() => api.posts.get(address.did, address.rkey),
				(result) => {
					if (
						visible(result) &&
						result.author.did === address.did &&
						result.rkey === address.rkey
					)
						post = {
							...p,
							quoteView: { state: 'visible', revealable: true, post: result }
						};
				}
			);
	}
</script>

{#if !post}<article class="panel protected-post">
		<p>This content is {value.availability || 'unavailable'}.</p>
		{#if value.availability === 'muted' && value.relationship?.revealable}<button
				class="secondary-button"
				onclick={revealPost}
				disabled={busy || !!retry}>Show post</button
			>{/if}
	</article>{:else}
	<div
		class="member-post"
		style={profileStyle(post.author.customisation)}
		onclick={showGallery}
		role="presentation"
	>
		{#if post.moderation && !revealed}<div class="notice">
				<p>Content warning. This post has been marked by moderation.</p>
				<button onclick={() => (revealed = true)}>Show content</button>
			</div>{:else}
			<PostCard post={publicPost(post)} {detail} member>
				{#snippet field(text, facets)}<RichText {text} {facets} />{/snippet}
				{#snippet content(text)}<RichText
						{text}
						facets={post!.facets}
					/>{/snippet}
				{#snippet media()}
					{#if post?.video}<VideoPlayer
							video={post.video}
							{api}
							did={post.author.did}
							rkey={post.rkey}
						/>{/if}{/snippet}
			</PostCard>
			{#if post.external && safeLink(post.external.uri)}<a
					class="external-card"
					href={safeLink(post.external.uri)}
					rel="ugc nofollow noreferrer"
					><strong>{post.external.title}</strong>
					{#if safeLink(post.external.thumb?.url || '')}<img
							src={safeLink(post.external.thumb!.url)}
							alt=""
							loading="lazy"
						/>{/if}
					<p>{post.external.description}</p></a
				>{/if}
			{#if post.quoteView}<aside class="quote-card">
					{#if quoteCanRender(post) && post.quoteView.post}<a
							href={postRoute(post.quoteView.post)}
							><strong
								>{post.quoteView.post.author.displayName ||
									post.quoteView.post.author.handle}</strong
							>
							{#if post.quoteView.post.sponsored}<p class="badge sponsored">
									Sponsored / commercial relationship
								</p>{/if}
							{#if post.quoteView.post.project?.common.title}<h3>
									{post.quoteView.post.project.common.title}
								</h3>{/if}
							{#if post.quoteView.post.images?.[0] && safeLink(post.quoteView.post.images[0].thumb || post.quoteView.post.images[0].fullsize || '')}<img
									src={safeLink(
										post.quoteView.post.images[0].thumb ||
											post.quoteView.post.images[0].fullsize ||
											''
									)}
									alt={post.quoteView.post.images[0].alt}
									loading="lazy"
								/>{/if}
							<p>{post.quoteView.post.text}</p></a
						>{:else}<p>
							Quoted post {post.quoteView.post?.moderation
								? 'has a content warning'
								: post.quoteView.state}.
						</p>
						{#if post.quoteView.post?.moderation}<a
								href={postRoute(post.quoteView.post)}>View quoted post</a
							>{:else if post.quoteView.state === 'muted' && post.quoteView.revealable && post.quote}<button
								disabled={busy || !!retry}
								onclick={revealQuote}>Show quoted post</button
							>{/if}{/if}
				</aside>{/if}
			<nav class="post-actions" aria-label="Post actions">
				<button
					aria-pressed={post.viewerHasLiked}
					disabled={busy || !!retry}
					onclick={() => toggle('like')}
					>♡ {post.likeCount}<span class="visually-hidden"> Like</span></button
				>
				<a href={postRoute(post)}
					>↩ {post.replyCount}<span class="visually-hidden"> Comments</span></a
				>
				<button
					aria-pressed={post.viewerHasReposted}
					disabled={busy || !!retry}
					onclick={() => toggle('repost')}
					>↻ {post.repostCount}<span class="visually-hidden">
						Repost</span
					></button
				>
				{#if !post.reply}<a
						href={`/app/compose/quote?did=${encodeURIComponent(post.author.did)}&rkey=${encodeURIComponent(post.rkey)}`}
						>Quote</a
					>{/if}
				<button
					aria-pressed={post.viewerHasSaved}
					disabled={busy || !!retry}
					onclick={() => save()}
					>{post.viewerHasSaved ? 'Saved' : 'Save'}</button
				>{#if plus}<button disabled={busy || !!retry} onclick={chooseFolder}
						>Folder</button
					>{/if}<button onclick={share}>Share</button>
				<details>
					<summary aria-label="More post actions">•••</summary>
					<div class="menu-items">
						<a href={postRoute(post) + '/likes'}>Likes</a><a
							href={postRoute(post) + '/reposts'}>Reposts</a
						><a href={postRoute(post) + '/quotes'}>Quotes</a><button
							onclick={() => (reportOpen = !reportOpen)}>Report</button
						>
						{#if post.author.did === did}<a
								href={`/app/compose/edit?did=${encodeURIComponent(post.author.did)}&rkey=${encodeURIComponent(post.rkey)}`}
								>Edit</a
							>{#if plus && pinSlot(post) && pins.value}<button
									disabled={busy || !!retry || pins.busy}
									onclick={togglePin}
									>{pins.isPinned(post) ? 'Unpin' : 'Pin'}</button
								>{/if}<button disabled={busy || !!retry} onclick={remove}
								>Delete</button
							>{/if}
					</div>
				</details>
			</nav>{/if}
		{#if reportOpen}<ReportForm
				subject={post.reply ? 'comment' : 'post'}
				{busy}
				onSubmit={report}
				onCancel={() => (reportOpen = false)}
			/>{/if}
		{#if message}<p role="status" class="notice">
				{message}{#if retry}<button disabled={busy} onclick={() => retry?.()}
						>Retry the same action</button
					><button
						onclick={() => {
							retry = undefined;
							message = '';
						}}>Dismiss</button
					>{/if}
			</p>{/if}
	</div>
	<dialog bind:this={folderDialog}>
		<h2>Save to folder</h2>
		<button onclick={() => save(undefined, true)}>Unfiled</button
		>{#each folders as folder}<button onclick={() => save(folder.id)}
				>{folder.name}</button
			>{/each}<button onclick={() => folderDialog.close()}>Cancel</button>
	</dialog>
	<dialog
		bind:this={gallery}
		class="image-dialog"
		onkeydown={(event) => {
			if (event.key === 'ArrowLeft') imageIndex = Math.max(0, imageIndex - 1);
			if (event.key === 'ArrowRight')
				imageIndex = Math.min((post?.images?.length || 1) - 1, imageIndex + 1);
		}}
	>
		<button class="close" onclick={() => gallery.close()}>Close</button
		>{#if post.images?.[imageIndex]}<img
				src={safeLink(post.images[imageIndex].fullsize || '')}
				alt={post.images[imageIndex].alt}
			/>
			<p>{post.images[imageIndex].alt}</p>
			<p>{imageIndex + 1} / {post.images.length}</p>
			{#if post.images.length > 1}<button
					onclick={() => (imageIndex = Math.max(0, imageIndex - 1))}
					disabled={imageIndex === 0}>Previous</button
				><button
					onclick={() =>
						(imageIndex = Math.min(post!.images!.length - 1, imageIndex + 1))}
					disabled={imageIndex === post.images.length - 1}>Next</button
				>{/if}{/if}
	</dialog>
{/if}
