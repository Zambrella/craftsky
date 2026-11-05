<script lang="ts">
	import Snackbar from '../social/Snackbar.svelte';
	import { onMount, onDestroy } from 'svelte';
	import type { CraftskyAPI } from '#lib/api/client.ts';
	import { bindings } from './model.ts';
	import { configuration, registerPush, disablePush } from './client.ts';
	let {
		api,
		session
	}: {
		api: CraftskyAPI;
		session: { id?: string; did: string; handle: string };
	} = $props();
	let configured = $state(false),
		enabled = $state(false),
		permission = $state<NotificationPermission>('default'),
		busy = $state(false),
		error = $state(''),
		notice = $state('');
	let dead = false;
	function sync() {
		configured = !!configuration();
		try {
			enabled = bindings(localStorage).some(
				(row) => row.accountId === session.id && row.enabled
			);
			permission =
				'Notification' in window ? Notification.permission : 'denied';
		} catch {}
	}
	async function toggle() {
		if (!session.id) return;
		const id = session.id;
		busy = true;
		error = '';
		notice = '';
		try {
			if (enabled) await disablePush(api, id);
			else await registerPush(api, id, true);
			if (!dead) {
				sync();
				notice = enabled
					? 'Browser notifications enabled for this account.'
					: 'Browser notifications disabled for this account.';
			}
		} catch {
			if (!dead)
				error =
					'Could not update browser notifications. Check your browser notification permission and try again.';
		} finally {
			if (!dead) busy = false;
		}
	}
	onMount(() => {
		sync();
		window.addEventListener('craftsky-push-change', sync);
		window.addEventListener('storage', sync);
	});
	onDestroy(() => {
		dead = true;
		if (typeof window !== 'undefined') {
			window.removeEventListener('craftsky-push-change', sync);
			window.removeEventListener('storage', sync);
		}
	});
</script>

{#if configured}<section class="panel">
		<h2>This browser</h2>
		<p>
			Receive CraftSky notifications when you are away. This setting applies to
			@{session.handle} on this browser.
		</p>
		{#if permission === 'denied'}<p class="notice">
				Notifications are blocked by this browser. Allow notifications in your
				browser settings. On iPhone or iPad, open CraftSky from your Home
				Screen.
			</p>{/if}{#if error}<Snackbar
				message={error}
				warning
				onDismiss={() => (error = '')}
			/>{/if}{#if notice}<Snackbar
				message={notice}
				onDismiss={() => (notice = '')}
			/>{/if}<button
			class="primary"
			disabled={busy || !session.id}
			onclick={toggle}
			>{busy
				? 'Updating…'
				: enabled
					? 'Disable browser notifications'
					: 'Enable browser notifications'}</button
		>
	</section>{/if}
