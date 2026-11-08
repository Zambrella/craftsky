<script lang="ts">
	import { getContext } from 'svelte';
	import type { CraftskyAPI } from '#lib/api/client.ts';
	import {
		publicProfileForDisplay,
		publicPostForDisplay
	} from '#lib/api/public-client.ts';
	import ProfileView from '#lib/features/social/Profile.svelte';
	const api = getContext<CraftskyAPI>('craftsky-api');
	import AuthenticatedPage from '#lib/features/shell/AuthenticatedPage.svelte';
	import Metadata from '#lib/Metadata.svelte';
	import { authorName, description, profilePath } from '#lib/public.ts';
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

	{#key data}<ProfileView
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
		/>{/key}
{/if}
