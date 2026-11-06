import type { DiagnosticOperation } from '../diagnostics/privacy';
import { isCanceled } from '#lib/api/error-message.ts';
import { apiErrorMessage } from '#lib/api/error-message.ts';
import { notify } from './toasts.svelte';
import { presentFieldErrors } from '#lib/field-validation.ts';
import { APIError, MutationOutcomeUnknownError } from '#lib/api/client.ts';
import { reportFailure } from '../diagnostics/client.ts';
export class Work {
	constructor(private operationName: DiagnosticOperation = 'social') {}
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
		notice?: string,
		label: DiagnosticOperation = this.operationName
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
			if (isCanceled(error)) return;
			if (this.active && presentFieldErrors(error)) return;
			void reportFailure(error, label);
			if (this.active) {
				this.error =
					error instanceof APIError
						? apiErrorMessage(error)
						: 'Could not complete this request. Please try again.';
				if (
					error instanceof MutationOutcomeUnknownError ||
					(error instanceof APIError && error.retryable)
				)
					this.retry = () => this.run(operation, accept, notice, label);
			}
		} finally {
			if (this.active) {
				this.busy = false;
				this.hasLoaded = true;
			}
		}
	}
}
