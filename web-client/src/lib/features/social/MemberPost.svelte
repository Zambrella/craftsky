<script lang="ts">
	import { contextMenu } from './context-menu';
	import { goto } from '$app/navigation';
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
	import type { ReportSubmission, Author } from '#lib/api/types.ts';
	import RichText from './RichText.svelte';
	import ExpandableBody from './ExpandableBody.svelte';
	import VideoPlayer from './VideoPlayer.svelte';
	import Icon from '../shell/Icon.svelte';
	import PostSummary from './PostSummary.svelte';
	import ConfirmAction from './ConfirmAction.svelte';
	import { compactDate } from '#lib/public.ts';
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
		reason,
		api,
		did,
		detail = false,
		savedAt,
		onRemove = () => {},
		onMoved = () => {}
	}: {
		value: PostPresentation;
		reason?: Author;
		api: CraftskyAPI;
		did: string;
		detail?: boolean;
		savedAt?: string;
		onRemove?: () => void;
		onMoved?: () => void;
	} = $props();
	const dialogId = $props.id();
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
	let confirmation = $state<'delete' | 'block'>();
	let folderDialog = $state<HTMLDialogElement>(null!);
	let selectedFolder = $state('');
	let folderMode = $state<'save' | 'move'>('save');
	let creatingFolder = $state(false),
		folderName = $state('');
	let relationshipNotice = $state<'muted' | 'blocked' | ''>('');
	let useCurrentImage = $state(false);
	let folders = $state<SavedFolder[]>([]);
	let gallery = $state<HTMLDialogElement>(null!);
	let imageIndex = $state(0);
	let inlineImageIndex = $state(0);
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
		relationshipNotice = '';
		confirmation = undefined;
		useCurrentImage = false;
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
				if (savedAt && move) onMoved();
			}
		);
	}
	async function chooseFolder() {
		if (!post || busy || retry) return;
		folderMode = post.viewerHasSaved ? 'move' : 'save';
		selectedFolder = post.viewerSavedFolderId || '';
		creatingFolder = false;
		folderName = '';
		await action(
			() => (plus ? allFolders(api) : Promise.resolve([])),
			(result) => {
				folders = result;
				folderDialog.showModal();
			}
		);
	}
	function createFolder() {
		const name = folderName.trim();
		if (!name || !plus) return;
		void action(
			() => api.saved.createFolder(name),
			(folder) => {
				folders = [...folders, folder];
				selectedFolder = folder.id;
				creatingFolder = false;
				folderName = '';
			}
		);
	}
	function canonicalLink() {
		return post
			? 'https://app.craftsky.social/posts/' +
					encodeURIComponent(post.author.did) +
					'/' +
					encodeURIComponent(post.rkey)
			: '';
	}
	async function share() {
		const captured = revision;
		try {
			await navigator.clipboard.writeText(canonicalLink());
			if (!dead && captured === revision) message = 'Link copied.';
		} catch {
			if (!dead && captured === revision)
				message = 'The link could not be copied. Please try again.';
		}
	}
	function quoteHref() {
		if (!post) return '';
		const query = new URLSearchParams({
			did: post.author.did,
			rkey: post.rkey
		});
		const image = post.images?.[inlineImageIndex];
		if (useCurrentImage && image?.cid) query.set('imageCid', image.cid);
		return '/app/compose/quote?' + query;
	}
	function relationship(kind: 'mute' | 'block', confirmed = false) {
		if (!post || post.author.did === did) return;
		const current = post;
		if (kind === 'block' && !current.author.blocking && !confirmed) {
			confirmation = 'block';
			return;
		}
		const key = newOperationKey();
		void action(
			async () => {
				if (kind === 'mute')
					return current.author.muted
						? api.profiles.unmute(current.author.did)
						: api.profiles.mute(current.author.did);
				return current.author.blocking
					? api.profiles.unblock(current.author.did, key)
					: api.profiles.block(current.author.did, key);
			},
			() => {
				if (current.author[kind === 'mute' ? 'muted' : 'blocking']) {
					post = {
						...current,
						author: {
							...current.author,
							[kind === 'mute' ? 'muted' : 'blocking']: false
						}
					};
				} else {
					confirmation = undefined;
					relationshipNotice = kind === 'mute' ? 'muted' : 'blocked';
					post = undefined;
				}
				window.dispatchEvent(new Event('craftsky-refresh'));
			}
		);
	}
	function remove() {
		if (!post) return;
		const p = post,
			key = newOperationKey();
		void action(
			() => api.posts.delete(p.author.did, p.rkey, p.cid, key),
			onRemove
		);
	}
	function showGallery(event: MouseEvent) {
		const element = event.target as HTMLElement;
		if (
			!detail &&
			post &&
			element.closest('.prose') &&
			!element.closest('a,button,input,textarea,select,summary') &&
			!window.getSelection()?.toString() &&
			event.button === 0 &&
			!event.metaKey &&
			!event.ctrlKey &&
			!event.shiftKey &&
			!event.altKey
		) {
			void goto(postRoute(post));
			return;
		}
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
						relationshipNotice = '';
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
		<p>
			This content is {relationshipNotice ||
				value.availability ||
				'unavailable'}.
		</p>
		{#if relationshipNotice === 'muted' || (value.availability === 'muted' && value.relationship?.revealable)}<button
				class="secondary-button"
				onclick={revealPost}
				disabled={busy || !!retry}>Show post</button
			>{/if}
	</article>{:else}
	<div
		class="member-post"
		class:flat-saved={!!savedAt}
		style={profileStyle(post.author.customisation)}
		onclick={showGallery}
		role="presentation"
	>
		{#if reason && !savedAt}<a
				class="repost-attribution"
				href={'/app/profile/' + encodeURIComponent(reason.did)}
				><Icon name="repost" size={16} />Reposted by {reason.displayName ||
					reason.handle}</a
			>{/if}
		{#if post.moderation && !revealed}<div class="notice">
				<p>Content warning. This post has been marked by moderation.</p>
				<button onclick={() => (revealed = true)}>Show content</button>
			</div>{:else}
			{#if savedAt}<PostSummary {post} href={postRoute(post)} />{:else}<PostCard
					post={publicPost(post)}
					{detail}
					member
					onImageChange={(index) => {
						inlineImageIndex = index;
					}}
				>
					{#snippet field(text, facets)}<RichText {text} {facets} />{/snippet}
					{#snippet content(text)}{#key post!.uri}<ExpandableBody
								{text}
								facets={post!.facets}
								collapse={!detail}
							/>{/key}{/snippet}
					{#snippet media()}
						{#if post?.video}<VideoPlayer
								video={post.video}
								{api}
								did={post.author.did}
								rkey={post.rkey}
							/>{/if}{/snippet}
				</PostCard>{/if}
			{#if !savedAt && post.external && safeLink(post.external.uri)}<a
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
			{#if !savedAt && post.quoteView}<aside
					class="quote-card"
					aria-label="Quoted post"
				>
					{#if quoteCanRender(post) && post.quoteView.post}<PostSummary
							post={post.quoteView.post}
							href={postRoute(post.quoteView.post)}
						/>{:else}<div class="quote-placeholder">
							<p>
								Quoted post {post.quoteView.post?.moderation
									? 'has a content warning'
									: post.quoteView.state}.
							</p>
							{#if post.quoteView.post?.moderation}<a
									href={postRoute(post.quoteView.post)}>View quoted post</a
								>{:else if post.quoteView.state === 'muted' && post.quoteView.revealable && post.quote}<button
									disabled={busy || !!retry}
									onclick={revealQuote}>Show quoted post</button
								>{/if}
						</div>{/if}
				</aside>{/if}
			<nav class="post-actions" aria-label="Post actions">
				{#if savedAt}<time class="saved-date" datetime={savedAt}
						>{compactDate(savedAt)}</time
					><span class="action-spacer"></span>
					<details class="post-overflow" use:contextMenu>
						<summary aria-label="Saved post actions"
							><Icon name="more" size={22} /></summary
						>
						<button
							class="menu-backdrop"
							type="button"
							tabindex="-1"
							aria-label="Close post menu"
							onclick={(event) => {
								event.currentTarget.closest('details')!.open = false;
							}}
						></button>
						<div class="menu-items">
							<button onclick={share}><Icon name="link" />Copy link</button><a
								href={'/app/messages/new?link=' +
									encodeURIComponent(canonicalLink())}
								><Icon name="send" />Send via chat</a
							>
							<div class="menu-divider"></div>
							{#if plus}<button
									disabled={busy || !!retry}
									onclick={chooseFolder}
									><Icon name="folder" />Move to folder</button
								>{/if}<button
								disabled={busy || !!retry}
								onclick={() => {
									const current = post!;
									void action(
										() => api.saved.unsave(current.author.did, current.rkey),
										onRemove
									);
								}}><Icon name="saved" />Remove from saved posts</button
							>
						</div>
					</details>{:else}
					<button
						class="reaction"
						class:liked={post.viewerHasLiked}
						aria-label="Like"
						aria-pressed={post.viewerHasLiked}
						disabled={busy || !!retry}
						onclick={() => toggle('like')}
						><Icon name="heart" size={22} filled={post.viewerHasLiked} /><span
							>{post.likeCount || ''}</span
						></button
					>
					<a
						class="reaction"
						aria-label={post.reply ? 'Reply' : 'Comments'}
						href={post.reply || detail
							? '/app/compose/reply?' +
								new URLSearchParams({ did: post.author.did, rkey: post.rkey })
							: postRoute(post)}
						><Icon name="comment" size={22} /><span
							>{post.reply ? 'Reply' : post.replyCount || ''}</span
						></a
					>
					{#if !post.reply}<details class="post-share" use:contextMenu>
							<summary aria-label="Share post"
								><Icon name="repost" size={22} /><span
									>{post.repostCount + post.quoteCount || ''}</span
								></summary
							>
							<button
								class="menu-backdrop"
								type="button"
								tabindex="-1"
								aria-label="Close post menu"
								onclick={(event) => {
									event.currentTarget.closest('details')!.open = false;
								}}
							></button>
							<div class="menu-items">
								<button
									aria-pressed={post.viewerHasReposted}
									disabled={busy || !!retry}
									onclick={() => toggle('repost')}
									><Icon name="repost" />{post.viewerHasReposted
										? 'Undo repost'
										: 'Repost'}</button
								>
								<a href={quoteHref()}><Icon name="quote" />Quote post</a
								>{#if post.images?.length}<label class="quote-image-option"
										><input
											type="checkbox"
											bind:checked={useCurrentImage}
										/>Quote image {inlineImageIndex + 1} of {post.images
											.length}</label
									>{/if}
							</div>
						</details>{/if}
					<span class="action-spacer"></span>
					<button
						class="bookmark"
						aria-label={post.viewerHasSaved ? 'Saved' : 'Save'}
						title={post.viewerHasSaved
							? 'Remove from saved posts'
							: 'Save post'}
						aria-pressed={post.viewerHasSaved}
						disabled={busy || !!retry}
						onclick={() => (post!.viewerHasSaved ? save() : chooseFolder())}
						><Icon
							name="saved"
							size={22}
							filled={post.viewerHasSaved}
						/></button
					>
					<details class="post-overflow" use:contextMenu>
						<summary aria-label="More post actions"
							><Icon name="more" size={22} /></summary
						>
						<button
							class="menu-backdrop"
							type="button"
							tabindex="-1"
							aria-label="Close post menu"
							onclick={(event) => {
								event.currentTarget.closest('details')!.open = false;
							}}
						></button>
						<div class="menu-items">
							{#if post.reply && post.likeCount > 0}<a
									href={postRoute(post) + '/likes'}
									><Icon name="users" />Likes</a
								>
								<div class="menu-divider"></div>{/if}
							<button onclick={share}><Icon name="link" />Copy link</button>
							<a
								href={'/app/messages/new?link=' +
									encodeURIComponent(canonicalLink())}
								><Icon name="send" />Send via chat</a
							>
							{#if post.author.did !== did}<div class="menu-divider"></div>
								<button
									disabled={busy || !!retry}
									onclick={() => relationship('mute')}
									><Icon name="mute" />{post.author.muted
										? 'Unmute account'
										: 'Mute account'}</button
								><button
									class="delete-action"
									disabled={busy || !!retry}
									onclick={() => relationship('block')}
									><Icon name="block" />{post.author.blocking
										? 'Unblock account'
										: 'Block account'}</button
								><button onclick={() => (reportOpen = true)}
									><Icon name="report" />Report {post.reply
										? post.reply.parent.uri === post.reply.root.uri
											? 'comment'
											: 'reply'
										: 'post'}</button
								>{/if}
							{#if post.author.did === did}<a
									href={`/app/compose/edit?did=${encodeURIComponent(post.author.did)}&rkey=${encodeURIComponent(post.rkey)}`}
									><Icon name="edit" />Edit</a
								>
								{#if plus && pinSlot(post) && pins.value}<button
										disabled={busy || !!retry || pins.busy}
										onclick={togglePin}
										><Icon name="pin" />{pins.isPinned(post)
											? 'Unpin'
											: 'Pin'}</button
									>{/if}
								<button
									class="delete-action"
									disabled={busy || !!retry}
									onclick={() => (confirmation = 'delete')}
									><Icon name="trash" />Delete</button
								>{/if}
						</div>
					</details>
				{/if}
			</nav>{/if}
		{#if confirmation}<ConfirmAction
				title={confirmation === 'block'
					? 'Block this account?'
					: `Delete ${post.reply ? (post.reply.parent.uri === post.reply.root.uri ? 'comment' : 'reply') : 'post'}?`}
				detail={confirmation === 'block'
					? 'Blocking is public on the AT Protocol. You will no longer see or interact with each other’s content.'
					: `This removes the ${post.reply ? (post.reply.parent.uri === post.reply.root.uri ? 'comment' : 'reply') : 'post'} from CraftSky.`}
				confirmLabel={confirmation === 'block' ? 'Block' : 'Delete'}
				{busy}
				error={message}
				onRetry={retry}
				onConfirm={() =>
					confirmation === 'block' ? relationship('block', true) : remove()}
				onCancel={() => (confirmation = undefined)}
			/>{/if}
		{#if reportOpen}<ReportForm
				subject={post.reply
					? post.reply.parent.uri === post.reply.root.uri
						? 'comment'
						: 'reply'
					: 'post'}
				{busy}
				error={message}
				onRetry={retry}
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
	<dialog
		bind:this={folderDialog}
		class="save-dialog"
		aria-labelledby={dialogId + '-save-title'}
		oncancel={(event) => {
			if (busy) event.preventDefault();
		}}
	>
		<form
			class="stack"
			onsubmit={(event) => {
				event.preventDefault();
				void save(selectedFolder || undefined, true);
			}}
		>
			<h2 id={dialogId + '-save-title'}>
				{folderMode === 'move' ? 'Move saved post' : 'Save post'}
			</h2>
			<label
				>Folder<select bind:value={selectedFolder} disabled={busy}
					><option value="">No folder</option>{#each folders as folder}<option
							value={folder.id}>{folder.name}</option
						>{/each}</select
				></label
			>
			{#if plus}{#if creatingFolder}<label
						>Folder name<input
							bind:value={folderName}
							maxlength="80"
							disabled={busy}
						/></label
					><button
						type="button"
						disabled={busy || !folderName.trim()}
						class="button"
						onclick={createFolder}>Create folder</button
					>{:else}<button
						type="button"
						disabled={busy}
						class="new-folder"
						onclick={() => (creatingFolder = true)}>New folder</button
					>{/if}{/if}
			{#if message}<p role="status">
					{message}{#if retry}<button
							type="button"
							disabled={busy}
							onclick={() => retry?.()}>Retry the same action</button
						>{/if}
				</p>{/if}
			<div class="dialog-actions">
				<button
					type="button"
					disabled={busy}
					onclick={() => folderDialog.close()}>Cancel</button
				><button class="button" disabled={busy || !!retry}
					>{folderMode === 'move' ? 'Move' : 'Save post'}</button
				>
			</div>
		</form>
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

<style>
	.repost-attribution {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 12px 12px 0;
		color: var(--ink-2);
		font-size: 13px;
		font-weight: 600;
		text-decoration: none;
	}
	.member-post {
		margin: 12px 16px 8px;
		border: 1.5px solid var(--ink);
		border-radius: 14px;
		background: var(--surface);
		box-shadow: 3px 3px 0 var(--shadow);
		position: relative;
		min-width: 0;
	}
	.member-post :global(.post-card) {
		margin: 0;
		border: 0;
		border-radius: 14px 14px 0 0;
		box-shadow: none;
		background: transparent;
	}
	.post-actions {
		margin: 0 12px;
		padding: 4px 0;
		gap: 4px;
		flex-wrap: nowrap;
		border-top: 1px dashed var(--hairline);
	}
	.post-actions .reaction,
	.post-actions .bookmark,
	.post-actions summary {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		min-width: 42px;
		height: 44px;
		padding: 8px;
		border: 0;
		border-radius: 8px;
		background: transparent;
		color: var(--ink);
		box-shadow: none;
	}
	.post-actions .reaction span,
	.post-share summary span {
		font-variant-numeric: tabular-nums;
		font-size: 13px;
		min-width: 8px;
	}
	.post-actions .liked {
		color: var(--red-ink);
		background: transparent;
	}
	.post-actions .bookmark[aria-pressed='true'] {
		color: var(--primary);
		background: transparent;
	}
	.action-spacer {
		flex: 1;
	}
	.post-actions details {
		position: relative;
		margin: 0;
	}
	.post-actions summary {
		cursor: pointer;
		list-style: none;
	}
	.post-actions summary::-webkit-details-marker {
		display: none;
	}
	.post-actions summary:hover {
		background: var(--paper-2);
	}
	.post-share .menu-items {
		right: auto;
		left: 0;
	}
	.post-overflow .menu-items {
		right: -2px;
	}
	.menu-items {
		min-width: 215px;
		z-index: 15;
	}
	.menu-items a,
	.menu-items button {
		display: flex;
		align-items: center;
		gap: 10px;
		font-size: 15px;
		font-weight: 400;
		padding: 8px 10px;
		white-space: nowrap;
	}
	.menu-items :global(svg) {
		flex-shrink: 0;
		width: 20px;
		height: 20px;
	}
	.post-actions .delete-action {
		color: var(--red-ink);
	}
	.quote-card {
		margin: 0 12px 8px;
		padding: 0;
		border: 1px solid var(--hairline);
		border-radius: 8px;
		background: var(--paper-2);
		overflow: hidden;
	}
	.quote-placeholder {
		padding: 12px;
	}
	.menu-backdrop {
		display: none;
	}
	.menu-divider {
		border-top: 1px solid var(--hairline);
		margin: 6px 0;
	}
	.quote-image-option {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 8px 10px;
		font-size: 13px;
		color: var(--ink-2);
	}
	.quote-image-option input {
		width: 18px;
		height: 18px;
		margin: 0;
	}
	.save-dialog {
		width: min(500px, calc(100vw - 48px));
	}
	.save-dialog label {
		font-size: 16px;
		font-weight: 700;
	}
	.save-dialog select {
		font-weight: 400;
		border-color: var(--ink-3);
		box-shadow: none;
	}
	.save-dialog .new-folder {
		border: 0;
		background: transparent;
		box-shadow: none;
		color: var(--primary);
	}
	.dialog-actions > button:first-child {
		background: transparent;
		border: 0;
		box-shadow: none;
		color: var(--primary);
	}
	.save-dialog h2 {
		margin: 0;
		font-size: 20px;
	}
	.dialog-actions {
		display: flex;
		justify-content: flex-end;
		gap: 12px;
	}
	.external-card img {
		max-height: 240px;
		width: 100%;
		object-fit: cover;
		margin-top: 8px;
	}
	.protected-post {
		min-height: 110px;
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 20px;
	}
	@media (max-width: 600px) {
		.menu-backdrop {
			display: block;
			position: fixed;
			inset: 0;
			z-index: 39;
			border: 0;
			border-radius: 0;
			background: rgb(0 0 0 / 40%);
			width: 100%;
			height: 100dvh;
		}
		.post-actions .menu-items {
			position: fixed;
			z-index: 40;
			inset: auto 0 0 !important;
			width: 100%;
			max-height: 75dvh !important;
			padding: 12px 16px max(24px, env(safe-area-inset-bottom));
			border-radius: 16px 16px 0 0;
			border: 0;
			box-shadow: none;
			background: var(--paper-2);
		}
		.post-actions .menu-items::before {
			content: '';
			display: block;
			width: 32px;
			height: 4px;
			margin: 0 auto 12px;
			border-radius: 4px;
			background: var(--ink-3);
		}
		.menu-items a,
		.menu-items button {
			padding: 14px 10px;
			min-height: 48px;
		}
		.member-post {
			margin-left: 16px;
			margin-right: 16px;
		}
		.post-actions {
			margin-left: 10px;
			margin-right: 10px;
		}
	}

	.member-post.flat-saved {
		margin: 0;
		padding: 0 12px;
		border: 0;
		border-bottom: 1px solid var(--hairline);
		border-radius: 0;
		box-shadow: none;
		background: transparent;
	}
	.flat-saved :global(.post-card) {
		padding: 12px 0;
		border-radius: 0;
	}
	.flat-saved .post-actions {
		margin: 0 -4px;
		border-top: 0;
	}
	.saved-date {
		color: var(--ink-2);
		font-size: 12px;
	}
</style>
