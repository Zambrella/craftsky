import type { SubscriptionAccess } from '#lib/api/types.ts';

export const subscriptionAccessContext = Symbol('subscription-access');

/** Confirmed AppView access shared by one browser account's feature controls. */
export class SubscriptionAccessState {
	value = $state<SubscriptionAccess>();
	private active = true;
	private revision = 0;

	constructor(
		private readonly account: string,
		private readonly did: string,
		private readonly currentAccount: () => string
	) {}

	/** Capture before reading so an older boot or refresh cannot replace newer state. */
	beginRead(account: string): (value: SubscriptionAccess) => boolean {
		if (!this.isCurrent(account)) return () => false;
		const revision = ++this.revision;
		return (value) => {
			if (
				!this.isCurrent(account) ||
				revision !== this.revision ||
				value.did !== this.did
			)
				return false;
			this.value = value;
			return true;
		};
	}

	dispose() {
		this.active = false;
		this.revision++;
	}

	private isCurrent(account: string) {
		return (
			this.active &&
			account === this.account &&
			this.currentAccount() === this.account
		);
	}
}
