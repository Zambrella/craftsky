<script lang="ts">
	import { onMount, onDestroy, untrack } from 'svelte';
	import {
		APIError,
		type CraftskyAPI,
		type SubscriptionAccess,
		type BillingState
	} from '#lib/api/client.ts';
	import SettingsIcon from './SettingsIcon.svelte';
	import { UserFacingError } from '#lib/user-facing-error.ts';
	import { dateTime, title, type Session } from './model.ts';
	import { Work } from './work.svelte.ts';
	import Feedback from './Feedback.svelte';
	import { storeManagementURL } from './navigation.ts';
	import { tierState, tierStateLabel } from './native-settings.ts';
	import {
		canStartWebCheckout,
		hasCurrentSubscription,
		isCurrentSubscription,
		SubscriptionCheckout,
		SubscriptionManagement,
		subscriptionNeedsRefresh,
		subscriptionProvider,
		subscriptionTier,
		type SubscriptionTier
	} from './subscription-checkout.ts';
	let { api, session }: { api: CraftskyAPI; session: Session } = $props();
	const work = new Work('settings.Billing.action');
	let active = true;
	let access = $state<SubscriptionAccess | null>(null),
		billing = $state<BillingState | null>(null),
		purchasingTier = $state<SubscriptionTier | null>(null),
		managingSubscription = $state(false);
	const checkout = new SubscriptionCheckout(
		untrack(() => `${session.id}:${session.did}`),
		() => `${session.id}:${session.did}`,
		(tier) => api.billing.checkout(tier),
		(url) => location.assign(url)
	);
	const management = new SubscriptionManagement(
		untrack(() => `${session.id}:${session.did}`),
		() => `${session.id}:${session.did}`,
		() => api.billing.manage(),
		(url) => location.assign(url)
	);
	const currentSubscriptions = $derived(
		billing?.subscriptions.filter((subscription) =>
			isCurrentSubscription(subscription)
		) || []
	);
	async function read() {
		const state = await api.billing.get().catch((error: unknown) => {
			if (error instanceof APIError && error.status === 404)
				return api.billing.ensure();
			throw error;
		});
		if (!active) throw new DOMException('View closed', 'AbortError');
		return { access: await api.billing.access(), billing: state };
	}
	function apply(value: { access: SubscriptionAccess; billing: BillingState }) {
		access = value.access;
		billing = value.billing;
	}
	async function reconciled() {
		await api.billing.reconcile();
		let value = await read();
		for (
			let attempt = 0;
			attempt < 15 && subscriptionNeedsRefresh(value.billing);
			attempt++
		) {
			await new Promise((resolve) => setTimeout(resolve, 500));
			if (!active) throw new DOMException('View closed', 'AbortError');
			value = await read();
		}
		return value;
	}
	function load() {
		return work.load(
			async () => {
				const value = await read();
				if (!active) throw new DOMException('View closed', 'AbortError');
				return value.billing.webCheckoutAvailable ? reconciled() : value;
			},
			apply,
			'settings.Billing.load'
		);
	}
	function refresh() {
		void work.run(
			reconciled,
			apply,
			'Subscription status refreshed. Payment updates can take a little time.',
			'settings.Billing.refresh'
		);
	}
	function purchase(tier: SubscriptionTier) {
		if (
			!billing ||
			!billing.webCheckoutAvailable ||
			hasCurrentSubscription(billing) ||
			work.busy
		)
			return;
		purchasingTier = tier;
		void work
			.run(
				async () => {
					const value = await reconciled();
					if (!active) return;
					apply(value);
					if (!canStartWebCheckout(value.billing))
						throw new UserFacingError(
							'Checkout cannot open yet. Review your subscription status and try again.'
						);
					await checkout.start(tier);
				},
				() => {},
				'',
				'settings.Billing.action'
			)
			.finally(() => {
				if (active) purchasingTier = null;
			});
	}
	function manage() {
		if (work.busy) return;
		managingSubscription = true;
		void work
			.run(
				() => management.start(),
				() => {},
				'',
				'settings.Billing.action'
			)
			.finally(() => {
				if (active) managingSubscription = false;
			});
	}
	onMount(() => {
		void load();
		const restored = (event: PageTransitionEvent) => {
			if (event.persisted) refresh();
		};
		window.addEventListener('pageshow', restored);
		return () => window.removeEventListener('pageshow', restored);
	});
	onDestroy(() => {
		active = false;
		checkout.dispose();
		management.dispose();
		work.dispose();
	});
</script>

