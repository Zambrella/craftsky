const maximumBytes = 2 * 1024 * 1024;

/** Only the public Bluesky avatar CDN is allowed; never forward caller credentials. */
export async function profileShareAvatar(
	source: string | null,
	fetcher: typeof fetch = fetch
): Promise<Response> {
	const unavailable = (status: number) =>
		new Response(null, { status, headers: { 'cache-control': 'no-store' } });
	let target: URL;
	try {
		target = new URL(source || '');
	} catch {
		return unavailable(400);
	}
	if (
		target.origin !== 'https://cdn.bsky.app' ||
		target.username ||
		target.password ||
		target.search ||
		target.hash ||
		!/^\/img\/avatar\/plain\/did:(?:plc|web):[A-Za-z0-9._:%-]+\/[A-Za-z0-9]+@(jpeg|png|webp|gif)$/.test(
			target.pathname
		)
	)
		return unavailable(400);
	try {
		const response = await fetcher(target.href, {
			credentials: 'omit',
			redirect: 'manual',
			signal: AbortSignal.timeout(8000),
			headers: { accept: 'image/jpeg,image/png,image/webp,image/gif' }
		});
		const mime = response.headers.get('content-type')?.split(';')[0].trim();
		if (
			!response.ok ||
			!mime ||
			!['image/jpeg', 'image/png', 'image/webp', 'image/gif'].includes(mime)
		) {
			await response.body?.cancel();
			return unavailable(502);
		}
		if (Number(response.headers.get('content-length')) > maximumBytes) {
			await response.body?.cancel();
			return unavailable(413);
		}
		const reader = response.body?.getReader();
		if (!reader) return unavailable(502);
		const chunks: Uint8Array[] = [];
		let length = 0;
		try {
			for (;;) {
				const next = await reader.read();
				if (next.done) break;
				length += next.value.byteLength;
				if (length > maximumBytes) {
					await reader.cancel();
					return unavailable(413);
				}
				chunks.push(next.value);
			}
		} finally {
			reader.releaseLock();
		}
		const bytes = new Uint8Array(length);
		let offset = 0;
		for (const chunk of chunks) {
			bytes.set(chunk, offset);
			offset += chunk.byteLength;
		}
		return new Response(bytes, {
			headers: {
				'content-type': mime,
				'cache-control': 'no-store',
				'x-content-type-options': 'nosniff'
			}
		});
	} catch {
		return unavailable(502);
	}
}
