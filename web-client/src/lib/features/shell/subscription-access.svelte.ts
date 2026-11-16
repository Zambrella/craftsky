import type { SubscriptionAccess } from '#lib/api/types.ts';

export const subscriptionAccessContext = Symbol('subscription-access');

/** Confirmed AppView access shared by one browser account's feature controls. */
export class SubscriptionAccessState {
	value = $state<SubscriptionAccess>();
	private active = true;
	private revision = 0;
	private checkedAt: number | undefined;

	constructor(
		private readonly account: string,
		private readonly did: string,
		private readonly currentAccount: () => string
	) {}

	/** Capture before reading so an older boot or refresh cannot replace newer state. */
	beginRead(account = this.account): (value: SubscriptionAccess) => boolean {
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
			this.checkedAt = Date.now();
			return true;
		};
	}

	/** Dates request a server refresh; paid grace/renewal access remains provider-owned. */
	needsRefresh(now = Date.now()): boolean {
		if (this.checkedAt === undefined || !this.value) return true;
		const endsAt = Date.parse(this.value.accessEndsAt || '');
		return (
			now - this.checkedAt >= 5 * 60_000 ||
			now < this.checkedAt ||
			(Number.isFinite(endsAt) && this.checkedAt < endsAt && endsAt <= now)
		);
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
