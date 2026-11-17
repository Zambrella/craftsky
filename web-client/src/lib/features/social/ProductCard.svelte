<script lang="ts">
	import { confirmExternalLink } from './prompt';
	import type { BusinessProfile } from '#lib/api/types.ts';
	import { safeLink } from './model';
	import { money } from './business-model';
	import Icon from '../shell/Icon.svelte';
	import ImageCarousel from '#lib/ImageCarousel.svelte';
	let {
		product
	}: { product: NonNullable<BusinessProfile['products']>[number] } = $props();
	const href = $derived(safeLink(product.uri || ''));
	let productLink = $state<HTMLAnchorElement>();
</script>

<article class="panel product-card">
	{#if product.images?.length}<ImageCarousel
			images={product.images}
			interactive
			imageLink={href
				? {
						href,
						label: `Open ${product.title}`,
						target: '_blank',
						rel: 'ugc nofollow noopener noreferrer'
					}
				: undefined}
			onOpen={() => productLink?.click()}
		/>{/if}
	{#snippet copy()}
		<div class="product-copy">
			<h2>{product.title}</h2>
			{#if product.price}<p>{money(product.price)}</p>{/if}
		</div>
		{#if href}<Icon name="external" size={24} />{/if}
	{/snippet}
	{#if href}<a
			class="product-link"
			bind:this={productLink}
			{href}
			target="_blank"
			rel="ugc nofollow noopener noreferrer"
			onclick={confirmExternalLink}>{@render copy()}</a
		>
	{:else}<div class="product-link">{@render copy()}</div>{/if}
</article>

<style>
	.product-card {
		position: relative;
		margin: 0;
		padding: 0;
		overflow: hidden;
	}
	.product-card :global(.post-images) {
		position: relative;
		z-index: 1;
	}
	.product-card :global(.gallery) {
		margin: 0;
		border-radius: 0;
	}
	.product-card :global(.gallery > a),
	.product-card :global(.gallery img) {
		border-radius: 0;
		outline: none;
	}
	.product-link {
		display: flex;
		gap: 16px;
		align-items: center;
		padding: 16px;
		color: var(--ink);
		text-decoration: none;
		min-height: 56px;
	}
	.product-card :global(.gallery > a:focus-visible) {
		outline: 3px solid var(--primary);
		outline-offset: -4px;
	}
	a.product-link::after {
		content: '';
		position: absolute;
		inset: 0;
	}
	a.product-link:focus-visible::after {
		outline: 3px solid var(--primary);
		outline-offset: -4px;
	}
	.product-copy {
		flex: 1;
		min-width: 0;
	}
	h2 {
		font-size: 18px;
		line-height: 1.4;
		margin: 0;
		overflow-wrap: anywhere;
	}
	p {
		margin: 4px 0 0;
		font-size: 16px;
	}
</style>
