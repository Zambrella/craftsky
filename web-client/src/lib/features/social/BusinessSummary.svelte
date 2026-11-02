<script lang="ts">
	import type { BusinessProfile } from '#lib/api/types.ts';
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

<section class="business-summary stack" aria-label="Business details">
	{#if business.tagline}<p>
			<strong>{business.tagline}</strong>
		</p>{/if}{#if location}<p>
			{location}
		</p>{/if}{#if full}{#if business.hoursNote}<p class="prose">
				{business.hoursNote}
			</p>{/if}{#if business.serviceArea && business.location}<p>
				Serving {business.serviceArea}
			</p>{/if}{#if business.businessTypes?.length}<p>
				{business.businessTypes
					.map((item) => businessLabels(item.value))
					.join(' · ')}
			</p>{/if}{#if business.offerings?.length}<p>
				{business.offerings
					.map((item) => businessLabels(item.value))
					.join(' · ')}
			</p>{/if}{/if}{#if action}<a
			class="secondary-button"
			href={action.href}
			target={action.href.startsWith('https:') ? '_blank' : undefined}
			rel="ugc nofollow noopener noreferrer"
			onclick={(event) => {
				if (
					!confirm(
						action.href.startsWith('mailto:')
							? 'Open your email app?'
							: 'Open this external website?'
					)
				)
					event.preventDefault();
			}}>{action.label} ↗</a
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
	}
</style>
