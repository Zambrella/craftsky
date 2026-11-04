import { presentFieldErrors } from '#lib/field-validation.ts';
import { accountScope, message, pendingWrite } from './model.ts';
import { reportFailure } from '../diagnostics/client.ts';

export class Work {
	loading = $state(true);
	busy = $state(false);
	error = $state('');
	notice = $state('');
	retry = $state<(() => Promise<void>) | null>(null);
	private scope = accountScope();
	private active = true;
	dispose() {
		this.active = false;
		this.scope.invalidate();
	}
	async load<T>(request: () => Promise<T>, apply: (value: T) => void) {
		if (!this.active || this.busy || this.retry) return;
		this.scope.invalidate();
		const token = this.scope.token();
		this.loading = true;
		this.error = '';
		try {
			const value = await request();
			if (this.scope.current(token)) apply(value);
		} catch (error) {
			if (this.scope.current(token) && presentFieldErrors(error)) return;
			void reportFailure(error, 'workspace');
			if (this.scope.current(token)) this.error = message(error);
		} finally {
			if (this.scope.current(token)) this.loading = false;
		}
	}
	async run<T>(
		request: () => Promise<T>,
		apply: (value: T) => void = () => {},
		notice = 'Saved.'
	) {
		if (!this.active || this.busy || this.retry) return;
		const token = this.scope.token();
		this.busy = true;
		this.error = '';
		this.notice = '';
		this.retry = null;
		try {
			const value = await request();
			if (this.scope.current(token)) {
				apply(value);
				this.notice = notice;
			}
		} catch (error) {
			if (this.scope.current(token) && presentFieldErrors(error)) return;
			void reportFailure(error, 'workspace');
			if (this.scope.current(token)) {
				this.error = message(error);
				if (pendingWrite(error))
					this.retry = async () => {
						this.retry = null;
						await this.run(request, apply, notice);
					};
			}
		} finally {
			if (this.scope.current(token)) this.busy = false;
		}
	}
}
