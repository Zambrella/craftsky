import { abortable } from '#lib/async.ts';

export const videoThumbnailPercents = [0, 25, 50, 75] as const;
export type VideoThumbnailPercent = (typeof videoThumbnailPercents)[number];

export function videoThumbnailPercent(value: unknown): VideoThumbnailPercent {
	return videoThumbnailPercents.find((percent) => percent === value) ?? 50;
}

export function videoThumbnailTimeMs(
	durationMs: number,
	percent: VideoThumbnailPercent
): number {
	return Math.floor((durationMs * percent) / 100);
}

export async function captureVideoThumbnailFrames(
	source: Blob,
	durationMs: number | undefined,
	signal: AbortSignal,
	onFrame: (percent: VideoThumbnailPercent, image: Blob) => void
): Promise<void> {
	const video = document.createElement('video');
	const url = URL.createObjectURL(source);
	try {
		video.muted = true;
		video.preload = 'auto';
		await abortable(
			new Promise<void>((resolve, reject) => {
				video.onloadeddata = () => resolve();
				video.onerror = () => reject(new Error('Video preview unavailable'));
				video.src = url;
			}),
			signal
		);
		const duration = durationMs ?? Math.round(video.duration * 1000);
		if (
			!Number.isFinite(duration) ||
			duration <= 0 ||
			!video.videoWidth ||
			!video.videoHeight
		)
			return;
		const canvas = document.createElement('canvas');
		const scale = Math.min(
			1,
			320 / Math.max(video.videoWidth, video.videoHeight)
		);
		canvas.width = Math.max(1, Math.round(video.videoWidth * scale));
		canvas.height = Math.max(1, Math.round(video.videoHeight * scale));
		const context = canvas.getContext('2d');
		if (!context) return;

		for (const percent of videoThumbnailPercents) {
			signal.throwIfAborted();
			const time = videoThumbnailTimeMs(duration, percent) / 1000;
			if (video.currentTime !== time) {
				await abortable(
					new Promise<void>((resolve, reject) => {
						video.onseeked = () => resolve();
						video.onerror = () => reject(new Error('Video frame unavailable'));
						video.currentTime = time;
					}),
					signal
				);
			}
			context.drawImage(video, 0, 0, canvas.width, canvas.height);
			const image = await abortable(
				new Promise<Blob | null>((resolve) =>
					canvas.toBlob(resolve, 'image/jpeg', 0.85)
				),
				signal
			);
			if (image) onFrame(percent, image);
		}
	} finally {
		video.onseeked = null;
		video.onloadeddata = null;
		video.onerror = null;
		video.removeAttribute('src');
		video.load();
		URL.revokeObjectURL(url);
	}
}
