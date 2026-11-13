import type { BillingState, BillingSubscription } from '#lib/api/types.ts';
import { UserFacingError } from '#lib/user-facing-error.ts';

export type SubscriptionTier = 'plus' | 'business';

/** Provider state, rather than the displayed effective tier, prevents double billing. */
export function isCurrentSubscription(
	subscription: BillingSubscription,
	now = Date.now()
): boolean {
	const periods = [subscription.endsAt, subscription.currentPeriodEndsAt];
	return (
		subscription.givesAccess ||
		subscription.pendingPayment ||
		!['expired', 'refunded', 'revoked'].includes(subscription.status) ||
		periods.some(
			(ends) =>
				!!(
					ends &&
					(!Number.isFinite(Date.parse(ends)) || Date.parse(ends) > now)
				)
		)
	);
}

export function hasCurrentSubscription(state: BillingState): boolean {
	return state.subscriptions.some((subscription) =>
		isCurrentSubscription(subscription)
	);
}

export function subscriptionNeedsRefresh(state: BillingState): boolean {
	const checked = Date.parse(state.reconciledAt || '');
	return (
		!Number.isFinite(checked) ||
		Date.now() - checked > 5 * 60 * 1000 ||
		checked - Date.now() > 60 * 1000 ||
		state.reconciliationStale ||
		state.requestedGeneration > state.reconciledGeneration
	);
}

export function canStartWebCheckout(state: BillingState): boolean {
	return (
		state.webCheckoutAvailable === true &&
		!subscriptionNeedsRefresh(state) &&
		!hasCurrentSubscription(state) &&
		!state.subscriptions.some(
			(subscription) => subscription.anomaly && subscription.anomaly !== 'none'
		) &&
		!state.licenses.some(
			(license) => license.anomaly && license.anomaly !== 'none'
		)
	);
}

export function subscriptionTier(
	state: BillingState,
	subscription: BillingSubscription
): SubscriptionTier | undefined {
	return (
		subscription.tier ||
		state.licenses.find((license) => license.subscriptionId === subscription.id)
			?.tier
	);
}

export function subscriptionProvider(store: string): string {
	if (store === 'app_store') return 'App Store';
	if (store === 'play_store') return 'Google Play';
	if (store === 'stripe') return 'Stripe';
	return 'your billing provider';
}

/** Stripe session URLs are private capabilities; never report their contents. */
export function hostedCheckoutURL(value: string): string {
	// Match the original string so URL normalization cannot hide credentials,
	// ports, encoded path segments or control characters. Stripe's opaque
	// fragment is required by Checkout and must reach Stripe unchanged.
	if (
		typeof value !== 'string' ||
		/[\s\p{Cc}]/u.test(value) ||
		!/^https:\/\/checkout\.stripe\.com\/c\/pay\/cs_(?:test|live)_[A-Za-z0-9]+(?:#\S+)?$/.test(
			value
		)
	)
		throw new UserFacingError(
			'Checkout is unavailable. Please try again later.'
		);
	return value;
}

/** Customer-specific portal sessions are private capabilities, not email login links. */
export function stripePortalURL(value: string): string {
	if (
		typeof value !== 'string' ||
		/[\s\p{Cc}]/u.test(value) ||
		!/^https:\/\/billing\.stripe\.com\/p\/session\/[A-Za-z0-9_-]+(?:#\S+)?$/.test(
			value
		)
	)
		throw new UserFacingError(
			'Subscription management is unavailable. Please try again later.'
		);
	return value;
}

/** A late response for a previous account must never launch its billing page. */
class AccountBillingNavigation<Args extends unknown[]> {
	private active = true;
	private pending = false;
	constructor(
		private readonly account: string,
		private readonly currentAccount: () => string,
		private readonly request: (...args: Args) => Promise<{ url: string }>,
		private readonly navigate: (url: string) => void,
		private readonly destination: (value: string) => string
	) {}

	dispose() {
		this.active = false;
	}

	async start(...args: Args): Promise<void> {
		if (!this.isCurrent() || this.pending) return;
		this.pending = true;
		try {
			const result = await this.request(...args);
			if (this.isCurrent()) this.navigate(this.destination(result.url));
		} finally {
			this.pending = false;
		}
	}

	private isCurrent() {
		return this.active && this.currentAccount() === this.account;
	}
}

export class SubscriptionCheckout extends AccountBillingNavigation<
	[SubscriptionTier]
> {
	constructor(
		account: string,
		currentAccount: () => string,
		request: (tier: SubscriptionTier) => Promise<{ url: string }>,
		navigate: (url: string) => void
	) {
		super(account, currentAccount, request, navigate, hostedCheckoutURL);
	}
}

export class SubscriptionManagement extends AccountBillingNavigation<[]> {
	constructor(
		account: string,
		currentAccount: () => string,
		request: () => Promise<{ url: string }>,
		navigate: (url: string) => void
	) {
		super(account, currentAccount, request, navigate, stripePortalURL);
	}
}
