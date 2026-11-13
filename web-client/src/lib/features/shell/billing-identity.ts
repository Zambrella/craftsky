import type { CraftskyAPI } from '#lib/api/client.ts';
import type { SubscriptionAccessState } from './subscription-access.svelte';

/** Optional registration for one active browser account, independent of app readiness. */
export class BillingIdentity {
	private active = true;
	private attempted = false;

	constructor(
		private readonly account: string,
		private readonly currentAccount: () => string,
		private readonly billing: Pick<CraftskyAPI['billing'], 'ensure' | 'access'>,
		private readonly access: SubscriptionAccessState,
		private readonly isAllowed: () => boolean = () => true
	) {}

	async synchronize() {
		if (this.attempted || !this.isCurrent()) return;
		this.attempted = true;
		const publish = this.access.beginRead(this.account);
		try {
			// AppView owns the UUID and queues provider registration. No RevenueCat
			// credentials or native store operations belong in this browser flow.
			await this.billing.ensure();
			if (!this.isCurrent()) return;
			const value = await this.billing.access();
			if (this.isCurrent()) publish(value);
		} catch {
			// Billing setup is optional. Its failure must not hide the signed-in
			// app; billing actions retry setup through their guarded flow.
		}
	}

	dispose() {
		this.active = false;
	}

	private isCurrent() {
		return (
			this.active && this.isAllowed() && this.currentAccount() === this.account
		);
	}
}
