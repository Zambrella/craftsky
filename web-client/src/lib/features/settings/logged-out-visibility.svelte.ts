import {
	APIError,
	MutationOutcomeUnknownError,
	newOperationKey
} from '#lib/api/transport.ts';
import type { VisibilityIntent, VisibilitySetting } from '#lib/api/types.ts';
export type { VisibilityIntent, VisibilitySetting } from '#lib/api/types.ts';
export interface VisibilityPort {
	read: () => Promise<VisibilitySetting>;
	save: (
		body: Omit<VisibilityIntent, 'operationKey'>,
		key: string
	) => Promise<VisibilitySetting>;
}
export class LoggedOutVisibility {
	value = $state<VisibilitySetting>();
	pending = $state<VisibilityIntent>();
	busy = $state(false);
	error = $state('');
	notice = $state('');
	private active = true;
	constructor(
		private port: VisibilityPort,
		private key = newOperationKey
	) {}
	dispose() {
		this.active = false;
	}
	async load() {
		if (this.busy || !this.active) return;
		this.busy = true;
		this.error = '';
		try {
			const value = await this.port.read();
			if (this.active) {
				this.value = value;
				this.pending = value.pending;
			}
		} catch {
			if (this.active)
				this.error =
					'Could not read your shared Bluesky preference. Retry to load it.';
		} finally {
			if (this.active) this.busy = false;
		}
	}
	async setHidden(hideFromLoggedOut: boolean) {
		if (!this.value || this.pending || this.busy || !this.active) return;
		this.pending = {
			hideFromLoggedOut,
			expectedCid: this.value.expectedCid,
			operationKey: this.key()
		};
		await this.retry();
	}
	async retry() {
		const intent = this.pending;
		if (!intent || this.busy || !this.active) return;
		this.busy = true;
		this.error = '';
		this.notice = '';
		try {
			const { operationKey, ...body } = intent;
			const value = await this.port.save(body, operationKey);
			if (this.active) {
				this.value = value;
				this.pending = value.pending;
				this.notice = value.pending
					? ''
					: 'Saved. This preference also applies on Bluesky.';
			}
		} catch (error) {
			if (!this.active) return;
			if (error instanceof MutationOutcomeUnknownError) {
				this.error = 'Your save is still being confirmed. Retry the same save.';
			} else if (
				error instanceof APIError &&
				[400, 401, 403, 409, 410, 422].includes(error.status)
			) {
				this.pending = undefined;
				this.value = undefined;
				this.error =
					error.status === 409
						? 'Your profile changed. Reload before changing this preference.'
						: 'Could not save. Sign in again or reload your preference.';
			} else {
				this.error = 'Could not confirm your save. Retry the same save.';
			}
		} finally {
			if (this.active) this.busy = false;
		}
	}
}
