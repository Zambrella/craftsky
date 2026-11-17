<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { BusinessEvent } from '#lib/api/types.ts';
	import { safeLink } from './model';
	import { businessLabels, eventDisplay } from './business-model';
	import Icon from '../shell/Icon.svelte';
	let { event, actions }: { event: BusinessEvent; actions?: Snippet } =
		$props();
	const display = $derived(eventDisplay(event));
</script>

<article class="panel event-card">
	{#if safeLink(event.image?.thumb || event.image?.fullsize || '')}<img
			src={safeLink(event.image?.thumb || event.image?.fullsize || '')}
			alt={event.image?.alt || ''}
			loading="lazy"
		/>{/if}
	<div class="event-copy">
		<div class="title-row">
			<a
				class="event-title"
				href={'/events/' +
					encodeURIComponent(event.did) +
					'/' +
					encodeURIComponent(event.rkey)}
				><h2>
					{event.name}{#if !actions}<span aria-hidden="true">›</span>{/if}
				</h2></a
			>{#if actions}<div class="event-actions">{@render actions()}</div>{/if}
		</div>
		<p>
			<Icon name="calendar" size={18} /><time datetime={event.startsAt}
				>{display.date}</time
			>
		</p>
		<p><Icon name="clock" size={18} />{display.time}</p>
		{#if event.venueName}<p>
				<Icon name="location" size={18} />{event.venueName}
			</p>{/if}{#if event.mode}<p>
				<Icon name="users" size={18} />{event.mode.value === 'in-person'
					? 'In person'
					: businessLabels(event.mode.value)}
			</p>{/if}{#if event.roles.length}<p>
				<Icon name="business" size={18} />{event.roles
					.map((role) => businessLabels(role.value))
					.join(', ')}
			</p>{/if}
	</div>
</article>

<style>
	:global(.settings-workspace) .event-card {
		padding: 0;
		margin: 0;
	}
	.title-row {
		display: flex;
		align-items: flex-start;
		gap: 8px;
	}
	.event-title {
		flex: 1;
		min-width: 0;
		color: inherit;
		text-decoration: none;
	}
	.event-title::after {
		content: '';
		position: absolute;
		inset: 0;
	}
	.event-title:focus-visible::after {
		outline: 3px solid var(--primary);
		outline-offset: -3px;
		border-radius: 14px;
	}
	.event-actions {
		position: relative;
		z-index: 1;
		flex-shrink: 0;
	}

	.event-card {
		position: relative;
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
		align-items: center;
		margin: 4px 0 0;
		font-size: 14px;
	}
	p :global(svg) {
		flex-shrink: 0;
		color: var(--primary);
	}
</style>
