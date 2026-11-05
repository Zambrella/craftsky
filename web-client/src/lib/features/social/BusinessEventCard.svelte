<script lang="ts">
	import type { BusinessEvent } from '#lib/api/types.ts';
	import { safeLink } from './model';
	import { businessLabels, eventDisplay } from './business-model';
	import Icon from '../shell/Icon.svelte';
	let { event }: { event: BusinessEvent } = $props();
	const display = $derived(eventDisplay(event));
</script>

<a
	class="panel event-card"
	href={'/events/' +
		encodeURIComponent(event.did) +
		'/' +
		encodeURIComponent(event.rkey)}
>
	{#if safeLink(event.image?.thumb || event.image?.fullsize || '')}<img
			src={safeLink(event.image?.thumb || event.image?.fullsize || '')}
			alt={event.image?.alt || ''}
			loading="lazy"
		/>{/if}
	<div class="event-copy">
		<h2>{event.name}<span aria-hidden="true">›</span></h2>
		<p>
			<Icon name="calendar" size={18} /><time datetime={event.startsAt}
				>{display.date}</time
			>
		</p>
		<p><Icon name="clock" size={18} />{display.time}</p>
		{#if event.venueName}<p>
				<Icon name="location" size={18} />{event.venueName}
			</p>{/if}{#if event.mode}<p>
				<Icon name="profile" size={18} />{businessLabels(event.mode.value)}
			</p>{/if}{#if event.roles.length}<p>
				<Icon name="business" size={18} />{event.roles
					.map((role) => businessLabels(role.value))
					.join(', ')}
			</p>{/if}
	</div>
</a>

<style>
	.event-card {
		margin: 0;
		padding: 0;
		display: block;
		overflow: hidden;
		text-decoration: none;
		color: var(--ink);
	}
	img {
		width: 100%;
		aspect-ratio: 16/9;
		object-fit: cover;
		display: block;
	}
	.event-copy {
		padding: 12px;
	}
	h2 {
		font-size: 22px;
		line-height: 1.3;
		margin: 0 0 8px;
		display: flex;
		gap: 8px;
		justify-content: space-between;
	}
	p {
		display: flex;
		gap: 8px;
		align-items: flex-start;
		margin: 4px 0 0;
		font-size: 14px;
	}
	p :global(svg) {
		flex-shrink: 0;
	}
</style>
