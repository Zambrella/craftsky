<script lang="ts">
	import AuthenticatedPage from '#lib/features/shell/AuthenticatedPage.svelte';
	import Metadata from '#lib/Metadata.svelte';
	import PostCard from '#lib/PostCard.svelte';
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
	<p class="breadcrumb">
		<a href={profilePath(data.post.author.did)}
			>← More from {authorName(data.post.author)}</a
		>
	</p>
	<PostCard post={data.post} detail />
	<p class="reading-note">
		You’re viewing a public CraftSky page. Join the conversation in the mobile
		app.
	</p>
{/if}
