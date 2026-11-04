import { Buffer as BrowserBuffer } from 'buffer/index.js';
import Bunzip from 'seek-bzip';
import { assertBzipSize, bzipLimits } from './bzip-limits.ts';

// seek-bzip uses Buffer for its small block tables. This module is imported only
// by the dedicated archive worker (and unit tests), never the window runtime.
const workerScope = globalThis as unknown as { Buffer?: typeof BrowserBuffer };
workerScope.Buffer ??= BrowserBuffer;

/** Decode one complete BZip2 stream without unbounded output allocation. */
export function decodeBzip2(
	input: Uint8Array,
	expectedSize: number,
	deadline = performance.now() + bzipLimits.deadlineMs
): Uint8Array<ArrayBuffer> {
	assertBzipSize(expectedSize);
	if (input.byteLength > bzipLimits.compressedBytes)
		throw new Error('BZip2 compressed input exceeds its limit.');
	let at = 0,
		written = 0;
	const output = new Uint8Array(expectedSize);
	const checkTime = () => {
		if (performance.now() >= deadline)
			throw new Error('BZip2 decompression timed out.');
	};
	checkTime();
	const source = {
		readByte() {
			if ((at & 0x3fff) === 0) checkTime();
			// Throwing rather than returning EOF prevents a truncated footer from
			// being accepted by the codec's optional eof() short circuit.
			if (at >= input.length) throw new Error('Truncated BZip2 stream.');
			return input[at++];
		},
		read(buffer: Uint8Array, offset: number, length: number) {
			for (let i = 0; i < length; i++) buffer[offset + i] = this.readByte();
			return length;
		}
	};
	Bunzip.decode(source, {
		writeByte(byte: number) {
			if ((written & 0x3fff) === 0) checkTime();
			if (written >= expectedSize)
				throw new Error('BZip2 output exceeds its declared size.');
			output[written++] = byte;
		}
	});
	checkTime();
	if (at !== input.length || written !== expectedSize)
		throw new Error('BZip2 stream size does not match its archive entry.');
	return output;
}
