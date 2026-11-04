import { afterEach, describe, expect, it, vi } from 'vitest';
import { runBzipWorker } from './bzip-stream.ts';
import { bzipLimits } from './bzip-limits.ts';

class HangingWorker {
	static current: HangingWorker;
	terminated = false;
	onmessage: ((event: MessageEvent) => void) | null = null;
	onerror: (() => void) | null = null;
	onmessageerror: (() => void) | null = null;
	constructor() {
		HangingWorker.current = this;
	}
	postMessage() {}
	terminate() {
		this.terminated = true;
	}
}
afterEach(() => {
	vi.useRealTimers();
	vi.unstubAllGlobals();
});
describe('BZip2 worker lifecycle', () => {
	it('terminates a stalled worker at the hard CPU deadline', async () => {
		vi.useFakeTimers();
		vi.stubGlobal('Worker', HangingWorker);
		const result = runBzipWorker(
			new Uint8Array(4),
			1,
			new AbortController().signal
		);
		const assertion = expect(result).rejects.toThrow('timed out');
		await vi.advanceTimersByTimeAsync(bzipLimits.deadlineMs);
		await assertion;
		expect(HangingWorker.current.terminated).toBe(true);
	});
	it('terminates work when the owning import is cancelled', async () => {
		vi.stubGlobal('Worker', HangingWorker);
		const controller = new AbortController();
		const result = runBzipWorker(new Uint8Array(4), 1, controller.signal);
		controller.abort();
		await expect(result).rejects.toMatchObject({ name: 'AbortError' });
		expect(HangingWorker.current.terminated).toBe(true);
	});
	it('bounds worker output and terminates workers after completion or failure', async () => {
		vi.stubGlobal('Worker', HangingWorker);
		const result = runBzipWorker(
			new Uint8Array(4),
			1,
			new AbortController().signal
		);
		HangingWorker.current.onmessage!(
			new MessageEvent('message', { data: { output: new Uint8Array(2) } })
		);
		await expect(result).rejects.toThrow('Invalid BZip2 worker result');
		expect(HangingWorker.current.terminated).toBe(true);
	});
});
