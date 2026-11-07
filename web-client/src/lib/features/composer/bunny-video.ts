import { UserFacingError, InputError } from '#lib/user-facing-error.ts';
import type { JsonObject } from '#lib/api/client.ts';
import type { DraftMedia } from '../drafts/store.ts';

const endpoint = 'https://video.bunnycdn.com/tusupload';
const chunkBytes = 8 * 1024 * 1024;
const uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;
class BunnyUploadProtocolError extends UserFacingError {}
const invalid = () =>
	new BunnyUploadProtocolError(
		'The video service returned an invalid upload response.'
	);

function offset(response: Response, maximum: number): number {
	const value = response.headers.get('Upload-Offset');
	if (!value || !/^\d+$/.test(value)) throw invalid();
	const parsed = Number(value);
	if (!Number.isSafeInteger(parsed) || parsed > maximum) throw invalid();
	return parsed;
}

/** Only the active upload owns these capabilities. Never persist or log them. */
export async function uploadBunnyVideo(
	media: DraftMedia,
	authorization: JsonObject,
	signal: AbortSignal,
	progress: (message: string) => void,
	fetcher: typeof fetch
): Promise<string> {
	const { jobId, videoId, libraryId, token, expiresAt, title, collectionId } =
		authorization;
	if (
		authorization.provider !== 'bunny' ||
		typeof jobId !== 'string' ||
		!uuid.test(jobId) ||
		typeof videoId !== 'string' ||
		!uuid.test(videoId) ||
		typeof collectionId !== 'string' ||
		!uuid.test(collectionId) ||
		typeof libraryId !== 'string' ||
		!/^[1-9]\d*$/.test(libraryId) ||
		typeof token !== 'string' ||
		!/^[0-9a-f]{64}$/.test(token) ||
		typeof expiresAt !== 'string' ||
		!Number.isFinite(Date.parse(expiresAt)) ||
		Date.parse(expiresAt) <= Date.now() ||
		typeof title !== 'string' ||
		!title.length ||
		title.length > 200
	)
		throw new UserFacingError('Could not authorize this video upload.');
	const source = media.blob;
	if (
		!source ||
		source.size <= 0 ||
		source.size > 300_000_000 ||
		!['video/mp4', 'video/quicktime'].includes(media.mime)
	)
		throw new InputError(
			'Choose an MP4 or MOV video up to 300 MB and ten minutes.'
		);
	const headers = {
		'Tus-Resumable': '1.0.0',
		AuthorizationSignature: token,
		AuthorizationExpire: String(Math.floor(Date.parse(expiresAt) / 1000)),
		LibraryId: libraryId,
		VideoId: videoId
	};
	const request = async (url: string, options: RequestInit) => {
		if (signal.aborted) throw new DOMException('Cancelled', 'AbortError');
		const response = await fetcher(url, {
			...options,
			signal,
			credentials: 'omit',
			redirect: 'error',
			referrerPolicy: 'no-referrer',
			cache: 'no-store'
		});
		if (response.redirected) throw invalid();
		return response;
	};
	// btoa cannot encode arbitrary Unicode titles directly.
	const encode = (value: string) =>
		btoa(
			Array.from(new TextEncoder().encode(value), (byte) =>
				String.fromCharCode(byte)
			).join('')
		);
	progress('Uploading video…');
	const created = await request(endpoint, {
		method: 'POST',
		headers: {
			...headers,
			'Upload-Length': String(source.size),
			'Upload-Metadata': `filetype ${encode(media.mime)},title ${encode(title)},collection ${encode(collectionId)}`
		}
	});
	if (created.status !== 201)
		throw new UserFacingError('The video service could not start this upload.');
	const location = created.headers.get('Location');
	if (!location) throw invalid();
	let target: URL;
	try {
		target = new URL(location, endpoint);
	} catch {
		throw invalid();
	}
	if (
		target.origin !== 'https://video.bunnycdn.com' ||
		target.username ||
		target.password ||
		!target.pathname.startsWith('/tusupload/') ||
		target.search ||
		target.hash
	)
		throw invalid();
	let sent = 0,
		recoveries = 0;
	while (sent < source.size) {
		const end = Math.min(source.size, sent + chunkBytes);
		let response: Response | undefined;
		let cause: unknown;
		try {
			response = await request(target.href, {
				method: 'PATCH',
				headers: {
					...headers,
					'Upload-Offset': String(sent),
					'Content-Type': 'application/offset+octet-stream'
				},
				body: source.slice(sent, end)
			});
		} catch (error) {
			if (signal.aborted) throw new DOMException('Cancelled', 'AbortError');
			if (error instanceof BunnyUploadProtocolError) throw error;
			cause = error;
		}
		if (response?.status === 204) {
			if (offset(response, source.size) !== end) throw invalid();
			sent = end;
		} else {
			if (
				response &&
				![408, 409, 423, 429].includes(response.status) &&
				response.status < 500
			)
				throw new UserFacingError(
					'The video service could not accept this upload.'
				);
			if (++recoveries > 3)
				throw new UserFacingError(
					'Could not finish uploading this video. Your draft is still available.',
					{ cause }
				);
			const status = await request(target.href, { method: 'HEAD', headers });
			if (![200, 204].includes(status.status)) throw invalid();
			const next = offset(status, source.size);
			if (next < sent) throw invalid();
			sent = next;
		}
		progress(`Uploading video · ${Math.floor((sent * 100) / source.size)}%…`);
	}
	return jobId;
}
