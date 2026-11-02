import type {
	BlobReference,
	CraftskyAPI,
	PostPayload
} from '#lib/api/client.ts';
import type { DraftMedia } from '../drafts/store.ts';

export interface ImageTransform {
	rotation?: number;
	crop?: { x: number; y: number; width: number; height: number };
}
export async function prepareImage(
	source: Blob,
	name: string,
	transform: ImageTransform = {}
): Promise<DraftMedia> {
	if (source.size > 40_000_000)
		throw new Error('Choose an image smaller than 40 MB.');
	const bitmap = await createImageBitmap(source);
	try {
		const crop = transform.crop || { x: 0, y: 0, width: 1, height: 1 };
		if (
			crop.x < 0 ||
			crop.y < 0 ||
			crop.width <= 0 ||
			crop.height <= 0 ||
			crop.x + crop.width > 1.001 ||
			crop.y + crop.height > 1.001
		)
			throw new Error('Choose a crop within the image.');
		const sw = Math.max(1, Math.round(bitmap.width * crop.width)),
			sh = Math.max(1, Math.round(bitmap.height * crop.height));
		const rotation = (((transform.rotation || 0) % 4) + 4) % 4;
		const rotated = rotation % 2 === 1;
		let width = rotated ? sh : sw,
			height = rotated ? sw : sh;
		const scale = Math.min(1, 4000 / width, 4000 / height);
		width = Math.max(1, Math.round(width * scale));
		height = Math.max(1, Math.round(height * scale));
		const canvas = document.createElement('canvas');
		canvas.width = width;
		canvas.height = height;
		const context = canvas.getContext('2d');
		if (!context)
			throw new Error('Image editing is unavailable in this browser.');
		context.fillStyle = 'white';
		context.fillRect(0, 0, width, height);
		context.translate(width / 2, height / 2);
		context.rotate((rotation * Math.PI) / 2);
		const dw = rotated ? height : width,
			dh = rotated ? width : height;
		context.drawImage(
			bitmap,
			bitmap.width * crop.x,
			bitmap.height * crop.y,
			sw,
			sh,
			-dw / 2,
			-dh / 2,
			dw,
			dh
		);
		const encode = (quality: number) =>
			new Promise<Blob>((resolve, reject) =>
				canvas.toBlob(
					(blob) =>
						blob
							? resolve(blob)
							: reject(new Error('Could not prepare this image.')),
					'image/jpeg',
					quality
				)
			);
		let blob = await encode(0.88);
		let quality = 0.78;
		while (blob.size > 2_000_000 && quality >= 0.28) {
			blob = await encode(quality);
			quality -= 0.1;
		}
		if (blob.size > 2_000_000)
			throw new Error('Crop this image further to fit the 2 MB image limit.');
		return {
			id: crypto.randomUUID(),
			name,
			blob,
			mime: blob.type,
			alt: '',
			width,
			height
		};
	} finally {
		bitmap.close();
	}
}
export async function inspectVideo(file: File): Promise<DraftMedia> {
	if (file.type !== 'video/mp4' || !file.size || file.size > 300_000_000)
		throw new Error('Choose an MP4 video up to 300 MB and ten minutes.');
	const url = URL.createObjectURL(file);
	const element = document.createElement('video');
	try {
		await new Promise<void>((resolve, reject) => {
			const timeout = setTimeout(() => {
				element.src = '';
				reject(new Error('Could not read this video.'));
			}, 15000);
			element.onloadedmetadata = () => {
				clearTimeout(timeout);
				resolve();
			};
			element.onerror = () => {
				clearTimeout(timeout);
				reject(new Error('This video cannot be played in your browser.'));
			};
			element.preload = 'metadata';
			element.src = url;
		});
		if (
			!Number.isFinite(element.duration) ||
			element.duration > 600 ||
			element.duration <= 0
		)
			throw new Error('Videos must be no longer than ten minutes.');
		return {
			id: crypto.randomUUID(),
			name: file.name,
			blob: file,
			mime: 'video/mp4',
			alt: '',
			width: element.videoWidth,
			height: element.videoHeight
		};
	} finally {
		element.removeAttribute('src');
		element.load();
		URL.revokeObjectURL(url);
	}
}
const videoOrigin = 'https://video.bsky.app';
async function boundedJSON(
	response: Response
): Promise<Record<string, unknown>> {
	if (!response.body)
		throw new Error('The video service did not return a response.');
	const reader = response.body.getReader();
	const chunks: Uint8Array[] = [];
	let size = 0;
	try {
		for (;;) {
			const part = await reader.read();
			if (part.done) break;
			size += part.value.length;
			if (size > 65536)
				throw new Error('The video service returned an invalid response.');
			chunks.push(part.value);
		}
	} finally {
		await reader.cancel().catch(() => {});
		reader.releaseLock();
	}
	const data = new Uint8Array(size);
	let offset = 0;
	for (const chunk of chunks) {
		data.set(chunk, offset);
		offset += chunk.length;
	}
	try {
		const value = JSON.parse(new TextDecoder().decode(data));
		if (!value || typeof value !== 'object' || Array.isArray(value)) throw 0;
		return value;
	} catch {
		throw new Error('The video service returned an invalid response.');
	}
}
function blobValue(value: unknown): BlobReference | undefined {
	if (!value || typeof value !== 'object') return;
	const blob = value as BlobReference;
	if (
		blob.$type === 'blob' &&
		typeof blob.ref?.$link === 'string' &&
		blob.mimeType === 'video/mp4' &&
		typeof blob.size === 'number' &&
		blob.size > 0 &&
		blob.size <= 300_000_000
	)
		return blob;
}
const delay = (milliseconds: number, signal: AbortSignal) =>
	new Promise<void>((resolve, reject) => {
		const aborted = () => {
			clearTimeout(timer);
			reject(new DOMException('Cancelled', 'AbortError'));
		};
		const timer = setTimeout(() => {
			signal.removeEventListener('abort', aborted);
			resolve();
		}, milliseconds);
		if (signal.aborted) aborted();
		else signal.addEventListener('abort', aborted, { once: true });
	});
