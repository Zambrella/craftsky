<script lang="ts">
	import { reportConfirmation } from '../moderation/report-confirmation';
	import { notify } from './toasts.svelte';
	import { confirmExternalLink, usePrompt } from './prompt';
	import Avatar from '#lib/Avatar.svelte';
	import { goto } from '$app/navigation';
	import { contextMenu } from './context-menu';
	import { onMount, onDestroy } from 'svelte';
	import {
		CraftskyAPI,
		newOperationKey,
		type BusinessEvent,
		type Profile
	} from '#lib/api/client.ts';
	import { eventStatusBody } from '../settings/model';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import { safeLink } from './model';
	import ReportForm from '../moderation/ReportForm.svelte';
	import Icon from '../shell/Icon.svelte';
	import {
		businessLabels,
		eventDisplay,
		businessEventLink
	} from './business-model';
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
	const ask = usePrompt();
	const creatorWork = new Work('social.Event.load');
	let creator = $state<Profile>();
	const creatorName = $derived(
		creator?.displayName?.trim() ||
			(creator?.handle ? '@' + creator.handle : 'View creator profile')
	);
	function loadCreator() {
		return creatorWork.run(
			() => api.profiles.get(did),
			(value) => {
				creator = value;
			}
		);
	}
	const eventLink = $derived(businessEventLink(did, rkey));
	const creatorProfile = $derived('/profiles/' + encodeURIComponent(did));
	const creatorEvents = $derived(creatorProfile + '?tab=events');
	let event = $state<BusinessEvent>(),
		reporting = $state(false);
	const display = $derived(event ? eventDisplay(event) : undefined);
	const imageUrl = $derived(safeLink(event?.image?.fullsize || ''));
	let hero = $state<HTMLElement>();
	let collapsed = $state(false);
	function updateCollapsed() {
		collapsed = !!hero && hero.getBoundingClientRect().bottom <= 56;
	}
	$effect(() => {
		if (hero) updateCollapsed();
	});
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
	function changeStatus(status: 'cancelled' | 'postponed') {
		const original = event;
		if (!original || viewerDid !== original.did || work.busy || work.retry)
			return;
		const key = newOperationKey();
		void work.run(
			() =>
				api.business.updateEvent(
					original.did,
					original.rkey,
					eventStatusBody(original, status),
					original.cid,
					key
				),
			(result) => {
				event = {
					...original,
					cid: result.cid,
					status: { value: status, known: true }
				};
			},
			status === 'cancelled' ? 'Event cancelled.' : 'Event postponed.'
		);
	}
	async function deleteEvent() {
		const original = event;
		if (!original || viewerDid !== original.did || work.busy || work.retry)
			return;
		if (
			!(await ask({
				title: 'Delete this event?',
				detail: 'This permanently deletes the event record from your account.',
				confirmLabel: 'Delete',
				danger: true
			}))
		)
			return;
		const key = newOperationKey();
		void work.run(
			() =>
				api.business.deleteEvent(
					original.did,
					original.rkey,
					original.cid,
					key
				),
			() => {
				event = undefined;
				void goto(creatorEvents, { replaceState: true });
			},
			'Event deleted.'
		);
	}
	onMount(() => {
		void load();
		void loadCreator();
	});

	onDestroy(() => {
		work.dispose();
		creatorWork.dispose();
	});
</script>

<svelte:window onscroll={updateCollapsed} onresize={updateCollapsed} />

<div
	class="workspace-header event-appbar"
	class:with-image={!!imageUrl}
	class:expanded={!!imageUrl && !collapsed}
