<script lang="ts">
	import { InputError, userErrorMessage } from '#lib/user-facing-error.ts';
	import { APIError, APITransport } from '#lib/api/transport.ts';
	import { reportFailure } from '../diagnostics/client';
	import { expectedFailure } from '../diagnostics/privacy';
	import { onMount, onDestroy } from 'svelte';
	import Snackbar from '../social/Snackbar.svelte';
	import { goto } from '$app/navigation';
	import type { CraftskyAPI } from '#lib/api/client.ts';
	import {
		bindings,
		accountForTarget,
		pushTarget,
		targetFromURL,
		pushURL,
		type PushTarget
	} from './model.ts';
	import { configuration, registerPush, listenPush } from './client.ts';
	let {
		api,
		session
	}: {
		api: CraftskyAPI;
		session: { id: string; did: string; handle: string };
	} = $props();
	let notification = $state<PushTarget | null>(null),
		error = $state('');
	let dead = false,
		stop: (() => void) | undefined,
		busy = false,
		refreshing = false;
	let resumeFailureReported = false;
	async function open(target: PushTarget) {
		if (busy) return;
		busy = true;
		try {
			const accountId = accountForTarget(bindings(localStorage), target);
			if (!accountId)
				throw new InputError(
					'Sign in to the account for this notification to open it.'
				);
			if (accountId !== session.id) {
				try {
					await new APITransport(fetch, '/api/auth').request(
						'POST',
						'/switch',
						{
							body: { accountId },
							response: 'empty'
						}
					);
				} catch (cause) {
					if (
						cause instanceof APIError &&
						cause.status >= 400 &&
						cause.status < 500
					)
						throw new InputError(
							'Sign in to the account for this notification to open it.',
							{ cause }
						);
					throw cause;
				}
				if (!dead) window.location.assign(pushURL(target));
				return;
			}
			if (target.reference) {
				const resolved = await api.chat.notificationOpen(target.reference);
				if (!dead)
					await goto(
						`/messages/conversations/${encodeURIComponent(resolved.conversationId)}`,
						{ replaceState: true }
					);
			} else if (!dead) await goto('/notifications', { replaceState: true });
			if (!dead) notification = null;
		} catch (e) {
			void reportFailure(e, 'push.open');
			if (!dead) error = userErrorMessage(e);
		} finally {
			busy = false;
		}
	}
	async function resume() {
		if (
			refreshing ||
			!configuration() ||
			!('Notification' in window) ||
			Notification.permission !== 'granted'
		)
			return;
		refreshing = true;
		try {
			const rows = bindings(localStorage);
			if (rows.some((row) => row.accountId === session.id && row.enabled))
				await registerPush(api, session.id);
			if (dead) return;
			stop?.();
			stop = undefined;
			if (rows.some((row) => row.enabled)) {
				const unsubscribe = await listenPush((data) => {
					const target = pushTarget(data);
					if (target && !dead) notification = target;
				});
				if (dead) unsubscribe();
				else stop = unsubscribe;
			}
			resumeFailureReported = false;
		} catch (error) {
			const unexpected = !expectedFailure(error);
			if (!unexpected || !resumeFailureReported)
				void reportFailure(error, 'push.resume');
			if (unexpected) resumeFailureReported = true;
			/* Optional push never prevents use of CraftSky. Settings provides an explicit retry. */
		} finally {
			refreshing = false;
		}
	}
	onMount(() => {
		const target = targetFromURL(new URL(location.href));
		if (target) void open(target);
		void resume();
		window.addEventListener('craftsky-push-change', resume);
		window.addEventListener('storage', resume);
	});
	onDestroy(() => {
		dead = true;
		stop?.();
		if (typeof window !== 'undefined') {
			window.removeEventListener('craftsky-push-change', resume);
			window.removeEventListener('storage', resume);
		}
	});
</script>

{#if notification}<aside class="push-banner" role="status">
		<span>You have a new CraftSky notification.</span><button
			onclick={() => notification && open(notification)}>Open</button
		><button
			aria-label="Dismiss notification"
			onclick={() => (notification = null)}>×</button
		>
	</aside>{/if}
{#if error}<Snackbar
		message={error}
		warning
		onDismiss={() => (error = '')}
	/>{/if}

<style>
	.push-banner {
		position: fixed;
		bottom: 90px;
		left: 50%;
		transform: translateX(-50%);
		z-index: 100;
		max-width: 90vw;
		background: var(--surface);
		color: var(--ink);
		border: 1.5px solid var(--ink);
		box-shadow: 3px 3px 0 var(--ink);
		padding: 1rem;
		border-radius: 12px;
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}
	.push-banner button {
		font: inherit;
		color: var(--primary);
		background: transparent;
		border: 0;
		cursor: pointer;
	}
</style>
