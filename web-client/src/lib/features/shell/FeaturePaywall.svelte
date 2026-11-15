<script lang="ts">
	import { onMount, onDestroy, getContext, untrack } from 'svelte';
	import type { CraftskyAPI } from '#lib/api/client.ts';
	import { APIError } from '#lib/api/client.ts';
	import { message } from '../settings/model';
	import { UserFacingError } from '#lib/user-facing-error.ts';
	import { isCanceled } from '#lib/api/error-message.ts';
	import { reportFailure } from '../diagnostics/client';
	import RevenueCatPaywall from '../settings/RevenueCatPaywall.svelte';
	import type { WebPaywallDetails } from '../settings/revenuecat-paywall';
	import { subscriptionNeedsRefresh } from '../settings/subscription-checkout';
	import {
		subscriptionAccessContext,
		type SubscriptionAccessState
	} from './subscription-access.svelte';
	let {
		api,
		account,
		did,
		tier,
		onClose
	}: {
		api: CraftskyAPI;
		account: string;
		did: string;
		tier: 'plus' | 'business';
		onClose: () => void;
	} = $props();
	const owner = untrack(() => account),
		ownerDid = untrack(() => did);
	const shared = getContext<SubscriptionAccessState>(subscriptionAccessContext);
	let active = true,
		dialog = $state<HTMLDialogElement>(null!);
	let details = $state<WebPaywallDetails | null>(null),
		error = $state('');
	function current() {
		if (!active || account !== owner || did !== ownerDid)
			throw new DOMException('Account changed', 'AbortError');
	}
	async function reconcile() {
		current();
		await api.billing.get().catch((error: unknown) => {
			if (error instanceof APIError && error.status === 404)
				return api.billing.ensure();
			throw error;
		});
		current();
		await api.billing.reconcile();
		current();
		let state = await api.billing.get();
		current();
		for (let i = 0; i < 15 && subscriptionNeedsRefresh(state); i++) {
			await new Promise((resolve) => setTimeout(resolve, 500));
			current();
			state = await api.billing.get();
			current();
		}
		const publish = shared.beginRead(owner);
		const access = await api.billing.access();
		current();
		if (access.did !== ownerDid)
			throw new UserFacingError(
				'Subscription access is unavailable. Please try again.'
			);
		publish(access);
		return access;
	}
	async function prepare() {
		try {
			const access = await reconcile();
			current();
			if (
				access.givesAccess &&
				(access.effectiveTier === 'business' ||
					(tier === 'plus' && access.effectiveTier === 'plus'))
			) {
				onClose();
				return;
			}
			const value = await api.billing.paywall(tier);
			current();
			dialog.close();
			details = value;
		} catch (cause) {
			if (active && !isCanceled(cause)) {
				void reportFailure(cause, 'settings.Billing.action');
				error = message(cause);
			}
		}
	}
	onMount(() => {
		dialog.showModal();
		void prepare();
	});
	onDestroy(() => {
		active = false;
		dialog?.close();
	});
	async function completed() {
		try {
			await reconcile();
		} catch (cause) {
			if (active && !isCanceled(cause))
				void reportFailure(cause, 'settings.Billing.refresh');
		} finally {
			if (active) onClose();
		}
	}
</script>

{#if details}
	<RevenueCatPaywall
		{details}
		beforePurchase={async () => {
			await reconcile();
			current();
			const next = await api.billing.paywall(tier);
			current();
			if (
				next.appUserId !== details?.appUserId ||
				next.apiKey !== details.apiKey ||
				next.offering !== details.offering
			)
				throw new DOMException('Account changed', 'AbortError');
		}}
		onComplete={() => {
			void completed();
		}}
		{onClose}
	/>
{:else}
	<dialog
		bind:this={dialog}
		aria-label="Subscription details"
		oncancel={(event) => {
			event.preventDefault();
			onClose();
		}}
	>
		{#if error}<p role="alert">{error}</p>{:else}<p role="status">
				Loading subscription details…
			</p>{/if}
		<button type="button" onclick={onClose}>Close</button>
	</dialog>
{/if}

<style>
	dialog {
		max-width: min(420px, calc(100vw - 32px));
		border: 1.5px solid var(--ink);
		border-radius: 16px;
		background: var(--surface);
		color: var(--ink);
		padding: 24px;
	}
	dialog::backdrop {
		background: #161210a6;
	}
</style>
