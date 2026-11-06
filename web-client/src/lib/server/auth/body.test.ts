import { describe, it, expect } from 'vitest';
import { boundedBody, bodyLimit } from './body';
const request = (body: ReadableStream<Uint8Array>, length?: string) =>
	new Request('https://app.craftsky.social/api/appview/v1/posts', {
		method: 'POST',
		body,
		duplex: 'half',
		headers: length ? { 'content-length': length } : {}
	} as RequestInit);
describe('bounded streaming proxy bodies', () => {
	it('rejects declared oversize without reading and applies small limits to JSON routes', () => {
		let reads = 0;
		const input = request(
			new ReadableStream(
				{
					pull() {
						reads++;
					}
				},
				{ highWaterMark: 0 }
			),
			'1048577'
		);
		expect(() =>
			boundedBody(
				input,
				bodyLimit('POST', '/v1/posts'),
				new AbortController().signal
			)
		).toThrow();
		expect(reads).toBe(0);
		expect(bodyLimit('POST', '/v1/blobs/images')).toBe(32 * 1024 * 1024);
		expect(bodyLimit('POST', '/v1/posts')).toBe(1024 * 1024);
	});
	it('preserves backpressure and cancels a chunked body as soon as it exceeds the limit', async () => {
		let reads = 0,
			canceled = false;
		const input = boundedBody(
			request(
				new ReadableStream(
					{
						pull(controller) {
							reads++;
							controller.enqueue(new Uint8Array(4));
						},
						cancel() {
							canceled = true;
						}
					},
					{ highWaterMark: 0 }
				)
			),
			5,
			new AbortController().signal
		);
		expect(reads).toBe(0);
		const reader = input.body!.getReader();
		expect((await reader.read()).value?.length).toBe(4);
		expect(reads).toBe(1);
		await expect(reader.read()).rejects.toMatchObject({
			status: 413,
			code: 'body_too_large'
		});
		expect(canceled).toBe(true);
		input.close();
	});
	it('cancels a stalled incoming read on deadline', async () => {
		const abort = new AbortController();
		let canceled = false;
		const input = boundedBody(
			request(
				new ReadableStream({
					cancel() {
						canceled = true;
					}
				})
			),
			10,
			abort.signal
		);
		const read = input.body!.getReader().read();
		const result = expect(read).rejects.toMatchObject({ name: 'TimeoutError' });
		abort.abort(new DOMException('Timeout', 'TimeoutError'));
		await result;
		expect(canceled).toBe(true);
		input.close();
	});
});
