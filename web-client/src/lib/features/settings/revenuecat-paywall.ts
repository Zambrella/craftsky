import type { Offering, PurchaseParams } from '@revenuecat/purchases-js';
import { UserFacingError } from '#lib/user-facing-error.ts';
export interface WebPaywallDetails {
	apiKey: string;
	appUserId: string;
	offering: 'plus' | 'business';
}
type SDK = typeof import('@revenuecat/purchases-js');
let owner: symbol | undefined;

/** One identified SDK/paywall scope. Never configure an anonymous customer. */
export class RevenueCatPaywall {
	private active = true;
	private token = Symbol();
	private sdk: InstanceType<SDK['Purchases']> | undefined;
	private purchasing = false;
	private release: (() => void) | undefined;
	constructor(
		private readonly details: WebPaywallDetails,
		private readonly beforePurchase: () => Promise<void>,
		private readonly current: () => boolean,
		private readonly load: () => Promise<SDK> = () =>
			import('@revenuecat/purchases-js')
	) {}
	private check() {
		if (!this.active || !this.current())
			throw new DOMException('View closed', 'AbortError');
	}
	close() {
		this.active = false;
		if (owner === this.token) {
			this.sdk?.close();
			owner = undefined;
		}
		this.release?.();
	}
	async present(
		target: HTMLElement,
		onReady: () => void,
		onPurchase: () => void,
		onCanceled: () => void,
		onFailure: (message: string) => void = () => {},
		presentation?: {
			purchaseTarget: HTMLElement;
			onCheckoutChange: (open: boolean) => void | Promise<void>;
		}
	) {
		const run = async () => {
			this.check();
			if (owner)
				throw new UserFacingError(
					'Another subscription window is already open. Close it and try again.'
				);
			owner = this.token;
			try {
				const { Purchases, LogLevel } = await this.load();
				this.check();
				Purchases.setLogLevel(LogLevel.Silent);
				const sdk = Purchases.configure({
					apiKey: this.details.apiKey,
					appUserId: this.details.appUserId
				});
				this.sdk = sdk;
				// The supported public purchase method is the single dispatch point used by
				// presentPaywall in the pinned SDK. Recheck server state on every purchase,
				// including retries after an error or a long-lived paywall.
				const purchase = sdk.purchase.bind(sdk);
				sdk.purchase = async (params: PurchaseParams) => {
					this.check();
					if (this.purchasing)
						throw new UserFacingError(
							'A payment attempt is already in progress. Finish or close it before trying again.'
						);
					this.purchasing = true;
					let checkoutStarted = false;
					try {
						await this.beforePurchase();
						this.check();
						checkoutStarted = true;
						await presentation?.onCheckoutChange(true);
						this.check();
						return await purchase(params);
					} finally {
						try {
							if (checkoutStarted && this.active && this.current())
								await presentation?.onCheckoutChange(false);
						} finally {
							this.purchasing = false;
						}
					}
				};
				const offerings = await sdk.getOfferings();
				this.check();
				const offering: Offering | undefined =
					offerings.all[this.details.offering];
				if (
					!offering?.availablePackages.some(
						(p) => p.identifier === '$rc_monthly'
					)
				)
					throw new UserFacingError(
						'This subscription option is unavailable right now.'
					);
				onReady();
				const dismissed = new Promise<void>((resolve) => {
					this.release = resolve;
				});
				const result = sdk
					.presentPaywall({
						offering,
						htmlTarget: target,
						purchaseHtmlTarget: presentation?.purchaseTarget ?? target,
						onBack: (close) => close(),
						listener: {
							onPurchaseError: (error) => {
								if ('errorCode' in error && error.errorCode === 1) return;
								if (this.active && this.current())
									onFailure(
										error instanceof UserFacingError
											? error.message
											: 'Payment could not be completed. Please try again.'
									);
							}
						}
					})
					.then(() => {
						if (this.active && this.current()) onPurchase();
					})
					.catch((error) => {
						if (!this.active || !this.current()) return;
						if (error?.errorCode === 1) {
							onCanceled();
							return;
						}
						throw new UserFacingError(
							'The subscription provider could not complete that action. Please try again.'
						);
					});
				await Promise.race([result, dismissed]);
			} finally {
				if (owner === this.token) {
					this.sdk?.close();
					owner = undefined;
				}
			}
		};
		if (typeof navigator !== 'undefined' && navigator.locks) {
			await navigator.locks.request(
				`craftsky-paywall:${this.details.appUserId}`,
				{ ifAvailable: true },
				async (lock) => {
					if (!lock)
						throw new UserFacingError(
							'This account already has a subscription window open in another tab.'
						);
					await run();
				}
			);
		} else await run();
	}
}
