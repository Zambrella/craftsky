<script lang="ts">
	import { getContext, onMount, onDestroy, untrack, tick } from 'svelte';
	import {
		APIError,
		type CraftskyAPI,
		type SubscriptionAccess,
		type WebSubscriptionPrice,
		type BillingState
	} from '#lib/api/client.ts';
	import SettingsIcon from './SettingsIcon.svelte';
	import PlusTierIcon from '../shell/PlusTierIcon.svelte';
	import RevenueCatPaywall from './RevenueCatPaywall.svelte';
	import type { WebPaywallDetails } from './revenuecat-paywall';
	import { monthlyPrice } from './subscription-prices';
	import { UserFacingError } from '#lib/user-facing-error.ts';
	import { dateTime, title, type Session } from './model.ts';
	import { Work } from './work.svelte.ts';
	import Feedback from './Feedback.svelte';
	import { storeManagementURL } from './navigation.ts';
	import { tierState, tierStateLabel } from './native-settings.ts';
	import {
		subscriptionAccessContext,
		type SubscriptionAccessState
	} from '../shell/subscription-access.svelte';
	import {
		activePromotionalGrants,
		canStartWebCheckout,
		hasCurrentSubscription,
		isCurrentSubscription,
		SubscriptionManagement,
		subscriptionNeedsRefresh,
		subscriptionProvider,
		subscriptionTier,
		type SubscriptionTier
	} from './subscription-checkout.ts';
	let { api, session }: { api: CraftskyAPI; session: Session } = $props();
	const work = new Work('settings.Billing.action');
	const account = untrack(() => `${session.id}:${session.did}`);
	const ownerDid = untrack(() => session.did);
	const sharedAccess = getContext<SubscriptionAccessState | undefined>(
		subscriptionAccessContext
	);
	let active = true;
	let prices = $state<WebSubscriptionPrice[]>([]);
	let pricesLoading = $state(true);
	let priceRead = 0;
	async function loadPrices() {
		const read = ++priceRead;
		pricesLoading = true;
		try {
			const result = await api.billing.prices();
			if (active && read === priceRead) prices = result.items;
		} catch {
			if (active && read === priceRead) prices = [];
		} finally {
			if (active && read === priceRead) pricesLoading = false;
		}
	}
	let access = $state<SubscriptionAccess | null>(null),
		billing = $state<BillingState | null>(null),
		purchasingTier = $state<SubscriptionTier | null>(null),
		managingSubscription = $state(false);
	let paywallDetails = $state<WebPaywallDetails | null>(null);
	let checkoutTier = $state<SubscriptionTier>('plus');
	let cancelingCheckout = $state(false);
	async function restoreCheckoutFocus() {
		await tick();
		if (active && `${session.id}:${session.did}` === account)
			document
				.getElementById(`subscription-${checkoutTier}-details`)
				?.focus({ preventScroll: true });
	}
	async function cancelCheckout() {
		if (!active || `${session.id}:${session.did}` !== account) return false;
		cancelingCheckout = true;
		try {
			await api.billing.cancelCheckout();
			requireCurrent();
			paywallDetails = null;
			await load();
			await restoreCheckoutFocus();
			return true;
		} catch {
			if (active) {
				try {
					const value = await read();
					apply(value);
					if (hasCurrentSubscription(value.billing)) {
						paywallDetails = null;
						await restoreCheckoutFocus();
						return true;
					}
				} catch {
					/* Keep the unresolved checkout available for a safe retry. */
				}
			}
			return false;
		} finally {
			if (active) cancelingCheckout = false;
		}
	}
	const management = new SubscriptionManagement(
		account,
		() => `${session.id}:${session.did}`,
		() => api.billing.manage(),
		(url) => location.assign(url)
	);
	const currentSubscriptions = $derived(
		billing?.subscriptions.filter((subscription) =>
			isCurrentSubscription(subscription)
		) || []
	);
	const promotionalGrants = $derived(
		billing ? activePromotionalGrants(billing) : []
	);
	function requireCurrent() {
		if (!active || `${session.id}:${session.did}` !== account)
			throw new DOMException('View closed', 'AbortError');
	}
	async function read() {
		requireCurrent();
		const publishAccess = sharedAccess?.beginRead(account);
		const state = await api.billing.get().catch((error: unknown) => {
			requireCurrent();
			if (error instanceof APIError && error.status === 404)
				return api.billing.ensure();
			throw error;
		});
		requireCurrent();
		const currentAccess = await api.billing.access();
		requireCurrent();
		if (currentAccess.did !== ownerDid)
			throw new UserFacingError(
				'Subscription status is unavailable. Please try again.'
			);
		return { access: currentAccess, billing: state, publishAccess };
	}
	function apply(value: Awaited<ReturnType<typeof read>>) {
		requireCurrent();
		if (value.publishAccess && !value.publishAccess(value.access)) return;
		access = value.access;
		billing = value.billing;
	}
	async function reconciled() {
		requireCurrent();
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
		void loadPrices();
		return work.run(
			reconciled,
			apply,
			'Subscription status refreshed. Payment updates can take a little time.',
			'settings.Billing.refresh'
		);
	}
	async function closePaywall() {
		paywallDetails = null;
		await refresh();
		await restoreCheckoutFocus();
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
					const details = await api.billing.paywall(tier);
					requireCurrent();
					checkoutTier = tier;
					paywallDetails = details;
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
		void loadPrices();
		const restored = (event: PageTransitionEvent) => {
			if (event.persisted) refresh();
		};
		window.addEventListener('pageshow', restored);
		return () => window.removeEventListener('pageshow', restored);
	});
	onDestroy(() => {
		active = false;
		paywallDetails = null;
		management.dispose();
		work.dispose();
	});
