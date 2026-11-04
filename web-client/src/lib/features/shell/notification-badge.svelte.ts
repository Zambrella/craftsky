export type FinishNotificationSeen = (succeeded: boolean) => void;
export type BeginNotificationSeen = () => FinishNotificationSeen;
export const notificationBadgeContext = Symbol('notification-badge');

/** Account-local badge state, fenced against polls started before viewing activity. */
export class NotificationBadge {
	count = $state(0);
	private revision = 0;
	private pending = new Set<symbol>();
	private previous = 0;
	private acknowledged = false;

	capture() {
		return this.revision;
	}
	accept(count: number, revision: number) {
		if (!this.pending.size && revision === this.revision) this.count = count;
	}
	beginSeen(): FinishNotificationSeen {
		if (!this.pending.size) {
			this.previous = this.count;
			this.acknowledged = false;
		}
		const operation = Symbol('seen');
		this.pending.add(operation);
		this.revision++;
		this.count = 0;
		return (succeeded) => {
			if (!this.pending.delete(operation)) return;
			this.acknowledged ||= succeeded;
			this.revision++;
			if (!this.pending.size)
				this.count = this.acknowledged ? 0 : this.previous;
		};
	}
}
