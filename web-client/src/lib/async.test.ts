import { it, expect, vi } from 'vitest';
import { withDeadline } from './async';
import { APITransport, MutationOutcomeUnknownError } from './api/transport';
it('stops waiting and aborts a dependency which never resolves', async () => {
	vi.useFakeTimers();
	try {
		let signal!: AbortSignal;
		const task = withDeadline(100, undefined, (s) => {
			signal = s;
			return new Promise(() => {});
		});
		const result = expect(task).rejects.toMatchObject({ name: 'TimeoutError' });
		await vi.advanceTimersByTimeAsync(101);
		await result;
		expect(signal.aborted).toBe(true);
	} finally {
		vi.useRealTimers();
	}
});
it('keeps a stalled mutation response uncertain with its exact key', async () => {
	vi.useFakeTimers();
	try {
		const api = new APITransport(
			async () => new Response(new ReadableStream())
		);
		const key = crypto.randomUUID();
		const task = api.request('POST', '/v1/posts', {
			operationKey: key,
			body: { text: 'PRIVATE_CANARY' }
		});
		const result = expect(task).rejects.toMatchObject({
			name: 'MutationOutcomeUnknownError',
			operationKey: key
		});
		await vi.advanceTimersByTimeAsync(120001);
		await result;
		await task.catch((error) =>
			expect(error).toBeInstanceOf(MutationOutcomeUnknownError)
		);
	} finally {
		vi.useRealTimers();
	}
});
