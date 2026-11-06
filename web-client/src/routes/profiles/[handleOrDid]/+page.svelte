<script lang="ts">
	import { onMount, getContext } from 'svelte';
	import type { CraftskyAPI } from '#lib/api/client.ts';
	import {
		publicProfileForDisplay,
		publicPostForDisplay
	} from '#lib/api/public-client.ts';
	import BusinessSummary from '#lib/features/social/BusinessSummary.svelte';
	import ProductCard from '#lib/features/social/ProductCard.svelte';
	import BusinessEventCard from '#lib/features/social/BusinessEventCard.svelte';
	import ProfileView from '#lib/features/social/Profile.svelte';
	const api = getContext<CraftskyAPI>('craftsky-api');
	let enhanced = $state(false);
	onMount(() => {
		enhanced = true;
	});
	import AuthenticatedPage from '#lib/features/shell/AuthenticatedPage.svelte';
	import Metadata from '#lib/Metadata.svelte';
	import PostCard from '#lib/PostCard.svelte';
	import Avatar from '#lib/Avatar.svelte';
	import CraftBadge from '#lib/CraftBadge.svelte';
	import {
		authorName,
		description,
		profilePath,
		safeURL
	} from '#lib/public.ts';
	import type { PageData } from './$types';
	let { data }: { data: PageData } = $props();
</script>

<svelte:head
	>{#if !data.authenticated && data.cursor}<meta
			name="robots"
			content="noindex,follow"
		/>{/if}</svelte:head
>

{#if data.authenticated}<AuthenticatedPage data={data.authenticated} />{:else}
	<Metadata
		title={authorName(data.profile)}
		description={description(
			data.profile.description ||
				`Explore ${authorName(data.profile)}’s posts and projects on CraftSky.`
		)}
		path={profilePath(data.profile.did)}
		image={data.profile.avatar}
		imageAlt={authorName(data.profile)}
	/>

	{#if enhanced}{#key data}<ProfileView
				{api}
				id={data.profile.did}
				initial={{
					profile: publicProfileForDisplay(data.profile),
					posts: {
						...data.posts,
						items: data.posts.items.map(publicPostForDisplay)
					},
					events: data.events,
					tab: data.tab
				}}
			/>{/key}{:else}
		<section class="profile">
			<div class="profile-band" aria-hidden="true">
				{#if safeURL(data.profile.banner)}<img
						class="banner"
						src={safeURL(data.profile.banner)}
						alt=""
						width="800"
						height="128"
					/>{/if}
			</div>
			<div class="profile-identity">
				<Avatar author={data.profile} large />
				<h1>{authorName(data.profile)}</h1>
				{#if data.profile.pronouns}<p class="pronouns">
						{data.profile.pronouns}
					</p>{/if}
				{#if data.profile.business}<p class="business-label">
						Business account
					</p>{/if}
				<p class="handle">@{data.profile.handle}</p>
				{#if data.profile.crafts.length}<ul class="chips" aria-label="Crafts">
						{#each data.profile.crafts as craft}<li>
								<CraftBadge {craft} />
							</li>{/each}
					</ul>{/if}
			</div>
			<div class="profile-meta">
				{#if data.profile.description}<p class="prose">
						{data.profile.description}
					</p>{/if}
				{#if data.profile.links.length}<ul class="profile-links">
						{#each data.profile.links as link}{#if safeURL(link.url)}<li>
									<a href={safeURL(link.url)} rel="ugc nofollow noreferrer"
										>{link.title || link.url}<span aria-hidden="true">
											↗</span
										></a
									>
								</li>{/if}{/each}
					</ul>{/if}
			</div>
		</section>
		<nav class="tabs" aria-label="Profile content">
			{#each ['posts', 'projects', 'comments', 'reposts', ...(data.profile.business ? ['about', 'products', 'events'] : [])] as tab}
				<a
					href={profilePath(data.profile.did) +
						'?' +
						new URLSearchParams({ tab })}
					aria-current={data.tab === tab ? 'page' : undefined}
					>{tab === 'events'
						? 'Upcoming events'
						: tab[0].toUpperCase() + tab.slice(1)}</a
				>
			{/each}
		</nav>
		{#if data.tab === 'about'}
			{#if data.profile.businessProfile}<BusinessSummary
					business={data.profile.businessProfile}
					full
				/>{/if}
		{:else if data.tab === 'products'}
			{#each data.profile.businessProfile?.products || [] as product}<ProductCard
					{product}
				/>{:else}<p>No products shared yet.</p>{/each}
		{:else if data.tab === 'events'}
			{#each data.events.items as event}<BusinessEventCard {event} />{:else}<p>
					No upcoming events.
				</p>{/each}
			{#if data.events.cursor}<a
					href={profilePath(data.profile.did) +
						'?' +
						new URLSearchParams({ tab: 'events', cursor: data.events.cursor })}
					>More events</a
				>{/if}
		{:else}
			<section aria-label={data.tab === 'projects' ? 'Projects' : 'Posts'}>
				{#each data.posts.items as post}<PostCard {post} />{:else}<p
						class="empty"
					>
						No {data.tab} to show yet.
					</p>{/each}
				{#if data.posts.cursor}<a
						class="button next"
						href={profilePath(data.profile.did) +
							'?' +
							new URLSearchParams({ tab: data.tab, cursor: data.posts.cursor })}
						>Older {data.tab} →</a
					>{/if}
			</section>
		{/if}
	{/if}
{/if}