/** ADR 012 capability stays within this stack frame and is used only at the approved upload URL. */
export async function uploadVideo(
	api: CraftskyAPI,
	owner: string,
	media: DraftMedia,
	signal: AbortSignal,
	progress: (message: string) => void,
	fetcher: typeof fetch = fetch,
	bypassDeduplication = false
): Promise<NonNullable<PostPayload['embed']>> {
	if (signal.aborted) throw new DOMException('Cancelled', 'AbortError');
	if (!media.blob) throw new Error('The local video file is missing.');
	const limits = await api.media.videoLimits();
	if (!limits.canUpload)
		throw new Error(
			'Video upload is unavailable for this account. Verify your PDS email or check your upload allowance.'
		);
	const authorization = await api.media.videoAuthorization();
	if (signal.aborted) throw new DOMException('Cancelled', 'AbortError');
	if (
		typeof authorization.token !== 'string' ||
		typeof authorization.expiresAt !== 'string' ||
		Date.parse(authorization.expiresAt) <= Date.now()
	)
		throw new Error('Could not authorize this video upload.');
	const endpoint = new URL('/xrpc/app.bsky.video.uploadVideo', videoOrigin);
	endpoint.searchParams.set('did', owner);
	endpoint.searchParams.set('name', `craftsky-${crypto.randomUUID()}.mp4`);
	progress('Uploading video…');
	let source = media.blob;
	if (bypassDeduplication) {
		if (source.size > 300_000_000 - 16)
			throw new Error(
				'This video is too large to restore. Choose a smaller file.'
			);
		const box = crypto.getRandomValues(new Uint8Array(16));
		box.set([0, 0, 0, 16, 102, 114, 101, 101]);
		source = new Blob([source, box], { type: 'video/mp4' });
	}
	const response = await fetcher(endpoint, {
		method: 'POST',
		body: source,
		headers: {
			'Content-Type': 'video/mp4',
			Authorization: `Bearer ${authorization.token}`
		},
		credentials: 'omit',
		redirect: 'error',
		referrerPolicy: 'no-referrer',
		cache: 'no-store',
		signal
	});
	if (!response.ok && response.status !== 409)
		throw new Error('The video service could not accept this upload.');
	let data = await boundedJSON(response);
	let status = (data.jobStatus || data) as Record<string, unknown>;
	const started = Date.now();
	let jobId = '';
	for (;;) {
		if (typeof status.jobId === 'string') jobId = status.jobId;
		const blob = blobValue(status.blob);
		if (
			blob &&
			jobId &&
			(status.state === 'JOB_STATE_COMPLETED' ||
				status.error === 'already_exists')
		)
			return {
				video: {
					jobId,
					blob,
					alt: media.alt,
					...(media.width && media.height
						? { aspectRatio: { width: media.width, height: media.height } }
						: {})
				}
			};
		if (
			status.state === 'JOB_STATE_FAILED' ||
			(status.error && status.error !== 'already_exists')
		)
			throw new Error(
				'Video processing failed. Check your account allowance and choose a supported MP4 video.'
			);
		if (!jobId || Date.now() - started > 15 * 60 * 1000)
			throw new Error(
				'Video processing is taking too long. Keep your draft and try again.'
			);
		progress(
			`Processing video${typeof status.progress === 'number' ? ` · ${Math.min(100, Math.max(0, status.progress))}%` : ''}…`
		);
		await delay(2000, signal);
		const query = new URL('/xrpc/app.bsky.video.getJobStatus', videoOrigin);
		query.searchParams.set('jobId', jobId);
		const poll = await fetcher(query, {
			credentials: 'omit',
			redirect: 'error',
			referrerPolicy: 'no-referrer',
			cache: 'no-store',
			signal
		});
		if (!poll.ok)
			throw new Error(
				'Could not check video processing. Your draft is still available.'
			);
		data = await boundedJSON(poll);
		if (!data.jobStatus || typeof data.jobStatus !== 'object')
			throw new Error('The video service returned an invalid job.');
		status = data.jobStatus as Record<string, unknown>;
	}
}
