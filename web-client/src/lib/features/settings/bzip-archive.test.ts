import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { parseInstagramExport } from './archive.ts';
import { decodeBzip2 } from './bzip-decode.ts';
import fixture from './bzip-fixture.json';

class DecoderWorker {
	onmessage: ((event: MessageEvent) => void) | null = null;
	terminated = false;
	postMessage(value: { input: Uint8Array; expectedSize: number }) {
		queueMicrotask(() => {
			if (this.terminated) return;
			try {
				this.onmessage?.(
					new MessageEvent('message', {
						data: { output: decodeBzip2(value.input, value.expectedSize) }
					})
				);
			} catch {
				this.onmessage?.(
					new MessageEvent('message', { data: { error: 'Invalid entry.' } })
				);
			}
		});
	}
	terminate() {
		this.terminated = true;
	}
}
const zip = () =>
	Uint8Array.from(atob(fixture.zip), (byte) => byte.charCodeAt(0));
function directory(bytes: Uint8Array) {
	const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
	for (let i = 0; i <= bytes.length - 4; i++)
		if (view.getUint32(i, true) === 0x02014b50) return i;
	throw new Error('Fixture has no central directory.');
}
beforeEach(() => vi.stubGlobal('Worker', DecoderWorker));
afterEach(() => vi.unstubAllGlobals());
describe('Instagram BZip2 ZIP method 12', () => {
	it('imports a real BZip2 ZIP through the strict ZIP reader and bounded byte decoder', async () => {
		expect((await parseInstagramExport(new Blob([zip()]))).entries).toEqual([
			{ username: 'maker.one' }
		]);
	});
	it('still enforces the ZIP CRC after the BZip2 stream CRC succeeds', async () => {
		const bytes = zip(),
			view = new DataView(bytes.buffer),
			at = directory(bytes);
		view.setUint32(14, 1, true);
		view.setUint32(at + 16, 1, true);
		await expect(parseInstagramExport(new Blob([bytes]))).rejects.toThrow(
			'invalid or damaged'
		);
	});
	it('rejects false size claims before allowing output beyond the declared or absolute bound', async () => {
		for (const size of [1, 20 * 1024 * 1024 + 1]) {
			const bytes = zip(),
				view = new DataView(bytes.buffer),
				at = directory(bytes);
			view.setUint32(22, size, true);
			view.setUint32(at + 24, size, true);
			await expect(parseInstagramExport(new Blob([bytes]))).rejects.toThrow();
		}
	});
	it('rejects mismatched local headers before decoding', async () => {
		const bytes = zip();
		bytes[30] = 'X'.charCodeAt(0);
		await expect(parseInstagramExport(new Blob([bytes]))).rejects.toThrow(
			'invalid or damaged'
		);
	});
});
