<script lang="ts">
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
	import { money } from './business-model';
	import RichText from './RichText.svelte';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
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
	let posts = $state<PostPresentation[]>([]),
		events = $state<BusinessEvent[]>([]),
		cursor = $state<string | null>(),
		mutuals = $state<Author[]>([]);
	let mutualCursor = $state<string | null | undefined>();
	let reportLink = $state<string>();
	let reportOpen = $state(false);
	let reveal = $state(false);
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
			: 'projects'
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
	function relationship(action: 'mute' | 'block') {
		if (!profile) return;
		const current = profile,
			key = newOperationKey();
		if (
			action === 'block' &&
			!current.blocking &&
			!confirm(
				'Block this account? You will no longer see each other’s content.'
			)
		)
			return;
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

<Status {work} />{#if profile}<div
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
		<div class="profile-identity">
			<Avatar
				author={{
					did: profile.did,
					handle: profile.handle,
					displayName: profile.displayName || undefined,
					avatar: profile.avatar || undefined
				}}
				large
			/>
			<h1>{name(profile)}</h1>
			{#if profile.pronouns}<p class="pronouns">{profile.pronouns}</p>{/if}
			<p class="handle">@{profile.handle}</p>
			{#if profile.accountType === 'business'}<p class="business-label">
					Business account
				</p>{/if}
			<ul class="chips">
				{#each profile.crafts || [] as craft}<li>
						<CraftBadge {craft} />
					</li>{/each}
			</ul>
		</div>
		<div class="profile-meta stack">
			{#if profile.moderation && !reveal}<div class="notice">
					This profile has a content warning. <button
						onclick={() => (reveal = true)}>Show profile</button
					>
				</div>{:else}<p class="prose">{profile.description || ''}</p>
				{#if profile.business}<BusinessSummary
						business={profile.business}
					/>{/if}
				<ul class="profile-links">
					{#each profile.links || [] as link}{#if safeLink(link.url)}<li>
								<a
									href={safeLink(link.url)}
									target="_blank"
									rel="ugc nofollow noopener noreferrer"
									onclick={(event) => {
										if (!confirm('Open this external website?'))
											event.preventDefault();
									}}>{link.title || new URL(link.url).hostname} ↗</a
								>{#if profile.did !== session.did}<button
										class="icon-button"
										aria-label={'Report link ' + (link.title || link.url)}
										onclick={() => {
											reportLink = link.uri.split('/').pop();
											reportOpen = true;
										}}>⚑</button
									>{/if}
							</li>{/if}{/each}
				</ul>{/if}
			<div class="profile-stats">
				{#if profile.createdAt}<span
						>Joined {new Date(profile.createdAt).toLocaleDateString(undefined, {
							month: 'short',
							year: 'numeric'
						})}</span
					>{/if}<span
					>{profile.postsLast7Days || 0} posts in the last 7 days</span
				><span>{profile.projectCount || 0} projects</span>
				{#if profile.mutualFollowerCount}<button
						onclick={() =>
							void work.run(
								() => api.profiles.mutualFollowers(id),
								(value) => {
									mutuals = value.items;
									mutualCursor = value.cursor;
								}
							)}>{profile.mutualFollowerCount} mutual followers</button
					>{/if}
			</div>
			{#if mutuals.length}<div class="panel">
					{#each mutuals as author}<a
							class="person-row"
							href={'/app/profile/' + encodeURIComponent(author.did)}
							>{name(author)}</a
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
				</div>{/if}
			<div class="actions">
				{#if profile.did === session.did}<a
						class="button"
						href="/app/profile/edit">Edit profile</a
					><a href="/app/settings/customisation">Customise</a>{:else}<button
						class="button"
						disabled={work.busy ||
							!!work.retry ||
							profile.blocking ||
							profile.blockedBy}
						onclick={follow}
						>{profile.viewerIsFollowing ? 'Following' : 'Follow'}</button
					>{#if !profile.blocking && !profile.blockedBy}<a
							class="secondary-button"
							href={'/app/messages/new?did=' + encodeURIComponent(profile.did)}
							>Message</a
						>{/if}
					<details>
						<summary>More</summary>
						<div class="menu-items">
							<button onclick={() => relationship('mute')}
								>{profile.muted ? 'Unmute' : 'Mute'}</button
							><button onclick={() => relationship('block')}
								>{profile.blocking ? 'Unblock' : 'Block'}</button
							><button
								onclick={() => {
									reportLink = undefined;
									reportOpen = !reportOpen;
								}}>Report profile</button
							><button
								onclick={() =>
									void navigator.clipboard.writeText(
										'https://app.craftsky.social/profiles/' +
											encodeURIComponent(profile!.did)
									)}>Copy profile link</button
							>
						</div>
					</details>{/if}
			</div>
		</div>
		{#if reportOpen}<ReportForm
				subject={reportLink ? 'profile link' : 'profile'}
				busy={work.busy}
				onSubmit={report}
				onCancel={() => {
					reportOpen = false;
					reportLink = undefined;
				}}
			/>{/if}
		<nav class="tabs" aria-label="Profile content">
			{#each ['projects', 'posts', 'comments', 'reposts', ...(profile.business ? ['products', 'events'] : []), 'about'] as item}<a
					href={'?tab=' + item}
					aria-current={tab === item ? 'page' : undefined}
					>{item[0].toUpperCase() + item.slice(1)}</a
				>{/each}
		</nav>
		{#if tab === 'about'}<section class="panel stack">
				<h2>About {name(profile)}</h2>
				{#if profile.business}<BusinessSummary
						business={profile.business}
						full
					/>{/if}{#if profile.createdAt}<p>
						Joined {new Date(profile.createdAt).toLocaleDateString(undefined, {
							month: 'long',
							year: 'numeric'
						})}
					</p>{/if}
				<p>{profile.postsLast7Days || 0} posts this week</p>
			</section>{:else if tab === 'products'}<section class="product-grid">
				{#each profile.business?.products || [] as product}<article
						class="panel stack"
					>
						{#if safeLink(product.image?.fullsize || '')}<img
								src={safeLink(product.image!.fullsize!)}
								alt={product.image?.alt || product.title}
								loading="lazy"
							/>{/if}
						<h2>{product.title}</h2>
						{#if product.price}<p>
								{money(product.price)}
							</p>{/if}{#if safeLink(product.uri || '')}<a
								href={safeLink(product.uri!)}
								target="_blank"
								rel="ugc nofollow noreferrer">View product ↗</a
							>{/if}
					</article>{:else}<p class="empty">No products shared yet.</p>{/each}
			</section>{:else if tab === 'events'}{#each events as event}<a
					class="panel event-card"
					href={'/app/events/' +
						encodeURIComponent(event.did) +
						'/' +
						encodeURIComponent(event.rkey)}
					><h2>{event.name}</h2>
					<time datetime={event.startsAt}
						>{new Date(event.startsAt).toLocaleString()}</time
					>
					<p>{event.summary || ''}</p></a
				>{:else}{#if !work.busy}<p class="empty">
						No upcoming events.
					</p>{/if}{/each}{:else}{#each posts as post, i (`${'uri' in post ? post.uri : i}`)}<MemberPost
					value={post}
					{api}
					did={session.did}
					onRemove={() => {
						posts = posts.filter((value) => value !== post);
					}}
				/>{:else}{#if !work.busy}<p class="empty">
						Nothing shared here yet.
					</p>{/if}{/each}{/if}
		{#if cursor}<button class="next" disabled={work.busy} onclick={more}
				>Load more</button
			>{/if}
	</div>{/if}

<style>
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
</style>
