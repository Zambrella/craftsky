<script lang="ts">
	import {
		signInContext,
		protectAccountAction,
		type RequestSignIn
	} from '../browsing/account-action';
	import { reportFailure } from '../diagnostics/client';
	import type { DiagnosticOperation } from '../diagnostics/privacy';
	import { userErrorMessage } from '#lib/user-facing-error.ts';
	import SingleSelect from '#lib/SingleSelect.svelte';
	import ImageGallery from './ImageGallery.svelte';
	import Snackbar from './Snackbar.svelte';
	import { notify } from './toasts.svelte';
	import { Publications, publicationsContext } from './publications.svelte';
	import { contextMenu } from './context-menu';
	import { subscriptionsEnabled, comingSoon } from '../shell/release-config';
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
	import LikeReaction from './LikeReaction.svelte';
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
		followedBy,
		api,
		did,
		detail = false,
		flat = false,
		initialImageCid,
		savedAt,
		onRemove = () => {},
		onMoved = () => {}
	}: {
		value: PostPresentation;
		reason?: Author;
		followedBy?: Author;
		api: CraftskyAPI;
		did?: string;
		detail?: boolean;
		flat?: boolean;
		initialImageCid?: string;
		savedAt?: string;
		onRemove?: () => void;
		onMoved?: () => void;
	} = $props();
	const guest = $derived(api.audience === 'guest');
	const requestSignIn = getContext<RequestSignIn | undefined>(signInContext);
	const publications = getContext<Publications | undefined>(
		publicationsContext
	);
	const dialogId = $props.id();
	const access =
		getContext<() => SubscriptionAccess | undefined>('craftsky-access');
	const plus = $derived(
		subscriptionsEnabled() &&
			access?.()?.effectiveTier !== 'free' &&
			!!access?.()?.givesAccess
	);
	const pins = profilePins(untrack(() => api));
	let post = $state<Post | undefined>(
		untrack(() => (visible(value) ? value : undefined))
	);
	let revealed = $state(false),
		message = $state(''),
		messageError = $state(false),
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
	let galleryOpen = $state(false);
	let imageIndex = $state(0);
	let inlineImageIndex = $state(0);
	let cardElement = $state<HTMLElement>();
	let heart = $state<{ play: () => void; cancel: () => void }>();
	let revision = 0;
	$effect(() => {
		const incoming = publications?.overlay(value) || value;
		revision++;
		heart?.cancel();
		post = visible(incoming) ? incoming : undefined;
		revealed = false;
		message = '';
		messageError = false;
		busy = false;
		retry = undefined;
		reportOpen = false;
		relationshipNotice = '';
		confirmation = undefined;
		useCurrentImage = false;
	});
	$effect(() => {
		if (plus && post && post.author.did === did && pinSlot(post))
			void pins.load().catch(() => {});
	});
	onDestroy(() => {
		dead = true;
	});
	async function action<T>(
		operation: DiagnosticOperation,
		run: () => Promise<T>,
		commit: (result: T) => void,
		continuing = false
	) {
		if (busy || (retry && !continuing)) return;
		const captured = revision;
		busy = true;
		message = '';
		messageError = false;
		try {
			const result = await run();
			if (dead || captured !== revision) return;
			commit(result);
			retry = undefined;
		} catch (error) {
			if (dead || captured !== revision) return;
			heart?.cancel();
			messageError = true;
			void reportFailure(error, operation);
			message = userErrorMessage(error);
			if (
				error instanceof MutationOutcomeUnknownError ||
				(error instanceof APIError && error.retryable)
			)
				retry = () => action(operation, run, commit, true);
		} finally {
			if (!dead && captured === revision) busy = false;
		}
	}
	function toggle(kind: 'like' | 'repost') {
		if (!post || busy || retry) return;
		if (kind === 'like') {
			if (post.viewerHasLiked) heart?.cancel();
			else heart?.play();
		}
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
		void action<unknown>(
			kind === 'like' ? 'post.like' : 'post.repost',
			call,
			() => {
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
			}
		);
	}
	async function save(folderId?: string, move = false) {
		if (!post) return;
		const p = post;
		await action<unknown>(
			'post.save',
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
				notify(
					post.viewerHasSaved ? 'Post saved.' : 'Post removed from saved posts.'
				);
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
			'post.chooseFolder',
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
			'post.createFolder',
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
			if (!dead && captured === revision) {
				messageError = false;
				message = 'Link copied.';
			}
		} catch {
			if (!dead && captured === revision) {
				messageError = true;
				message = 'The link could not be copied. Please try again.';
			}
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
		return '/compose/quote?' + query;
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
			'post.relationship',
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
			}
		);
	}
	function remove() {
		if (!post) return;
		const p = post,
			key = newOperationKey();
		void action(
			'post.remove',
			() => api.posts.delete(p.author.did, p.rkey, p.cid, key),
			() => {
				publications?.remove(p.uri);
				notify('Post deleted.');
				onRemove();
			}
		);
	}
	function showGallery(event: MouseEvent) {
		const element = event.target as HTMLElement;
		if (
			!detail &&
			post &&
			!element.closest(
				'.quote-card,.external-card,.post-actions,.post-images,.notice'
			) &&
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
		if (!detail) {
			const cid = post.images[index].cid;
			void goto(
				postRoute(post) + (cid ? '?imageCid=' + encodeURIComponent(cid) : '')
			);
			return;
		}
		imageIndex = index;
		galleryOpen = true;
	}
	async function report(submission: ReportSubmission) {
		if (!post) return;
		const p = post;
		await action(
			'post.report',
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
			'post.togglePin',
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
				'post.revealPost',
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
				'post.revealQuote',
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
		bind:this={cardElement}
		class:flat-saved={!!savedAt}
		class:flat-reply={flat}
		style={profileStyle(post.author.customisation)}
		onclick={showGallery}
		role="presentation"
	>
		{#if !guest}<LikeReaction bind:this={heart} host={cardElement} />{/if}
		{#if reason && !savedAt}<a
				class="repost-attribution"
				href={'/profiles/' + encodeURIComponent(reason.did)}
				><Icon name="repost" size={16} />Reposted by {reason.displayName ||
					reason.handle}</a
			>{:else if followedBy && !savedAt}<a
				class="network-attribution"
				href={'/profiles/' + encodeURIComponent(followedBy.did)}
				><Icon name="users" size={16} />Followed by {followedBy.displayName?.trim() ||
					followedBy.handle ||
					followedBy.did}</a
			>{/if}
		{#if post.moderation && !revealed}<div class="notice">
				<p>Content warning. This post has been marked by moderation.</p>
				<button onclick={() => (revealed = true)}>Show content</button>
			</div>{:else}
			{#if savedAt}<PostSummary {post} href={postRoute(post)} />{:else}<PostCard
					post={publicPost(post)}
					{detail}
					member
					initialImageIndex={Math.max(
						0,
						post.images?.findIndex((image) => image.cid === initialImageCid) ??
							0
					)}
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
			<nav
				class="post-actions"
				aria-label="Post actions"
				onclickcapture={(event) =>
					protectAccountAction(guest, requestSignIn, event)}
			>
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
							<button data-public-action onclick={share}
								><Icon name="link" />Copy link</button
							><a
								href={'/messages/new?link=' +
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
										'post.unsave',
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
							? '/compose/reply?' +
								new URLSearchParams({ did: post.author.did, rkey: post.rkey })
							: postRoute(post) + '?show-comments=true'}
						><Icon name="comment" size={22} /><span
							>{post.reply ? 'Reply' : post.replyCount || ''}</span
						></a
					>
					{#if !post.reply}<details class="post-share" use:contextMenu>
							<summary
								aria-label="Share post"
								onclick={() => (useCurrentImage = false)}
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
								<div class="quote-menu-row">
									<a href={quoteHref()} aria-label="Quote post"
										><Icon name="quote" /><span
											>Quote post{#if post.images?.length}<small
													>Use current image ({inlineImageIndex + 1} of {post
														.images.length})</small
												>{/if}</span
										></a
									>
									{#if post.images?.length}<input
											type="checkbox"
											role="switch"
											aria-label={`Use current image (${inlineImageIndex + 1} of ${post.images.length})`}
											bind:checked={useCurrentImage}
										/>{/if}
								</div>
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
							<button data-public-action onclick={share}
								><Icon name="link" />Copy link</button
							>
							<a
								href={'/messages/new?link=' +
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
									href={`/compose/edit?did=${encodeURIComponent(post.author.did)}&rkey=${encodeURIComponent(post.rkey)}`}
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
				onRetry={retry}
				onSubmit={report}
				onCancel={() => (reportOpen = false)}
			/>{/if}
		{#if message}<Snackbar
				{message}
				warning={messageError}
				onDismiss={() => {
					message = '';
					messageError = false;
				}}
			>
				{#if retry}<button
						type="button"
						disabled={busy}
						onclick={() => retry?.()}>Retry the same action</button
					>{/if}
			</Snackbar>{/if}
		{#if !message && retry}<p class="action-recovery">
				This action still needs confirmation. <button
					type="button"
					disabled={busy}
					onclick={() => retry?.()}>Retry the same action</button
				>
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
			<SingleSelect
				label="Folder"
				value={selectedFolder}
				showRequired={false}
				options={[
					{ id: '', label: 'No folder' },
					...folders.map((folder) => ({ id: folder.id, label: folder.name }))
				]}
				disabled={busy}
				onchange={(value) => (selectedFolder = value)}
			/>
			{#if plus}{#if creatingFolder}<label
						>Folder name<input
							name="folderName"
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
					>{/if}{:else if !subscriptionsEnabled()}<button
					type="button"
					class="new-folder"
					aria-label="New folder is coming soon"
					onclick={() => comingSoon('New folder')}>New folder</button
				>{/if}
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
	{#if galleryOpen && post.images?.length}<ImageGallery
			images={post.images}
			initialIndex={imageIndex}
			onclose={() => (galleryOpen = false)}
		/>{/if}
{/if}

<style>
	.repost-attribution,
	.network-attribution {
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
		padding: 0;
		gap: 0;
		flex-wrap: nowrap;
		border-top: 1px solid var(--hairline);
	}
	.post-actions .reaction,
	.post-actions .bookmark,
	.post-actions summary {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 4px;
		min-width: 42px;
		height: 40px;
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
		font-size: 15px;
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
	.post-actions .reaction:hover,
	.post-actions .bookmark:hover,
	.post-actions summary:hover {
		background: color-mix(in srgb, var(--primary) 8%, transparent);
	}
	.post-share .menu-items {
		right: auto;
		left: 0;
	}
	.post-overflow .menu-items {
		right: -2px;
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
	.quote-menu-row {
		display: flex;
		align-items: center;
		padding-right: 16px;
	}
	.quote-menu-row a {
		flex: 1;
		min-width: 0;
	}
	.quote-menu-row a > span {
		display: grid;
		gap: 4px;
	}
	.quote-menu-row small {
		font-size: 12px;
		font-weight: 400;
		color: var(--ink-3);
	}
	.quote-menu-row input {
		flex-shrink: 0;
	}
	.save-dialog {
		width: min(500px, calc(100vw - 48px));
	}
	.save-dialog label {
		font-size: 16px;
		font-weight: 700;
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
		.member-post {
			margin-left: 16px;
			margin-right: 16px;
		}
		.post-actions {
			margin-left: 10px;
			margin-right: 10px;
		}
	}

	.member-post.flat-reply {
		margin: 0;
		padding: 0 12px 12px;
		border: 0;
		border-radius: 0;
		box-shadow: none;
		background: transparent;
	}
	.flat-reply :global(.post-card) {
		padding: 12px 0;
		border-radius: 0;
	}
	.flat-reply .post-actions {
		margin: 0 -4px;
		border-top: 0;
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
