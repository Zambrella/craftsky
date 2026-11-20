<script lang="ts">
	import { InputError, userErrorMessage } from '#lib/user-facing-error.ts';
	import { APIError, APITransport } from '#lib/api/transport.ts';
	import { reportFailure } from '../diagnostics/client';
	import { expectedFailure } from '../diagnostics/privacy';
	import { onMount, onDestroy } from 'svelte';
	import Snackbar from '../social/Snackbar.svelte';
	import { afterNavigate, goto } from '$app/navigation';
	import type { CraftskyAPI } from '#lib/api/client.ts';
	import {
		bindings,
		accountForTarget,
		pushTarget,
		pushBindingsKey,
		isPushURL,
		targetFromURL,
		pushURL,
		type PushTarget
	} from './model.ts';
	import { configuration, registerPush, listenPush } from './client.ts';
	let {
		api,
		session,
		refreshCounts
	}: {
		api: CraftskyAPI;
		session: { id: string; did: string; handle: string };
		refreshCounts: () => Promise<void>;
	} = $props();
	let notification = $state<PushTarget | null>(null),
		error = $state('');
	let dead = false,
		stop: (() => void) | undefined,
		starting = false;
	type OpenAttempt = {
		target: PushTarget | null;
		fromURL: boolean;
		read: AbortController;
	};
	let latestOpen: OpenAttempt | undefined;
	let opening = false;
	let resumeFailureReported = false;
	const accountFor = (target: PushTarget) =>
		accountForTarget(bindings(localStorage), target);
	const refreshActivity = () =>
		window.dispatchEvent(new Event('craftsky-activity-refresh'));

	function cancelOpen() {
		latestOpen?.read.abort();
		latestOpen = undefined;
	}

	function open(target: PushTarget | null, fromURL = false) {
		cancelOpen();
		latestOpen = { target, fromURL, read: new AbortController() };
		error = '';
		if (!opening) void drainOpens();
	}

	async function drainOpens() {
		opening = true;
		try {
			while (!dead && latestOpen) {
				const attempt = latestOpen;
				await openCurrent(attempt);
				if (latestOpen === attempt) latestOpen = undefined;
			}
		} finally {
			opening = false;
		}
	}

	async function selectRecipient(accountId: string) {
		try {
			await new APITransport(fetch, '/api/auth').request('POST', '/switch', {
				body: { accountId },
				response: 'empty'
			});
		} catch (cause) {
			if (
				cause instanceof APIError &&
				cause.status >= 400 &&
				cause.status < 500
			)
				throw new InputError(
					'Sign in to the account for this notification to open it.',
					{
						cause
					}
				);
			throw cause;
		}
	}

	async function openCurrent(attempt: OpenAttempt) {
		const { target, fromURL, read } = attempt;
		const current = () => !dead && latestOpen === attempt;
		const unavailable = () =>
			new InputError(
				'Sign in to the account for this notification to open it.'
			);
		try {
			if (!target) throw new InputError('This notification link is invalid.');
			const accountId = accountFor(target);
			if (!accountId) throw unavailable();
			let destination = '/notifications';
			if (accountId === session.id && target.reference) {
				const resolved = await api.chat.notificationOpen(
					target.reference,
					read.signal
				);
				destination = `/messages/conversations/${encodeURIComponent(resolved.conversationId)}`;
			}
			if (!current()) return;
			if (accountFor(target) !== accountId) throw unavailable();
			if (
				accountId === session.id &&
				!target.reference &&
				location.pathname === '/notifications' &&
				!location.search
			) {
				if (notification === target) notification = null;
				refreshActivity();
				return;
			}

			// Account selection writes a shared cookie. Serialize these requests
			// and reaffirm the recipient even if another tab changed the selection.
			await selectRecipient(accountId);
			if (!current()) return;
			if (accountFor(target) !== accountId) throw unavailable();
			if (notification === target) notification = null;
			if (accountId !== session.id) {
				window.location.assign(pushURL(target));
				return;
			}
			await goto(destination, { replaceState: true });
		} catch (e) {
			if (!current()) return;
			void reportFailure(e, 'push.open');
			error = userErrorMessage(e);
			if (fromURL) await goto('/feed', { replaceState: true }).catch(() => {});
		}
	}
	function canListen() {
		return (
			!!configuration() &&
			'Notification' in window &&
			Notification.permission === 'granted' &&
			bindings(localStorage).some((row) => row.enabled)
		);
	}
	function reportResumeFailure(error: unknown) {
		const unexpected = !expectedFailure(error);
		if (!unexpected || !resumeFailureReported)
			void reportFailure(error, 'push.resume');
		if (unexpected) resumeFailureReported = true;
	}
	async function syncListener() {
		try {
			if (notification && !accountFor(notification)) notification = null;
			if (!canListen()) {
				stop?.();
				stop = undefined;
				return;
			}
			if (stop || starting) return;
			starting = true;
			try {
				const unsubscribe = await listenPush((data) => {
					if (dead) return;
					const target = pushTarget(data);
					if (!target) return;
					const accountId = accountFor(target);
					if (!accountId) return;
					notification = target;
					if (accountId === session.id) {
						if (!target.reference) refreshActivity();
						void refreshCounts();
					}
				});
				if (dead || !canListen()) unsubscribe();
				else stop = unsubscribe;
				resumeFailureReported = false;
			} finally {
				starting = false;
			}
		} catch (error) {
			reportResumeFailure(error);
		}
	}
	afterNavigate(({ to }) => {
		if (to && isPushURL(to.url)) open(targetFromURL(to.url), true);
		else cancelOpen();
	});
	onMount(() => {
		if (configuration())
			void registerPush(api, session.id).catch(reportResumeFailure);
		void syncListener();
		const syncStorage = (event: StorageEvent) => {
			if (event.key === null || event.key === pushBindingsKey)
				void syncListener();
		};
		window.addEventListener('craftsky-push-change', syncListener);
		window.addEventListener('storage', syncStorage);
		window.addEventListener('focus', syncListener);
		return () => {
			window.removeEventListener('craftsky-push-change', syncListener);
			window.removeEventListener('storage', syncStorage);
			window.removeEventListener('focus', syncListener);
		};
	});
	onDestroy(() => {
		dead = true;
		cancelOpen();
		stop?.();
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
