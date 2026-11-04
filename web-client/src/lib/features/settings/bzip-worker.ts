/// <reference lib="webworker" />
import { decodeBzip2 } from './bzip-decode.ts';

const worker = self as unknown as DedicatedWorkerGlobalScope;
worker.onmessage = (
	event: MessageEvent<{ input: Uint8Array; expectedSize: number }>
) => {
	try {
		const output = decodeBzip2(event.data.input, event.data.expectedSize);
		worker.postMessage({ output }, [output.buffer]);
	} catch {
		// Never reflect imported content or the codec's detailed input errors.
		worker.postMessage({ error: 'Invalid or oversized BZip2 archive entry.' });
	} finally {
		worker.close();
	}
};
