import { APIError, MutationOutcomeUnknownError } from '#lib/api/client.ts';
import { reportFailure } from '../diagnostics/client.ts';
export class Work {
	busy = $state(false);
	hasLoaded = $state(false);
	error = $state('');
	retry = $state<(() => Promise<void>) | undefined>();
	private active = true;
	dispose() {
		this.active = false;
	}
	async run<T>(
		operation: () => Promise<T>,
		accept: (value: T) => void = () => {}
	) {
		if (!this.active || this.busy) return;
		this.busy = true;
		this.error = '';
		this.retry = undefined;
		try {
			const value = await operation();
			if (this.active) {
				accept(value);
				this.retry = undefined;
			}
		} catch (error) {
			void reportFailure(error, 'workspace');
			if (this.active) {
				this.error =
					error instanceof APIError
						? error.message
						: 'Could not complete this request. Please try again.';
				if (
					error instanceof MutationOutcomeUnknownError ||
					(error instanceof APIError && error.retryable)
				)
					this.retry = () => this.run(operation, accept);
			}
		} finally {
			if (this.active) {
				this.busy = false;
				this.hasLoaded = true;
			}
		}
	}
}
