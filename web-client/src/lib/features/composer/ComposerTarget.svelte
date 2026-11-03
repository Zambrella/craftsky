<script lang="ts">
	import type { Post } from '#lib/api/client.ts';
	import Avatar from '#lib/Avatar.svelte';
	import CompactIdentity from '../social/CompactIdentity.svelte';
	let { post, fullText = false }: { post: Post; fullText?: boolean } = $props();
</script>

<blockquote
	class="composer-target"
	class:compact={!fullText}
	aria-label={fullText ? 'Replying to' : 'Quoted post'}
>
	<div class="target-identity">
		<Avatar author={post.author} /><CompactIdentity author={post.author} />
	</div>
	<p>{post.text}</p>
</blockquote>

<style>
	.composer-target {
		margin: 0;
		padding: 16px;
		border: 1px solid var(--hairline);
		border-radius: 16px;
		background: var(--paper-2);
		min-width: 0;
	}
	.target-identity {
		display: flex;
		align-items: center;
		gap: 12px;
		min-width: 0;
		margin-bottom: 12px;
	}
	p {
		margin: 0;
		font-size: 16px;
		line-height: 1.5;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}
	.compact p {
		display: -webkit-box;
		-webkit-line-clamp: 3;
		line-clamp: 3;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
</style>