<h1 class="workspace-heading">Subscriptions</h1>
<Feedback {work} reload={load} />
{#if !work.loading && access && billing}
	<section class="panel">
		<h2 class="section-heading access-title">
			<SettingsIcon name="plus" />Current access: {title(access.effectiveTier)}
		</h2>
		<p>
			Your subscription belongs to <strong
				>@{session.handle || session.did}</strong
			> and follows this account across devices.
		</p>
		{#if access.accessEndsAt}<p class="muted">
				Access until {dateTime(access.accessEndsAt)}
			</p>{/if}
	</section>
	<div class="billing-refresh">
		<button disabled={work.busy} onclick={refresh}>
			{work.busy && !purchasingTier && !managingSubscription
				? 'Refreshing…'
				: 'Refresh status'}
		</button>
	</div>
	{#if subscriptionNeedsRefresh(billing)}
		<p class="notice">
			Your subscription status is still being checked. Refresh status before
			making a purchase.
		</p>
	{/if}
	{#each currentSubscriptions as subscription}
		{@const tier = subscriptionTier(billing, subscription)}
		{@const managementURL = storeManagementURL(subscription.store)}
		<section class="panel" aria-label="Current subscription">
			<h2>{tier ? `CraftSky ${title(tier)}` : 'Subscription'}</h2>
			<p>Billed through {subscriptionProvider(subscription.store)}.</p>
			{#if subscription.pendingTier && subscription.autoRenewalStatus === 'will_change_product'}
				<p class="notice">
					Changes to CraftSky {title(subscription.pendingTier)} at the next renewal.
					Your current benefits remain available until then.
				</p>
			{/if}
			{#if subscription.pendingPayment}<p class="notice">
					Payment is pending. Resolve it with your billing provider before
					making another purchase.
				</p>{/if}
			{#if subscription.endsAt || subscription.currentPeriodEndsAt}
				<p class="muted">
					{subscription.autoRenewalStatus === 'will_renew'
						? 'Renews on'
						: 'Current period ends'}
					{dateTime(subscription.endsAt || subscription.currentPeriodEndsAt)}
				</p>
			{/if}
			{#if managementURL}<a
					class="action"
					href={managementURL}
					target="_blank"
					rel="noopener noreferrer"
					>Manage in {subscriptionProvider(subscription.store)}</a
				>
			{:else if subscription.store === 'stripe'}
				<button disabled={work.busy} onclick={manage}
					>{managingSubscription
						? 'Opening Stripe…'
						: 'Manage in Stripe'}</button
				>
			{/if}
		</section>
	{/each}
	{#if hasCurrentSubscription(billing)}
		<p class="muted">
			This account already has a subscription or a payment awaiting resolution.
			Web plan changes are not available yet. To choose another plan here,
			cancel your current subscription and wait for it to expire.
		</p>
	{:else if !billing.webCheckoutAvailable}
		<p class="muted">Web subscriptions are not available yet.</p>
	{:else}
		<p class="muted">
			Choose Plus or Business for this account. You can review the price and
			billing details before paying securely with Stripe.
		</p>
	{/if}
	{#each ['plus', 'business'] as tierValue}
		{@const tier = tierValue as SubscriptionTier}
		{@const status = tierState(billing, tier)}
		<section
			class="panel tier-card"
			class:business-tier={tier === 'business'}
			aria-label={`CraftSky ${title(tier)}`}
		>
			<div class="tier-heading">
				<span class="tier-icon"
					><SettingsIcon
						name={tier === 'business' ? 'business' : 'subscription'}
					/></span
				>
				<div>
					<h2>{title(tier)}</h2>
					{#if status !== 'empty'}<p class="tier-status">
							{tierStateLabel[status]}
						</p>{/if}
				</div>
			</div>
			<p class="muted">
				{tier === 'business'
					? 'Everything in Plus, Featured products, Profile action button, Business details and events.'
					: 'Schedule posts, Saved folders, Profile customisation, Pinned posts, Follower growth.'}
			</p>
			{#if canStartWebCheckout(billing)}
				<button
					class="primary"
					disabled={work.busy}
					onclick={() => purchase(tier)}
					>{purchasingTier === tier
						? 'Opening checkout…'
						: `Choose ${title(tier)}`}</button
				>
			{/if}
		</section>
	{/each}
	{#if billing.reconciledAt}<p class="muted">
			Last checked {dateTime(billing.reconciledAt)}
		</p>{/if}
{/if}

<style>
	.billing-refresh {
		display: flex;
		justify-content: flex-end;
		margin-bottom: 16px;
	}
	.tier-card.panel {
		background: color-mix(in srgb, var(--primary) 8%, var(--surface));
	}
	.business-tier.panel {
		background: color-mix(in srgb, var(--moss) 8%, var(--surface));
	}
	.tier-heading {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		margin-bottom: 10px;
	}
	.tier-heading h2 {
		margin: 0 0 2px;
	}
	.tier-icon {
		width: 44px;
		height: 44px;
		border: 1.5px solid var(--primary);
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		background: color-mix(in srgb, var(--primary) 16%, var(--surface));
		color: var(--primary);
		flex-shrink: 0;
	}
	.business-tier .tier-icon {
		border-color: var(--moss);
		background: color-mix(in srgb, var(--moss) 16%, var(--surface));
		color: var(--ink);
	}
	.tier-status {
		color: var(--primary);
		font-size: 14px;
	}
	.access-title {
		font-size: 16px;
	}
</style>
