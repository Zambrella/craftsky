<script lang="ts">
	import { notify } from './toasts.svelte';
	import { confirmExternalLink } from './prompt';
	import { contextMenu } from './context-menu';
	import { onMount, onDestroy } from 'svelte';
	import { CraftskyAPI, type BusinessEvent } from '#lib/api/client.ts';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import { safeLink } from './model';
	import ReportForm from '../moderation/ReportForm.svelte';
	import Icon from '../shell/Icon.svelte';
	import { businessLabels, eventDisplay } from './business-model';
	let {
		api,
		did,
		rkey,
		viewerDid = ''
	}: {
		api: CraftskyAPI;
		did: string;
		rkey: string;
		viewerDid?: string;
	} = $props();
	const work = new Work('social.Event.action');
	let event = $state<BusinessEvent>(),
		reporting = $state(false);
	const display = $derived(event ? eventDisplay(event) : undefined);
	function load() {
		return work.run(
			() => api.business.event(did, rkey),
			(value) => {
				event = value;
			},
			undefined,
			'social.Event.load'
		);
	}
	onMount(() => {
		void load();
	});

	onDestroy(() => work.dispose());
</script>

<div class="workspace-header event-appbar">
	<a
		class="icon-button"
		href={'/profiles/' + encodeURIComponent(did) + '?tab=events'}
		aria-label="Back to organiser"><Icon name="back" /></a
	>
	<h1>Event</h1>
	{#if event}<details use:contextMenu>
			<summary class="icon-button" aria-label="Event actions"
				><Icon name="more" /></summary
			>
			<div class="menu-items">
				<button
					onclick={() =>
						void work.run(
							() =>
								navigator.clipboard.writeText(
									'https://app.craftsky.social/events/' +
										encodeURIComponent(did) +
										'/' +
										encodeURIComponent(rkey)
								),
							undefined,
							'Link copied.'
						)}><Icon name="link" />Copy event link</button
				>{#if viewerDid && viewerDid !== did}<button
						onclick={() => (reporting = !reporting)}
						><Icon name="shield" />Report event</button
					>{/if}
			</div>
		</details>{/if}
</div>
<Status {work} skeleton="posts" reload={() => void load()} />
{#if event}<article class="event-detail">
		<header
			class="event-hero"
			class:with-image={!!safeLink(event.image?.fullsize || '')}
		>
			{#if safeLink(event.image?.fullsize || '')}<img
					src={safeLink(event.image!.fullsize!)}
					alt={event.image?.alt || ''}
				/>{/if}
			<h1>{event.name}</h1>
		</header>
		<div class="event-content stack">
			{#if event.summary}<p class="prose event-summary">{event.summary}</p>{/if}
			<div class="event-badges">
				<span
					><Icon name="calendar" size={18} />{businessLabels(
						event.status.value
					)}</span
				><span
					><Icon name="clock" size={18} />{event.past
						? 'Past'
						: 'Upcoming'}</span
				>
			</div>
			<section class="panel event-information" aria-label="Event details">
				<div>
					<Icon name="calendar" />
					<p>{display?.date}</p>
				</div>
				<div>
					<Icon name="clock" />
					<p>
						{display?.time}{#if event.timeZone}<small>{event.timeZone}</small
							>{/if}
					</p>
				</div>
				{#if event.venueName}<div>
						<Icon name="location" />
						<p>{event.venueName}</p>
					</div>{/if}
				{#if event.mode}<div>
						<Icon name="profile" />
						<p>{businessLabels(event.mode.value)}</p>
					</div>{/if}
				{#if viewerDid === did}{#if event.roles.length}<div>
							<Icon name="business" />
							<p>
								{event.roles
									.map((role) => businessLabels(role.value))
									.join(', ')}<small>Your roles</small>
							</p>
						</div>{/if}
					<div>
						<Icon name="calendar" />
						<p>Published {new Date(event.createdAt).toLocaleDateString()}</p>
					</div>{/if}
			</section>
			<div class="event-destinations">
				{#if safeLink(event.eventUri || '')}<a
						class="secondary-button"
						href={safeLink(event.eventUri!)}
						target="_blank"
						rel="ugc nofollow noopener noreferrer"
						onclick={confirmExternalLink}
						><Icon name="external" size={20} />Event website</a
					>{/if}{#if safeLink(event.registrationUri || '')}<a
						class="button"
						href={safeLink(event.registrationUri!)}
						target="_blank"
						rel="ugc nofollow noopener noreferrer"
						onclick={confirmExternalLink}
						><Icon name="external" size={20} />Register</a
					>{/if}
			</div>
			<a class="organiser" href={'/profiles/' + encodeURIComponent(did)}
				><Icon name="business" size={18} />View organiser</a
			>
			{#if reporting}<ReportForm
					subject="event"
					busy={work.busy}
					onRetry={work.retry}
					onSubmit={(submission) =>
						void work.run(
							() => api.business.reportEvent(did, rkey, submission),
							() => {
								reporting = false;
								notify('Report submitted.');
							}
						)}
					onCancel={() => (reporting = false)}
				/>{/if}
		</div>
	</article>{/if}

<style>
	.event-appbar {
		justify-content: flex-start;
		gap: 8px;
	}
	.event-appbar h1 {
		flex: 1;
	}
	.event-appbar details {
		position: relative;
	}
	.event-appbar summary {
		list-style: none;
		cursor: pointer;
	}
	.event-appbar .menu-items button {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.event-hero {
		position: relative;
		padding: 24px 16px;
	}
	.event-hero h1 {
		font-size: 28px;
		line-height: 1.2;
		margin: 0;
	}
	.event-hero.with-image {
		min-height: 260px;
		display: flex;
		align-items: flex-end;
		background: #24221f;
	}
	.event-hero img {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.event-hero.with-image:after {
		content: '';
		position: absolute;
		inset: 0;
		background: linear-gradient(transparent 35%, rgba(0, 0, 0, 0.75));
	}
	.event-hero.with-image h1 {
		position: relative;
		color: white;
		z-index: 1;
	}
	.event-content {
		padding: 16px;
		gap: 16px;
	}
	.event-summary {
		font-size: 16px;
	}
	.event-badges {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.event-badges span {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		border: 1px solid var(--hairline);
		border-radius: 20px;
		background: var(--surface);
		padding: 6px 10px;
		font-size: 13px;
	}
	.event-information {
		margin: 0;
		padding: 16px;
	}
	.event-information > div {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		padding: 16px 0;
	}
	.event-information > div:first-child {
		padding-top: 0;
	}
	.event-information > div:last-child {
		padding-bottom: 0;
	}
	.event-information > div + div {
		border-top: 1px solid var(--hairline);
	}
	.event-information :global(svg) {
		color: var(--primary);
		flex-shrink: 0;
	}
	.event-information p {
		margin: 0;
		font-size: 15px;
	}
	.event-information small {
		display: block;
		color: var(--ink-2);
		font-size: 13px;
		margin-top: 4px;
	}
	.event-destinations {
		display: flex;
		flex-wrap: wrap;
		gap: 12px;
	}
	.event-destinations a,
	.organiser {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
	}
	.organiser {
		font-size: 14px;
		align-self: flex-start;
	}
</style>