>
	<a
		class="icon-button"
		href={'/profiles/' + encodeURIComponent(did) + '?tab=events'}
		aria-label="Back to organiser"><Icon name="back" /></a
	>
	<h1>{event?.name || 'Event'}</h1>
	{#if event}<details use:contextMenu>
			<summary class="icon-button" aria-label="Event actions"
				><Icon name="more" /></summary
			>
			<div class="menu-items">
				<button
					onclick={() =>
						void work.run(
							() => navigator.clipboard.writeText(eventLink),
							undefined,
							'Link copied.'
						)}><Icon name="link" />Copy event link</button
				>
				{#if viewerDid}<a
						href={'/messages/new?link=' + encodeURIComponent(eventLink)}
						><Icon name="send" />Send via chat</a
					>{/if}
				{#if viewerDid === did}
					<div class="menu-divider"></div>
					<a
						href={'/compose/event/edit?event=' + encodeURIComponent(rkey)}
						inert={work.busy || !!work.retry}><Icon name="edit" />Edit event</a
					>
					{#if event.status.value !== 'postponed'}<button
							disabled={work.busy || !!work.retry}
							onclick={() => changeStatus('postponed')}
							><Icon name="clock" />Postpone event</button
						>{/if}
					{#if event.status.value !== 'cancelled'}<button
							disabled={work.busy || !!work.retry}
							onclick={() => changeStatus('cancelled')}
							><Icon name="close" />Cancel event</button
						>{/if}
					<button
						class="destructive"
						disabled={work.busy || !!work.retry}
						onclick={deleteEvent}><Icon name="delete" />Delete event</button
					>
				{/if}
				{#if viewerDid && viewerDid !== did}<button
						onclick={() => (reporting = !reporting)}
						><Icon name="flag" />Report event</button
					>{/if}
			</div>
		</details>{/if}
</div>
<Status {work} skeleton="posts" reload={() => void load()} />
<Status
	work={creatorWork}
	showLoading={false}
	reload={() => void loadCreator()}
/>
{#snippet creatorIdentity()}
	<a class="creator-identity" href={creatorProfile}>
		<span class="creator-avatar"
			><Avatar
				author={creator || { did, handle: '', displayName: creatorName }}
			/></span
		>
		<span>{creatorName}</span>
	</a>
{/snippet}
{#if event}<article class="event-detail">
		{#if imageUrl}<header class="event-hero" bind:this={hero}>
				<img src={imageUrl} alt={event.image?.alt || ''} />
				<h1>{event.name}</h1>
			</header>{/if}
		<div class="event-content stack">
			{#if event.summary}<p class="prose event-summary">{event.summary}</p>{/if}
			<div class="event-badges">
				<span
					><Icon name="eventAvailable" size={18} />{businessLabels(
						event.status.value
					)}</span
				><span
					><Icon
						name={event.past ? 'history' : 'upcoming'}
						size={18}
					/>{event.past ? 'Past' : 'Upcoming'}</span
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
						<Icon name="people" />
						<p>{businessLabels(event.mode.value)}</p>
					</div>{/if}
				{#if event.roles.length}<div>
						<Icon name="businessIdentity" />
						<p>
							{event.roles
								.map((role) => businessLabels(role.value))
								.join(', ')}{#if viewerDid === did}<small>Your role</small
								>{:else}{@render creatorIdentity()}{/if}
						</p>
					</div>{/if}
				{#if viewerDid === did}<div>
						<Icon name="publish" />
						<p>Published {new Date(event.createdAt).toLocaleDateString()}</p>
					</div>{/if}
			</section>
			{#if viewerDid === did}<a class="creator-profile" href={creatorProfile}
					><Icon name="profile" />View creator profile</a
				>{:else if !event.roles.length}{@render creatorIdentity()}{/if}
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
			{#if reporting}<ReportForm
					subject="event"
					busy={work.busy}
					onRetry={work.retry}
					onSubmit={(submission) =>
						void work.run(
							() => api.business.reportEvent(did, rkey, submission),
							() => {
								reporting = false;
								notify(reportConfirmation, false, true);
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
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.event-appbar.with-image {
		margin-bottom: -56px;
	}
	.event-appbar.expanded {
		background: transparent;
		border-bottom-color: transparent;
		color: white;
	}
	.event-appbar.expanded h1 {
		visibility: hidden;
	}
	.event-appbar.expanded :global(.icon-button) {
		color: white;
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
		aspect-ratio: 16 / 9;
		min-height: 180px;
		display: flex;
		align-items: flex-end;
		justify-content: center;
		padding: 56px 16px 20px;
		background: #24221f;
		border-bottom: 1.5px solid var(--ink);
	}
	.event-hero img {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.event-hero:after {
		content: '';
		position: absolute;
		inset: 0;
		background: linear-gradient(
			rgba(0, 0, 0, 0.6),
			transparent 55%,
			rgba(0, 0, 0, 0.6)
		);
	}
	.event-hero h1 {
		position: relative;
		color: white;
		z-index: 1;
		font-size: 30px;
		line-height: 1.2;
		font-weight: 600;
		text-align: center;
		margin: 0;
		overflow-wrap: anywhere;
	}
	.event-content {
		padding: 16px;
		gap: 16px;
		max-width: none;
		margin-inline: auto;
	}
	.event-summary {
		font-size: 16px;
		line-height: 1.5;
		margin: 0 0 -4px;
	}
	.event-badges {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.event-badges span {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		border: 1.5px solid var(--ink);
		border-radius: 999px;
		background: var(--surface);
		padding: 8px 12px;
		font-size: 13px;
		font-weight: 600;
		line-height: 1.3;
	}
	.event-information {
		margin: 0;
		padding: 16px;
	}
	.event-information > div {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		padding: 12px 0;
	}
	.event-information > div:first-child {
		padding-top: 0;
	}
	.event-information > div:last-child {
		padding-bottom: 0;
	}
	.event-information > div + div {
		border-top: 1px solid var(--ink-4);
	}
	.event-information :global(svg) {
		color: var(--primary);
		flex-shrink: 0;
	}
	.event-information p {
		margin: 0;
		font-size: 16px;
		font-weight: 600;
		line-height: 1.3;
		min-width: 0;
		overflow-wrap: anywhere;
	}
	.event-information small {
		display: block;
		color: var(--ink-2);
		font-size: 14px;
		font-weight: 400;
		line-height: 1.5;
		margin-top: 4px;
	}
	.creator-identity {
		display: flex;
		align-items: center;
		gap: 8px;
		min-height: 48px;
		font-size: 14px;
		font-weight: 400;
		text-decoration: none;
		width: fit-content;
		max-width: 100%;
	}
	.creator-avatar {
		display: flex;
		width: 24px;
		height: 24px;
		flex-shrink: 0;
	}
	.creator-avatar :global(.avatar) {
		width: 24px;
		height: 24px;
		min-width: 24px;
		font-size: 12px;
		border-width: 1.5px;
	}
	.creator-profile {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		min-height: 48px;
		align-self: flex-start;
		font-size: 15px;
		font-weight: 600;
		text-decoration: none;
	}
	.event-destinations {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.event-destinations:empty {
		display: none;
	}
	.event-destinations a {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		min-height: 48px;
		padding: 12px 20px;
		font-size: 16px;
		font-weight: 700;
		line-height: 1.2;
	}
</style>