</script>

{#if paywallDetails}<RevenueCatPaywall
		details={paywallDetails}
		beforePurchase={async () => {
			requireCurrent();
			const fresh = await reconciled();
			apply(fresh);
			if (
				!canStartWebCheckout(fresh.billing) ||
				fresh.billing.webCheckoutPending
			)
				throw new UserFacingError(
					'Refresh subscription status before purchasing.'
				);
			const identity = await api.billing.paywall(checkoutTier);
			requireCurrent();
			if (
				identity.appUserId !== paywallDetails?.appUserId ||
				identity.apiKey !== paywallDetails.apiKey ||
				identity.offering !== paywallDetails.offering
			)
				throw new DOMException('Account changed', 'AbortError');
		}}
		onComplete={() => {
			void closePaywall();
		}}
		onClose={() => {
			void closePaywall();
		}}
	/>{/if}

<h1 class="workspace-heading">Subscriptions</h1>
<Feedback {work} reload={load} />
{#if !work.loading && access && billing}
	<section class="panel">
		<h2 class="access-title">
			Current access: {title(access.effectiveTier)}
		</h2>
		{#if access.accessEndsAt}<p class="muted">
				Access until {dateTime(access.accessEndsAt)}
			</p>{/if}
	</section>
	<p class="account-plan">
		Your subscription belongs to this CraftSky account. Business includes
		everything in Plus.
	</p>
	{#each promotionalGrants as grant (grant.id)}
		<section
			class="panel"
			aria-label={`Complimentary CraftSky ${title(grant.tier)}`}
		>
			<h2>Complimentary access</h2>
			<p>This complimentary access has no recurring charge.</p>
			{#if grant.expiresAt !== undefined && grant.expiresAt !== null}<p
					class="muted"
				>
					{title(grant.tier)} until {dateTime(grant.expiresAt)}
				</p>{:else}<p class="muted">
					{title(grant.tier)} with no expiry date
				</p>{/if}
		</section>
	{/each}
	{#if billing.webCheckoutPending && !paywallDetails}
		<div class="notice">
			<p>
				A purchase is still pending for this account. Finish or cancel it before
				choosing a plan.
			</p>
			<button disabled={work.busy || cancelingCheckout} onclick={cancelCheckout}
				>{cancelingCheckout ? 'Canceling…' : 'Cancel pending purchase'}</button
			>
		</div>
	{/if}
	{#if subscriptionNeedsRefresh(billing)}
		<p class="notice">
			Your subscription status is still being checked. Please wait a moment.
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
		{#if promotionalGrants.length}
			<p class="notice">
				Subscribing starts paid billing immediately. Remaining complimentary
				time is not added to your paid subscription.
			</p>
		{/if}
	{/if}
	{#each ['plus', 'business'] as tierValue}
		{@const tier = tierValue as SubscriptionTier}
		{@const status = tierState(billing, tier)}
		{@const price = prices.find((price) => price.tier === tier)}
		<section
			class="panel tier-card"
			class:business-tier={tier === 'business'}
			aria-label={`CraftSky ${title(tier)}`}
		>
			<div class="tier-heading">
				<span class="tier-icon"
					>{#if tier === 'plus'}<PlusTierIcon />{:else}<SettingsIcon
							name="business"
						/>{/if}</span
				>
				<div>
					<h2>{title(tier)}</h2>
					<p class="tier-price">
						{pricesLoading
							? 'Loading price…'
							: monthlyPrice(price) || 'Price unavailable'}
					</p>
					{#if status !== 'empty'}<p class="tier-status">
							{tierStateLabel[status]}
						</p>{/if}
				</div>
			</div>
			<p class="tier-benefits">
				{tier === 'business'
					? 'Everything in Plus, Featured products, Profile action button, Business details and events.'
					: 'Schedule posts, Saved folders, Profile customisation, Pinned posts, Follower growth.'}
			</p>
			{#if canStartWebCheckout(billing)}
				<button
					class="tier-action"
					id={`subscription-${tier}-details`}
					aria-label={promotionalGrants.length
						? `Subscribe to ${title(tier)}`
						: `View ${title(tier)} details`}
					disabled={work.busy}
					onclick={() => purchase(tier)}
					>{purchasingTier === tier
						? 'Opening checkout…'
						: promotionalGrants.length
							? `Subscribe to ${title(tier)}`
							: 'View details'}</button
				>
			{/if}
		</section>
	{/each}
	<p class="muted checkout-note">
		You can review billing details before paying securely with Stripe.
		{#if prices.length && prices.every((price) => price.taxBehavior === 'inclusive')}Tax
			included.{:else}Final price and tax are shown before you pay.{/if}
	</p>
{/if}

<style>
	.tier-card.panel {
		position: relative;
		--tier-accent: var(--primary);
		background: color-mix(in srgb, var(--tier-accent) 8%, var(--surface));
	}
	.business-tier.panel {
		--tier-accent: light-dark(#765a00, #f7d46a);
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
		border: 1.5px solid var(--tier-accent);
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		background: color-mix(in srgb, var(--tier-accent) 16%, var(--surface));
		color: var(--tier-accent);
		flex-shrink: 0;
	}
	.tier-status {
		color: var(--tier-accent);
		font-size: 14px;
	}
	.access-title {
		font-size: 16px;
		margin: 0;
	}
	.account-plan {
		margin: 0 0 16px;
	}
	.tier-heading h2 {
		font-size: 20px;
		line-height: 1.3;
	}
	.tier-price {
		margin: 2px 0 0;
		color: var(--tier-accent);
		font-weight: 700;
	}
	.tier-benefits {
		margin: 0 0 10px;
	}
	.tier-card .tier-action {
		color: var(--primary);
		border: 0;
		background: transparent;
		box-shadow: none;
		font-weight: 700;
		padding: 10px 12px;
	}
	.tier-action::after {
		content: '';
		position: absolute;
		inset: 0;
		border-radius: 14px;
	}
	.tier-action:focus-visible {
		outline: none;
	}
	.tier-card:has(.tier-action:focus-visible) {
		outline: 3px solid var(--primary);
		outline-offset: 3px;
	}
	.tier-card:has(.tier-action:not(:disabled)):hover {
		box-shadow: 4px 4px 0 var(--shadow);
	}

	.checkout-note {
		font-size: 14px;
	}
</style>
