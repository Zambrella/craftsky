<script lang="ts">
	import { confirmExternalLink } from './prompt';
	import { contextMenu } from './context-menu';
	import { onMount, onDestroy, untrack } from 'svelte';
	import { page } from '$app/state';
	import {
		CraftskyAPI,
		newOperationKey,
		type Profile,
		type PostPresentation,
		type BusinessEvent,
		type Author
	} from '#lib/api/client.ts';
	import Avatar from '#lib/Avatar.svelte';
	import CraftBadge from '#lib/CraftBadge.svelte';
	import MemberPost from './MemberPost.svelte';
	import ReportForm from '../moderation/ReportForm.svelte';
	import type { ReportSubmission } from '#lib/api/types.ts';
	import BusinessSummary from './BusinessSummary.svelte';
	import ProductCard from './ProductCard.svelte';
	import BusinessEventCard from './BusinessEventCard.svelte';
	import RichText from './RichText.svelte';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import Icon from '../shell/Icon.svelte';
	import EmptyState from './EmptyState.svelte';
	import CompactIdentity from './CompactIdentity.svelte';
	import ConfirmAction from './ConfirmAction.svelte';
	import { profileTabs, tabLabel, joinedAge } from './presentation';
	import { name, profileStyle, texture, safeLink, visible } from './model';
	let {
		api,
		session,
		id
	}: {
		api: CraftskyAPI;
		session: { id: string; did: string; handle: string };
		id: string;
	} = $props();
	const work = new Work();
	let profile = $state<Profile>();
	let identityElement = $state<HTMLElement>();
	let headerCollapsed = $state(false);
	function updateHeader() {
		headerCollapsed =
			!!identityElement && identityElement.getBoundingClientRect().bottom <= 56;
	}
	$effect(() => {
		if (identityElement) updateHeader();
	});
	let posts = $state<PostPresentation[]>([]),
		events = $state<BusinessEvent[]>([]),
		cursor = $state<string | null>(),
		mutuals = $state<Author[]>([]);
	let mutualCursor = $state<string | null | undefined>();
	let reportLink = $state<string>();
	let reportOpen = $state(false);
	let confirmBlock = $state(false);
	let reveal = $state(false);
	let tabs = $derived(
		profile
			? profileTabs(profile, profile.did === session.did)
			: ['projects', 'posts', 'comments', 'reposts']
	);
	let tab = $derived(
		[
			'posts',
			'projects',
			'comments',
			'reposts',
			'about',
			'events',
			'products'
		].includes(page.url.searchParams.get('tab') || '')
			? page.url.searchParams.get('tab')!
			: tabs[0]
	);
	let generation = 0;
	onMount(() => {
		void work.run(
			() => api.profiles.get(id),
			(value) => {
				profile = value;
			}
		);
	});
	onDestroy(() => {
		generation++;
		work.dispose();
	});
	$effect(() => {
		const current = tab;
		if (!profile) return;
		const revision = ++generation;
		cursor = undefined;
		posts = [];
		events = [];
		if (current === 'about' || current === 'products') return;
		if (current === 'events')
			untrack(
				() =>
					void work.run(
						() => api.business.profileEvents(id),
						(value) => {
							if (revision === generation) {
								events = value.items;
								cursor = value.cursor;
							}
						}
					)
			);
		else
			untrack(
				() =>
					void work.run(
						() => api.profiles.posts(id, current as 'posts'),
						(value) => {
							if (revision === generation) {
								posts = value.items;
								cursor = value.cursor;
							}
						}
					)
			);
	});
	function more() {
		if (!cursor) return;
		if (tab === 'events')
			void work.run(
				() => api.business.profileEvents(id, { cursor }),
				(value) => {
					events = [...events, ...value.items];
					cursor = value.cursor;
				}
			);
		else
			void work.run(
				() => api.profiles.posts(id, tab as 'posts', { cursor }),
				(value) => {
					posts = [...posts, ...value.items];
					cursor = value.cursor;
				}
			);
	}
	function follow() {
		if (!profile) return;
		const current = profile,
			key = newOperationKey(),
			following = profile.viewerIsFollowing;
		void work.run(
			() =>
				following
					? api.profiles.unfollow(id, key)
					: api.profiles.follow(id, key),
			(value) => {
				profile = value;
			}
		);
	}
	function relationship(action: 'mute' | 'block', confirmed = false) {
		if (!profile) return;
		const current = profile,
			key = newOperationKey();
		if (action === 'block' && !current.blocking && !confirmed) {
			confirmBlock = true;
			return;
		}
		void work.run(
			async () => {
				if (action === 'mute') {
					if (current.muted) await api.profiles.unmute(id);
					else await api.profiles.mute(id);
				} else {
					if (current.blocking) await api.profiles.unblock(id, key);
					else await api.profiles.block(id, key);
				}
				return api.profiles.get(id);
			},
			(value) => {
				profile = value;
				confirmBlock = false;
			}
		);
	}
	function report(submission: ReportSubmission) {
		const target = reportLink;
		void work.run(
			() =>
				target
					? api.profiles.reportLink(id, target, submission)
					: api.profiles.report(id, submission),
			() => {
				reportOpen = false;
				reportLink = undefined;
				work.error = 'Report submitted.';
			}
		);
	}
