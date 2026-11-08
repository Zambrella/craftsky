<script lang="ts">
	import { onMount, getContext } from 'svelte';
	import type { CraftskyAPI } from '#lib/api/client.ts';
	import { publicPostForDisplay } from '#lib/api/public-client.ts';
	import Thread from '#lib/features/social/Thread.svelte';
	const api = getContext<CraftskyAPI>('craftsky-api');
	let enhanced = $state(false);
	onMount(() => {
		enhanced = true;
	});
	import AuthenticatedPage from '#lib/features/shell/AuthenticatedPage.svelte';
	import Metadata from '#lib/Metadata.svelte';
	import MemberPost from '#lib/features/social/MemberPost.svelte';
	import {
		description,
		postPath,
		postTitle,
		profilePath,
		authorName
	} from '#lib/public.ts';
	import type { PageData } from './$types';
	let { data }: { data: PageData } = $props();
</script>

{#if data.authenticated}<AuthenticatedPage data={data.authenticated} />{:else}
	<Metadata
		title={postTitle(data.post)}
		description={description(data.post.text)}
		path={postPath(data.post)}
		image={data.post.images[0]?.fullsize || data.post.video?.thumbnail}
		imageAlt={data.post.images[0]?.alt || data.post.video?.alt}
	/>
	{#if enhanced}{#key `${data.post.author.did}/${data.post.rkey}`}<Thread
				{api}
				did={data.post.author.did}
				rkey={data.post.rkey}
				initialPost={publicPostForDisplay(data.post)}
			/>{/key}{:else}
		<p class="breadcrumb">
			<a href={profilePath(data.post.author.did)}
				>← More from {authorName(data.post.author)}</a
			>
		</p>
		<MemberPost value={publicPostForDisplay(data.post)} {api} detail />
	{/if}
{/if}
