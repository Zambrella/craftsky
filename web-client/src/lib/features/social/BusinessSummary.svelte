<script lang="ts">
	import { confirmExternalLink } from './prompt';
	import type { BusinessProfile } from '#lib/api/types.ts';
	import Icon from '../shell/Icon.svelte';
	import {
		businessAction,
		businessLabels,
		countryName
	} from './business-model';
	let {
		business,
		full = false
	}: { business: BusinessProfile; full?: boolean } = $props();
	const action = $derived(businessAction(business.primaryAction));
	const location = $derived(
		[business.location?.locality, countryName(business.location?.country)]
			.filter((value) => typeof value === 'string' && value)
			.join(', ') || business.serviceArea
	);
</script>

<section
	class="business-summary stack"
	class:full
	aria-label="Business details"
>
	{#if !full}{#if business.tagline}<p>
				<strong>{business.tagline}</strong>
			</p>{/if}{#if location}<p class="business-location">
				<Icon name="location" size={18} />{location}
			</p>{/if}{/if}
	{#if full}{#if business.businessTypes?.length}<div
				class="business-detail-section"
			>
				<h3>Business types</h3>
				<div class="business-chips">
					{#each business.businessTypes as item}<span
							>{businessLabels(item.value)}</span
						>{/each}
				</div>
			</div>{/if}
		{#if business.offerings?.length}<div class="business-detail-section">
				<h3>Offerings</h3>
				<div class="business-chips">
					{#each business.offerings as item}<span
							>{businessLabels(item.value)}</span
						>{/each}
				</div>
			</div>{/if}
		{#if business.serviceArea && business.location}<div>
				<h3>Service area</h3>
				<p>{business.serviceArea}</p>
			</div>{/if}
		{#if business.hoursNote}<div>
				<h3>Hours</h3>
				<p class="prose">{business.hoursNote}</p>
			</div>{/if}
	{/if}{#if action && !full}<a
			class="secondary-button"
			href={action.href}
			target={action.href.startsWith('https:') ? '_blank' : undefined}
			rel="ugc nofollow noopener noreferrer"
			onclick={confirmExternalLink}
			><Icon
				name={action.href.startsWith('mailto:') ? 'mail' : 'external'}
				size={18}
			/>{action.label}</a
		>{/if}
</section>

<style>
	.business-summary {
		align-items: center;
		text-align: center;
		gap: 8px;
	}
	.business-summary p {
		font-size: 14px;
	}
	.business-summary .secondary-button {
		max-width: 100%;
		display: inline-flex;
		align-items: center;
		gap: 8px;
	}
	.business-location {
		display: flex;
		align-items: center;
		gap: 4px;
	}
	.business-location :global(svg) {
		color: var(--ink-2);
		flex-shrink: 0;
	}
	.business-summary.full {
		align-items: stretch;
		text-align: left;
		gap: 20px;
	}
	.full h3 {
		font-size: 12px;
		line-height: 1.2;
		letter-spacing: 1.68px;
		color: var(--ink-2);
		font-weight: 600;
		margin: 0 0 4px;
	}
	.business-detail-section h3 {
		margin-bottom: 8px;
	}
	.full p {
		margin: 0;
		font-size: 14px;
		line-height: 1.5;
		white-space: pre-wrap;
	}
	.business-chips {
		display: flex;
		gap: 4px 8px;
		flex-wrap: wrap;
	}
	.business-chips span {
		display: inline-flex;
		align-items: center;
		background: var(--surface);
		border: 1.5px solid var(--ink);
		padding: 8px 12px;
		border-radius: 999px;
		font-size: 13px;
		font-weight: 600;
		line-height: 1.3;
	}
</style>