</script>

{#if profile ? profile.did !== session.did : ![session.did, session.handle].includes(id)}<div
		class="profile-navigation"
		class:collapsed={headerCollapsed}
		style={profile ? profileStyle(profile.customisation) : undefined}
	>
		<a
			class="icon-button"
			href="/app/home"
			aria-label="Back"
			onclick={(event) => {
				if (history.length > 1) {
					event.preventDefault();
					history.back();
				}
			}}><Icon name="back" /></a
		>
	</div>{/if}
<svelte:window onscroll={updateHeader} />
{#if profile && headerCollapsed}<div class="profile-compact-header">
		<div class="compact-profile-identity">
			<Avatar author={profile} /><CompactIdentity author={profile} />
		</div>
	</div>{/if}
<Status {work} skeleton="profile" />{#if profile}<div
		class="profile member-profile"
		style={profileStyle(profile.customisation)}
	>
		<div class="profile-band">
			{#if texture(profile.customisation)}<span
					class="profile-texture"
					style:mask-image={texture(profile.customisation)}
					aria-hidden="true"
				></span>{/if}
			{#if safeLink(profile.banner || '')}<img
					class="banner"
					src={safeLink(profile.banner!)}
					alt=""
				/>{/if}
		</div>
		<div class="profile-identity" bind:this={identityElement}>
			<Avatar
				author={{
					did: profile.did,
					handle: profile.handle,
					displayName: profile.displayName || undefined,
					avatar: profile.avatar || undefined
				}}
				large
			/>
			<h1>
				{profile.displayName ||
					(profile.handle ? '@' + profile.handle : 'Account unavailable')}
			</h1>
			{#if profile.pronouns}<p class="pronouns">{profile.pronouns}</p>{/if}
			<p class="handle">
				{#if profile.displayName}@{profile.handle}{/if}
			</p>
			{#if profile.accountType === 'business'}<p class="business-label">
					Business
				</p>{/if}
			<ul class="chips">
				{#each profile.crafts || [] as craft}<li>
						<CraftBadge {craft} />
					</li>{/each}
			</ul>
		</div>
		<div class="profile-meta stack">
			{#if !profile.isCraftskyProfile}<p class="muted non-member">
					Non CraftSky profile
				</p>{/if}
			{#if profile.moderation && !reveal}<div class="notice">
					<Icon name="shield" />
					<p>This profile has a content warning.</p>
					<button onclick={() => (reveal = true)}>Show profile</button>
				</div>{/if}
			{#if !profile.moderation || reveal}{#if profile.business}<BusinessSummary
						business={profile.business}
					/>{:else}<div class="profile-stats" aria-label="Profile summary">
						{#if profile.isCraftskyProfile && profile.createdAt}<div>
								<Icon name="calendar" size={20} /><strong
									>{joinedAge(profile.createdAt)}</strong
								><span>Joined</span>
							</div>{/if}
						{#if profile.postsLast7Days !== undefined}<div>
								<Icon name="edit" size={20} /><strong
									>{profile.postsLast7Days} posts</strong
								><span>Last 7 days</span>
							</div>{/if}
						{#if profile.projectCount !== undefined}<div>
								<Icon name="drafts" size={20} /><strong
									>{profile.projectCount}</strong
								><span>Projects</span>
							</div>{/if}
					</div>{/if}{/if}
			<div class="actions profile-actions">
				{#if profile.did === session.did}<a
						class="secondary-button"
						href="/app/profile/edit"><Icon name="edit" />Edit profile</a
					><a
						class="icon-button settings-action"
						href="/app/settings"
						aria-label="Settings"><Icon name="settings" /></a
					>
				{:else}{#if !profile.blocking && !profile.blockedBy}<button
							class="button"
							disabled={work.busy || !!work.retry}
							onclick={follow}
							>{profile.viewerIsFollowing ? 'Following' : 'Follow'}</button
						><a
							class="icon-button message-action"
							href={'/app/messages/new?did=' + encodeURIComponent(profile.did)}
							aria-label="Message"><Icon name="messages" /></a
						>{:else if profile.blocking}<button
							class="secondary-button"
							disabled={work.busy}
							onclick={() => relationship('block')}>Unblock</button
						>{/if}{/if}
				<details class="profile-menu" use:contextMenu>
					<summary class="icon-button" aria-label="More profile actions"
						><Icon name="more" /></summary
					>
					<button
						class="menu-backdrop"
						tabindex="-1"
						aria-label="Close profile menu"
						onclick={(event) => {
							event.currentTarget.closest('details')!.open = false;
						}}
					></button>
					<div class="menu-items">
						<button
							disabled={work.busy}
							onclick={() =>
								void work.run(() =>
									navigator.clipboard.writeText(
										'https://app.craftsky.social/profiles/' +
											encodeURIComponent(profile!.did)
									)
								)}><Icon name="link" />Copy link</button
						>
						<a
							href={'/app/messages/new?link=' +
								encodeURIComponent(
									'https://app.craftsky.social/profiles/' +
										encodeURIComponent(profile.did)
								)}><Icon name="send" />Send via chat</a
						>
						{#if profile.did !== session.did}<button
								onclick={() => {
									const url =
										'https://app.craftsky.social/profiles/' +
										encodeURIComponent(profile!.did);
									void work.run(() =>
										navigator.share
											? navigator.share({ url })
											: navigator.clipboard.writeText(url)
									);
								}}><Icon name="share" />Share</button
							>
							<div class="profile-menu-divider"></div>
							{#if !profile.blocking && !profile.blockedBy}<button
									disabled={work.busy || !!work.retry}
									onclick={() => relationship('mute')}
									><Icon name="mute" />{profile.muted
										? 'Unmute account'
										: 'Mute account'}</button
								>{/if}<button
								class="destructive"
								disabled={work.busy || !!work.retry}
								onclick={() => relationship('block')}
								><Icon name="block" />{profile.blocking
									? 'Unblock account'
									: 'Block account'}</button
							><button
								disabled={work.busy || !!work.retry}
								onclick={() => {
									reportLink = undefined;
									reportOpen = true;
								}}><Icon name="report" />Report profile</button
							>{/if}
					</div>
				</details>
			</div>
			{#if profile.did !== session.did && profile.mutualFollowerCount}<button
					class="mutual-link"
					onclick={() =>
						void work.run(
							() => api.profiles.mutualFollowers(id),
							(value) => {
								mutuals = value.items;
								mutualCursor = value.cursor;
							}
						)}
					><Icon name="profile" size={18} />{profile.mutualFollowerCount} mutual followers</button
				>{/if}
			{#if mutuals.length}<section
					class="mutual-list"
					aria-label="Mutual followers"
				>
					{#each mutuals as author}<a
							class="person-row"
							href={'/app/profile/' + encodeURIComponent(author.did)}
							><Avatar
								author={{
									did: author.did,
									handle: author.handle,
									displayName: author.displayName || undefined,
									avatar: author.avatar || undefined
								}}
							/><span
								><strong>{name(author)}</strong><small>@{author.handle}</small
								></span
							></a
						>{/each}
					{#if mutualCursor}<button
							disabled={work.busy}
							onclick={() =>
								void work.run(
									() =>
										api.profiles.mutualFollowers(id, { cursor: mutualCursor }),
									(value) => {
										mutuals = [
											...mutuals,
											...value.items.filter(
												(item) => !mutuals.some((old) => old.did === item.did)
											)
										];
										mutualCursor = value.cursor;
									}
								)}>Load more mutual followers</button
						>{/if}
				</section>{/if}
			{#if !profile.moderation || reveal}{#if profile.description}<p
						class="profile-bio prose"
					>
						{profile.description}
					</p>{/if}
				<ul class="profile-links">
					{#each profile.links || [] as link}{#if safeLink(link.url)}<li>
								<a
									href={safeLink(link.url)}
									target="_blank"
									rel="ugc nofollow noopener noreferrer"
									onclick={confirmExternalLink}
									><Icon name="link" size={18} />{link.title ||
										new URL(link.url).hostname}<Icon
										name="external"
										size={16}
									/></a
								>{#if profile.did !== session.did}<button
										class="icon-button"
										aria-label={'Report link ' + (link.title || link.url)}
										onclick={() => {
											reportLink = link.uri.split('/').pop();
											reportOpen = true;
										}}><Icon name="shield" size={18} /></button
									>{/if}
							</li>{/if}{/each}
				</ul>
			{/if}
		</div>
		{#if confirmBlock}<ConfirmAction
				title="Block this account?"
				detail="Blocking is public on the AT Protocol. You will no longer see or interact with each other’s content."
				confirmLabel="Block"
				busy={work.busy}
				error={work.error}
				onRetry={work.retry}
				onConfirm={() => relationship('block', true)}
				onCancel={() => (confirmBlock = false)}
			/>{/if}
		{#if reportOpen}<ReportForm
				subject={reportLink ? 'profile link' : 'profile'}
				busy={work.busy}
				error={work.error}
				onRetry={work.retry}
				onSubmit={report}
				onCancel={() => {
					reportOpen = false;
					reportLink = undefined;
				}}
			/>{/if}
		<nav class="tabs" aria-label="Profile content">
			{#each tabs as item}<a
					href={'?tab=' + item}
					aria-current={tab === item ? 'page' : undefined}>{tabLabel(item)}</a
				>{/each}
		</nav>
		{#if tab === 'about'}<section class="profile-about stack">
				{#if profile.business}<BusinessSummary
						business={profile.business}
						full
					/>{/if}{#if profile.createdAt}<div>
						<h3>Joined</h3>
						<p>
							{new Date(profile.createdAt).toLocaleDateString(undefined, {
								month: 'long',
								year: 'numeric'
							})}
						</p>
					</div>{/if}
			</section>
		{:else if tab === 'products'}<section class="product-grid">
				{#each profile.business?.products || [] as product}<ProductCard
						{product}
					/>{:else}<EmptyState
						title="Products"
						detail={profile.did === session.did
							? 'Add products to show what your business offers.'
							: 'No products shared yet.'}
						icon="business"
						>{#snippet actions()}{#if profile?.did === session.did}<a
									href="/app/settings/products">Manage products</a
								>{/if}{/snippet}</EmptyState
					>{/each}
			</section>
		{:else if tab === 'events'}<section class="events-list">
				{#each events as event}<BusinessEventCard
						{event}
					/>{:else}{#if !work.busy && !work.error}<EmptyState
							title="Upcoming events"
							detail={profile.did === session.did
								? 'Share your next workshop, market or event.'
								: 'No upcoming events.'}
							icon="calendar"
							>{#snippet actions()}{#if profile?.did === session.did}<a
										href="/app/settings/events">Manage events</a
									>{/if}{/snippet}</EmptyState
						>{/if}{/each}
			</section>{:else}{#each posts as post, i (`${'uri' in post ? post.uri : i}`)}<MemberPost
					value={post}
					{api}
					did={session.did}
					onRemove={() => {
						posts = posts.filter((value) => value !== post);
					}}
				/>{:else}{#if !work.busy}<EmptyState
						title={tabLabel(tab)}
						detail="Nothing shared here yet."
						icon="drafts"
					/>{/if}{/each}{/if}
		{#if cursor}<button class="next" disabled={work.busy} onclick={more}
				>Load more</button
			>{/if}
	</div>{/if}

<style>
	.profile-menu .destructive {
		color: var(--red-ink);
	}
	.profile-compact-header {
		position: sticky;
		top: 0;
		height: 0;
		z-index: 19;
	}
	.compact-profile-identity {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 4px 16px 4px 64px;
		height: 56px;
		background: var(--paper);
		border-bottom: 1.5px solid var(--ink);
	}

	.profile-navigation {
		position: sticky;
		top: 0;
		height: 0;
		z-index: 20;
	}
	.profile-navigation .icon-button {
		position: relative;
		top: 4px;
		left: 4px;
		width: 48px;
		height: 48px;
		background: transparent;
		border: 0;
		box-shadow: none;
		color: var(--profile-foreground, white);
		border-radius: 50%;
	}
	.profile-navigation.collapsed .icon-button {
		color: var(--ink);
	}
	.profile-band {
		position: relative;
		overflow: hidden;
	}
	.profile-texture {
		position: absolute;
		inset: 0;
		background: var(--profile-texture-tint, #fff);
		opacity: 0.18;
		mask-repeat: repeat;
		mask-mode: alpha;
		image-rendering: pixelated;
		pointer-events: none;
	}
	.profile-band .banner {
		position: relative;
	}

	.profile-meta {
		padding: 8px 16px 20px;
		gap: 16px;
	}
	.profile-bio {
		text-align: left;
		width: 100%;
		margin: 0;
		font-size: 15px;
	}
	.profile-stats {
		gap: 0;
		border: 1px solid var(--hairline);
		border-radius: 14px;
		padding: 12px 0;
		background: var(--surface);
		width: 100%;
	}
	.profile-stats > div {
		flex: 1;
		min-width: 0;
		display: grid;
		justify-items: center;
		gap: 4px;
	}
	.profile-stats > div + div {
		border-left: 1px solid var(--hairline);
	}
	.profile-stats :global(svg) {
		color: var(--primary);
	}
	.profile-stats strong {
		font-size: 16px;
		font-weight: 600;
	}
	.profile-stats span {
		font-size: 12px;
		color: var(--ink-2);
	}
	.profile-actions {
		width: 100%;
		max-width: none;
		align-self: center;
		align-items: center;
		flex-wrap: nowrap;
		gap: 12px;
	}
	.profile-actions > a,
	.profile-actions > button {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
	}
	.profile-actions .icon-button {
		width: 44px;
		height: 44px;
		border: 1.5px solid var(--ink);
		border-radius: 12px;
		background: var(--surface);
		box-shadow: 2px 2px var(--shadow);
	}
	.profile-actions summary {
		cursor: pointer;
		list-style: none;
	}
	.profile-actions summary::-webkit-details-marker {
		display: none;
	}
	.profile-actions details {
		position: relative;
	}
	.mutual-link {
		background: transparent;
		border: 0;
		box-shadow: none;
		color: var(--primary);
		padding: 0;
		display: inline-flex;
		gap: 8px;
		align-self: center;
		align-items: center;
		font-size: 14px;
	}
	.mutual-list {
		border: 1px solid var(--hairline);
		border-radius: 12px;
	}
	.mutual-list .person-row {
		flex-direction: row;
		align-items: center;
		gap: 12px;
	}
	.mutual-list small {
		display: block;
		color: var(--ink-2);
		font-size: 12px;
	}
	.profile-links {
		justify-content: flex-start;
		gap: 6px;
	}
	.profile-links li,
	.profile-links a {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.profile-links a {
		font-size: 14px;
	}
	.profile-identity :global(.profile-avatar) {
		width: 106px;
		height: 106px;
	}
	.profile-identity {
		margin-top: -53px;
	}
	.profile-identity h1 {
		margin-top: 19px;
		font-size: 28px;
	}
	.profile-identity .business-label {
		font-size: 13px;
		font-weight: 600;
	}
	.member-profile > .tabs {
		position: sticky;
		top: 0;
		z-index: 6;
		background: var(--paper);
		margin-top: 0;
		min-height: 48px;
	}
	.product-grid,
	.events-list {
		display: grid;
		grid-template-columns: 1fr;
		padding: 16px;
		gap: 12px;
	}
	.profile-about {
		padding: 16px;
		gap: 20px;
	}
	.profile-about h3 {
		font-size: 12px;
		color: var(--ink-2);
		font-weight: 500;
		margin: 0 0 4px;
	}
	.profile-about p {
		margin: 0;
		font-size: 15px;
	}

	.profile-actions > .button,
	.profile-actions > .secondary-button {
		flex: 1;
		min-height: 48px;
		font-size: 16px;
	}
	.profile-identity .chips :global(.craft-badge) {
		background: var(--profile-soft, var(--primary-soft));
		border: 0;
		padding: 6px 12px;
		font-size: 14px;
	}
	.profile-identity .chips :global(.craft-icon) {
		width: 18px;
		height: 18px;
	}
	.profile-actions > .secondary-button {
		min-height: 48px;
		padding: 10px 24px;
		font-size: 16px;
		background: var(--profile-soft, var(--primary-soft));
		border: 1.5px solid var(--ink);
		box-shadow: 3px 3px var(--shadow);
		color: var(--ink);
	}
	.profile-actions .settings-action,
	.profile-actions .message-action {
		border-radius: 50%;
		background: var(--profile-soft, var(--primary-soft));
	}
	.profile-actions summary.icon-button {
		border: 0;
		box-shadow: none;
		border-radius: 50%;
		background: var(--profile-soft, var(--primary-soft));
		width: 40px;
		height: 40px;
	}
	.profile-identity .business-label {
		color: var(--ink);
	}
</style>
