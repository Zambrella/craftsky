import { abortable } from '#lib/async.ts';
import { UpstreamError } from './upstream';

export function bodyLimit(method: string, path: string): number {
	return (method === 'POST' && path === '/v1/blobs/images') ||
		(method === 'PUT' && /^\/v1\/scheduled-post-media\/[^/]+$/.test(path))
		? 32 * 1024 * 1024
		: 1024 * 1024;
}
/** Pull at upstream speed, retaining at most the current chunk, never a second whole-body copy. */
export function boundedBody(
	request: Request,
	limit: number,
	signal: AbortSignal
) {
	const declared = request.headers.get('content-length');
	if (declared && (!/^\d+$/.test(declared) || Number(declared) > limit))
		throw new UpstreamError(413, 'body_too_large');
	const reader = request.body?.getReader();
	let failure: unknown;
	let length = 0;
	const stop = () => {
		void reader?.cancel().catch(() => {});
	};
	signal.addEventListener('abort', stop, { once: true });
	const body = reader
		? new ReadableStream<Uint8Array>(
				{
					async pull(controller) {
						try {
							const next = await abortable(reader.read(), signal);
							if (next.done) {
								controller.close();
								return;
							}
							length += next.value.byteLength;
							if (length > limit)
								throw new UpstreamError(413, 'body_too_large');
							controller.enqueue(next.value);
						} catch (error) {
							failure = error;
							controller.error(error);
							stop();
						}
					},
					cancel() {
						stop();
					}
				},
				{ highWaterMark: 0 }
			)
		: undefined;
	return {
		body,
		failure: () => failure,
		close() {
			signal.removeEventListener('abort', stop);
			stop();
		}
	};
}
