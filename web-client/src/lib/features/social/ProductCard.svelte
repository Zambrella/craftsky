<script lang="ts">
	import type { BusinessProfile } from '#lib/api/types.ts';
	import { safeLink } from './model';
	import { money } from './business-model';
	import Icon from '../shell/Icon.svelte';
	let {
		product
	}: { product: NonNullable<BusinessProfile['products']>[number] } = $props();
	const href = $derived(safeLink(product.uri || ''));
</script>

{#snippet content()}
	{#if safeLink(product.image?.thumb || product.image?.fullsize || '')}<img
			src={safeLink(product.image?.thumb || product.image?.fullsize || '')}
			alt={product.image?.alt || product.title}
			loading="lazy"
		/>{/if}
	<div class="product-copy">
		<h2>{product.title}</h2>
		{#if product.price}<p>{money(product.price)}</p>{/if}
	</div>
	{#if href}<span class="external"><Icon name="external" size={20} /></span
		>{/if}
{/snippet}
{#if href}<a
		class="panel product-card"
		{href}
		target="_blank"
		rel="ugc nofollow noopener noreferrer"
		aria-label={'View product ' + product.title}
		onclick={(event) => {
			if (!confirm('Open this external website?')) event.preventDefault();
		}}>{@render content()}</a
	>{:else}<article class="panel product-card">{@render content()}</article>{/if}

<style>
	.product-card {
		margin: 0;
		padding: 0;
		display: flex;
		align-items: flex-start;
		overflow: hidden;
		color: var(--ink);
		text-decoration: none;
		min-height: 56px;
	}
	img {
		width: 112px;
		height: 112px;
		object-fit: cover;
		flex-shrink: 0;
	}
	.product-copy {
		padding: 12px;
		flex: 1;
		min-width: 0;
	}
	h2 {
		font-size: 16px;
		line-height: 1.4;
		margin: 0;
		overflow-wrap: anywhere;
	}
	p {
		margin: 4px 0 0;
		font-size: 14px;
	}
	.external {
		padding: 8px;
	}
</style>
