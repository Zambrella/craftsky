import { notify } from './toasts.svelte';
import { presentFieldErrors } from '#lib/field-validation.ts';
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
		accept: (value: T) => void = () => {},
		notice?: string
	) {
		if (!this.active || this.busy) return;
		this.busy = true;
		this.error = '';
		this.retry = undefined;
		try {
			const value = await operation();
			if (this.active) {
				accept(value);
				if (notice) notify(notice);
				this.retry = undefined;
			}
		} catch (error) {
			if (this.active && presentFieldErrors(error)) return;
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
					this.retry = () => this.run(operation, accept, notice);
			}
		} finally {
			if (this.active) {
				this.busy = false;
				this.hasLoaded = true;
			}
		}
	}
}
