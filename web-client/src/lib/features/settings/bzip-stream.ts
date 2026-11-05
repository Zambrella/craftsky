import type { DecompressionStreamOptions } from '@zip.js/zip.js';
import { assertBzipSize, bzipLimits } from './bzip-limits.ts';

/** One worker per selected entry. Cancellation and deadline always terminate it. */
export function runBzipWorker(
	input: Uint8Array<ArrayBuffer>,
	expectedSize: number,
	signal: AbortSignal
): Promise<Uint8Array<ArrayBuffer>> {
	assertBzipSize(expectedSize);
	if (input.byteLength > bzipLimits.compressedBytes)
		return Promise.reject(
			new Error('BZip2 compressed input exceeds its limit.')
		);
	return new Promise((resolve, reject) => {
		if (signal.aborted) {
			reject(new DOMException('Import cancelled.', 'AbortError'));
			return;
		}
		const worker = new Worker(new URL('./bzip-worker.ts', import.meta.url), {
			type: 'module',
			name: 'craftsky-instagram-archive'
		});
		const done = (error?: Error, output?: Uint8Array<ArrayBuffer>) => {
			clearTimeout(timeout);
			signal.removeEventListener('abort', cancel);
			worker.terminate();
			if (error) reject(error);
			else resolve(output!);
		};
		const cancel = () =>
			done(new DOMException('Import cancelled.', 'AbortError'));
		const timeout = setTimeout(
			() => done(new Error('BZip2 decompression timed out.')),
			bzipLimits.deadlineMs
		);
		worker.onerror = () => done(new Error('BZip2 decompression failed.'));
		worker.onmessageerror = () =>
			done(new Error('Invalid BZip2 worker result.'));
		worker.onmessage = (event: MessageEvent<{ output?: unknown }>) => {
			const output = event.data?.output;
			if (!(output instanceof Uint8Array) || output.byteLength !== expectedSize)
				done(new Error('Invalid BZip2 worker result.'));
			else done(undefined, output as Uint8Array<ArrayBuffer>);
		};
		signal.addEventListener('abort', cancel, { once: true });
		try {
			worker.postMessage({ input, expectedSize }, [input.buffer]);
		} catch {
			done(new Error('BZip2 worker could not start.'));
		}
	});
}

/** zip.js retains its strict header, ZIP64 and ZIP CRC validation around this codec. */
export class Bzip2DecompressionStream {
	static supportedFormats = ['bzip2'];
	readonly readable: ReadableStream<Uint8Array<ArrayBuffer>>;
	readonly writable: WritableStream<Uint8Array>;
	constructor(_format: string, options?: DecompressionStreamOptions) {
		const expectedSize = options?.uncompressedSize;
		if (expectedSize === undefined)
			throw new Error('Missing BZip2 entry size.');
		assertBzipSize(expectedSize);
		const abort = new AbortController();
		let output: ReadableStreamDefaultController<Uint8Array<ArrayBuffer>>;
		let chunks: Uint8Array<ArrayBuffer>[] = [],
			length = 0;
		this.readable = new ReadableStream({
			start(controller) {
				output = controller;
			},
			cancel() {
				chunks = [];
				abort.abort();
			}
		});
		this.writable = new WritableStream({
			write(chunk) {
				if (abort.signal.aborted)
					throw new DOMException('Import cancelled.', 'AbortError');
				if (length + chunk.byteLength > bzipLimits.compressedBytes) {
					chunks = [];
					const error = new Error('BZip2 compressed input exceeds its limit.');
					output.error(error);
					throw error;
				}
				length += chunk.byteLength;
				chunks.push(new Uint8Array(chunk));
			},
			async close() {
				try {
					const input = new Uint8Array(length);
					let offset = 0;
					for (const chunk of chunks) {
						input.set(chunk, offset);
						offset += chunk.byteLength;
					}
					chunks = [];
					const decoded = await runBzipWorker(
						input,
						expectedSize,
						abort.signal
					);
					output.enqueue(decoded);
					output.close();
				} catch (error) {
					if (!abort.signal.aborted) output.error(error);
					throw error;
				}
			},
			abort(error) {
				chunks = [];
				abort.abort();
				output.error(error);
			}
		});
	}
}
