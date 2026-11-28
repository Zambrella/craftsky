import { abortable, withDeadline } from '#lib/async.ts';
import { IMPORT_SAFETY_POLICY as limits } from './safety';

function fitsThumbnailLimits(width: number, height: number): boolean {
	return (
		width > 0 &&
		height > 0 &&
		width <= limits.maxDimension &&
		height <= limits.maxDimension &&
		width * height <= limits.maxDecodedPixels
	);
}

async function loadPhoto(
	image: HTMLImageElement,
	url: string,
	signal: AbortSignal
) {
	await abortable(
		new Promise<void>((resolve, reject) => {
			image.onload = () => resolve();
			image.onerror = () => reject(new Error('Photo preview unavailable'));
			image.src = url;
		}),
		signal
	);
	return { width: image.naturalWidth, height: image.naturalHeight };
}

async function loadVideoFrame(
	video: HTMLVideoElement,
	url: string,
	signal: AbortSignal
) {
	video.muted = true;
	video.playsInline = true;
	video.preload = 'metadata';
	await abortable(
		new Promise<void>((resolve, reject) => {
			video.onloadedmetadata = () => resolve();
			video.onerror = () => reject(new Error('Video preview unavailable'));
			video.src = url;
		}),
		signal
	);
	const { videoWidth: width, videoHeight: height, duration } = video;
	if (
		!fitsThumbnailLimits(width, height) ||
		!Number.isFinite(duration) ||
		duration <= 0
	)
		return;

	// Seek just beyond the opening frame, or to the middle of a short clip.
	await abortable(
		new Promise<void>((resolve, reject) => {
			video.onseeked = () => resolve();
			video.onerror = () => reject(new Error('Video frame unavailable'));
			video.currentTime = Math.min(1, duration / 2);
		}),
		signal
	);
	return { width, height };
}

/** Decode one local archive attachment without retaining the original media. */
export async function renderInstagramThumbnail(
	source: Blob,
	signal: AbortSignal
): Promise<Blob | undefined> {
	try {
		return await withDeadline(5_000, signal, async (decodeSignal) => {
			const frame = source.type.startsWith('video/')
				? document.createElement('video')
				: document.createElement('img');
			const url = URL.createObjectURL(source);
			try {
				const dimensions =
					frame instanceof HTMLVideoElement
						? await loadVideoFrame(frame, url, decodeSignal)
						: await loadPhoto(frame, url, decodeSignal);
				decodeSignal.throwIfAborted();
				if (
					!dimensions ||
					!fitsThumbnailLimits(dimensions.width, dimensions.height)
				)
					return;
				const { width, height } = dimensions;
				const scale = Math.min(1, 160 / Math.max(width, height));
				const canvas = document.createElement('canvas');
				canvas.width = Math.max(1, Math.round(width * scale));
				canvas.height = Math.max(1, Math.round(height * scale));
				const context = canvas.getContext('2d');
				if (!context) return;
				context.drawImage(frame, 0, 0, canvas.width, canvas.height);
				return await abortable(
					new Promise<Blob | undefined>((resolve) => {
						canvas.toBlob(
							(blob) => resolve(blob ?? undefined),
							'image/jpeg',
							0.75
						);
					}),
					decodeSignal
				);
			} finally {
				frame.onerror = null;
				if (frame instanceof HTMLVideoElement) {
					frame.onloadedmetadata = null;
					frame.onseeked = null;
					frame.removeAttribute('src');
					frame.load();
				} else {
					frame.onload = null;
					frame.removeAttribute('src');
				}
				URL.revokeObjectURL(url);
			}
		});
	} catch {
		signal.throwIfAborted();
	}
}
