import { describe, expect, it, vi } from 'vitest';
import { APIError, MutationOutcomeUnknownError } from '#lib/api/client.ts';
import { Work } from './work.svelte.ts';
vi.mock('../diagnostics/client.ts', () => ({ reportFailure: vi.fn() }));

describe('social async work', () => {
	it('does not retain an older retry after a non-retryable failure', async () => {
		const work = new Work();
		await work.run(async () => {
			throw new APIError(503, 'unavailable', 'Retry later');
		});
		expect(work.retry).toBeTypeOf('function');
		await work.run(async () => {
			throw new APIError(400, 'invalid', 'Invalid request');
		});
		expect(work.retry).toBeUndefined();
		expect(work.error).toBe('Invalid request');
		expect(work.busy).toBe(false);
	});
	it('retries the original operation and clears retry state on success', async () => {
		const work = new Work();
		const operation = vi
			.fn()
			.mockRejectedValueOnce(new MutationOutcomeUnknownError('original-key'))
			.mockResolvedValueOnce('saved');
		const accept = vi.fn();
		await work.run(operation, accept);
		await work.retry?.();
		expect(operation).toHaveBeenCalledTimes(2);
		expect(accept).toHaveBeenCalledWith('saved');
		expect(work.retry).toBeUndefined();
	});
	it('ignores late completions and never starts work after disposal', async () => {
		const work = new Work();
		let finish!: (value: string) => void;
		const accept = vi.fn();
		const pending = work.run(
			() =>
				new Promise<string>((resolve) => {
					finish = resolve;
				}),
			accept
		);
		work.dispose();
		finish('old account');
		await pending;
		expect(accept).not.toHaveBeenCalled();
		const idle = new Work();
		idle.dispose();
		const operation = vi.fn();
		await idle.run(operation);
		expect(operation).not.toHaveBeenCalled();
	});
});
