<script lang="ts">
	import type { CraftskyAPI } from '#lib/api/client.ts';
	import type { Session } from './model.ts';
	import Business from './Business.svelte';
	import './settings.css';
	let {
		api,
		session,
		eventRkey
	}: {
		api: CraftskyAPI;
		session: Session;
		eventRkey?: string;
	} = $props();
</script>

<section class="settings-workspace event-composer">
	<Business {api} {session} {eventRkey} section="event" />
</section>

<style>
	.event-composer {
		width: 100%;
		height: 100dvh;
		min-height: 420px;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		padding: 0;
		background: var(--paper);
	}
</style>
