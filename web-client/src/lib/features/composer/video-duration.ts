// MP4 and QuickTime carry duration in the movie header, independent of whether
// the browser has a decoder for the video track. Skip media data by box size.
export async function containerDurationMs(
	file: Blob
): Promise<number | undefined> {
	async function findBox(type: string, start: number, end: number) {
		for (
			let offset = start, count = 0;
			offset + 8 <= end && count < 4096;
			count++
		) {
			const header = new DataView(
				await file.slice(offset, Math.min(offset + 16, end)).arrayBuffer()
			);
			const name = String.fromCharCode(...new Uint8Array(header.buffer, 4, 4));
			let size = header.getUint32(0);
			let headerSize = 8;
			if (size === 1) {
				if (header.byteLength < 16) return;
				size = Number(header.getBigUint64(8));
				headerSize = 16;
			} else if (size === 0) {
				size = end - offset;
			}
			if (size < headerSize || size > end - offset) return;
			if (name === type)
				return { start: offset + headerSize, end: offset + size };
			offset += size;
		}
	}
	const movie = await findBox('moov', 0, file.size);
	if (!movie) return;
	const header = await findBox('mvhd', movie.start, movie.end);
	if (!header) return;
	const data = new DataView(
		await file
			.slice(header.start, Math.min(header.start + 32, header.end))
			.arrayBuffer()
	);
	if (data.byteLength < 20) return;
	const version = data.getUint8(0);
	if (version !== 0 && (version !== 1 || data.byteLength < 32)) return;
	const timescale = data.getUint32(version === 0 ? 12 : 20);
	const duration =
		version === 0 ? data.getUint32(16) : Number(data.getBigUint64(24));
	if (timescale && duration > 0 && Number.isSafeInteger(duration))
		return Math.round((duration * 1000) / timescale);
}
