<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { CraftskyAPI, type BusinessEvent } from '#lib/api/client.ts';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import { safeLink } from './model';
	import ReportForm from '../moderation/ReportForm.svelte';
	let { api, did, rkey }: { api: CraftskyAPI; did: string; rkey: string } =
		$props();
	const work = new Work();
	let event = $state<BusinessEvent>(),
		reporting = $state(false),
		details = $state('');
	onMount(() => {
		void work.run(
			() => api.business.event(did, rkey),
			(value) => {
				event = value;
			}
		);
	});
	onDestroy(() => work.dispose());
</script>

<Status {work} />{#if event}<article class="panel stack">
		<p class="eyebrow">
			CraftSky event · {event.status.value.split('#').pop()}
		</p>
		<h1>{event.name}</h1>
		{#if safeLink(event.image?.fullsize || '')}<img
				src={safeLink(event.image!.fullsize!)}
				alt={event.image?.alt || ''}
			/>{/if}
		<p>
			{event.isAllDay
				? new Date(event.startsAt).toLocaleDateString()
				: new Date(event.startsAt).toLocaleString()} – {event.isAllDay
				? new Date(Date.parse(event.endsAt) - 86400000).toLocaleDateString()
				: new Date(event.endsAt).toLocaleString()}
			{event.isAllDay ? 'All day' : event.timeZone || ''}
		</p>
		<p class="prose">{event.summary || ''}</p>
		<p>{event.venueName || ''}</p>
		{#if event.mode}<p>{event.mode.value.split('#').pop()}</p>{/if}
		<p>{event.roles.map((role) => role.value.split('#').pop()).join(' · ')}</p>
		<a href={'/app/profile/' + encodeURIComponent(did)}>View organiser</a
		>{#if safeLink(event.eventUri || '')}<a
				href={safeLink(event.eventUri!)}
				target="_blank"
				rel="ugc nofollow noreferrer">Event website ↗</a
			>{/if}{#if safeLink(event.registrationUri || '')}<a
				class="button"
				href={safeLink(event.registrationUri!)}
				target="_blank"
				rel="ugc nofollow noreferrer">Register ↗</a
			>{/if}<button onclick={() => (reporting = !reporting)}
			>Report event</button
		>{#if reporting}<ReportForm
				subject="event"
				busy={work.busy}
				onSubmit={(submission) =>
					void work.run(
						() => api.business.reportEvent(did, rkey, submission),
						() => {
							reporting = false;
							work.error = 'Report submitted.';
						}
					)}
				onCancel={() => (reporting = false)}
			/>{/if}
	</article>{/if}
