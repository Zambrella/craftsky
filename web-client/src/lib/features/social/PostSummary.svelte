<script lang="ts">
	import type { QuotePreviewPost, Post } from '#lib/api/types.ts';
	import Avatar from '#lib/Avatar.svelte';
	import SponsoredLabel from '#lib/SponsoredLabel.svelte';
	import { compactDate, fullTimestamp } from '#lib/public.ts';
	import { profileStyle, safeLink, name } from './model';
	import CompactIdentity from './CompactIdentity.svelte';
	import Icon from '../shell/Icon.svelte';
	import VideoThumbnail from './VideoThumbnail.svelte';
	let { post, href }: { post: QuotePreviewPost | Post; href: string } =
		$props();
	const image = $derived(post.images?.[0]);
	const ratio = $derived(
		image?.aspectRatio &&
			image.aspectRatio.width > 0 &&
			image.aspectRatio.height > 0
			? image.aspectRatio.width / image.aspectRatio.height
			: undefined
	);
</script>

<div class="post-summary">
	<header style={profileStyle(post.author.customisation)}>
		<a
			class="summary-author"
			href={'/profiles/' + encodeURIComponent(post.author.did)}
			aria-label={'View profile of ' + name(post.author)}
			><Avatar author={post.author} /><CompactIdentity
				author={post.author}
			/></a
		>
		<time datetime={post.createdAt} title={fullTimestamp(post.createdAt)}
			>{#if post.editedAt}<Icon name="edit" size={13} />{/if}{compactDate(
				post.createdAt
			)}</time
		>
	</header>
	<a class="summary-content" {href}>
		{#if post.externalImport?.source === 'instagram'}<small class="import-label"
				>Imported from Instagram</small
			>{/if}
		{#if post.sponsored}<SponsoredLabel />{/if}
		{#if post.project?.common.title}<h3>{post.project.common.title}</h3>{/if}
		{#if post.text}<p class="summary-text">{post.text}</p>{/if}
		{#if post.video}<VideoThumbnail
				video={post.video}
			/>{:else if image && safeLink(image.thumb || image.fullsize || '')}<div
				class="summary-image"
				style:--image-ratio={ratio || 1}
				class:unknown-ratio={!ratio}
				class:portrait={!!ratio && ratio < 1}
			>
				<img
					src={safeLink(image.thumb || image.fullsize || '')}
					alt={image.alt}
					loading="lazy"
				/>
			</div>{:else if !image && post.external && safeLink(post.external.uri)}<div
				class="summary-external"
			>
				<strong>{post.external.title}</strong>{#if post.external.description}<p>
						{post.external.description}
					</p>{/if}{#if safeLink(post.external.thumb?.url || '')}<img
						src={safeLink(post.external.thumb!.url)}
						alt=""
						loading="lazy"
					/>{/if}
			</div>{/if}
	</a>
</div>

<style>
	.post-summary {
		display: block;
		padding: 12px;
		color: var(--ink);
		text-decoration: none;
		container-type: inline-size;
	}
	header {
		display: flex;
		gap: 12px;
		align-items: center;
		min-width: 0;
	}
	header :global(.avatar) {
		flex-shrink: 0;
	}
	.summary-author,
	.summary-content {
		color: inherit;
		text-decoration: none;
	}
	.summary-author {
		display: flex;
		align-items: center;
		gap: 12px;
		min-width: 0;
		flex: 1;
	}
	.summary-content {
		display: block;
	}
	time {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		flex-shrink: 0;
		min-width: 36px;
		justify-content: flex-end;
		font-size: 12px;
		line-height: 1.45;
		color: var(--ink-2);
	}
	h3 {
		margin: 8px 0 0;
		font-size: 14px;
		font-weight: 600;
		line-height: 1.4;
	}
	.summary-text {
		margin: 8px 0 0;
		display: -webkit-box;
		-webkit-line-clamp: 4;
		line-clamp: 4;
		-webkit-box-orient: vertical;
		overflow: hidden;
		line-height: 1.5;
	}
	.summary-image {
		margin-top: 8px;
		width: 100%;
		height: clamp(
			160px,
			calc(100cqw / var(--image-ratio)),
			clamp(420px, 70dvh, 720px)
		);
		border-radius: 8px;
		border: 1px solid var(--hairline);
		overflow: hidden;
		background: var(--paper-2);
	}
	.summary-image.unknown-ratio {
		height: 320px;
		max-height: clamp(420px, 70dvh, 720px);
	}
	.summary-image img {
		display: block;
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.summary-image.portrait img {
		object-fit: contain;
	}
	.summary-external {
		margin-top: 8px;
		border: 1px solid var(--hairline);
		border-radius: 8px;
		padding: 12px;
	}
	.summary-external p {
		margin: 4px 0;
		font-size: 13px;
		color: var(--ink-2);
	}
	.summary-external img {
		width: 100%;
		max-height: 180px;
		object-fit: cover;
		margin-top: 8px;
	}
	.import-label {
		display: flex;
		align-items: center;
		gap: 4px;
		margin-top: 8px;
		color: var(--ink-2);
		font-size: 12px;
	}
</style>
