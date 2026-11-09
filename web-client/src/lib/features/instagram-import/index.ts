import { InputError } from '#lib/user-facing-error.ts';
import type {
	InstagramArchive,
	InstagramMedia,
	InstagramPost,
	MediaSelection
} from './types';
export type {
	InstagramArchive,
	InstagramPost,
	InstagramMedia,
	MediaSelection
} from './types';
function userError(code: string) {
	if (code === 'cancelled')
		return new DOMException('Import cancelled.', 'AbortError');
	const messages: Record<string, string> = {
		postsMissing:
			'No Instagram post data was found. Export your posts in JSON format with media included, then select the downloaded ZIP.',
		unsupportedPosts:
			'This Instagram post export format is not supported. Choose a JSON export with post media included.',
		mediaMissing:
			'This media file is missing or ambiguous in the export. Choose different media or another export.',
		mediaFormat:
			'This media format is not supported. Choose JPEG, PNG, WebP, MP4 or MOV media.',
		mediaChoiceRequired:
			'Choose up to eight images or one video before opening the composer.',
		archiveClosed: 'Select your Instagram export again to continue.',
		imageDimensionLimit:
			'This image is too large to process safely. Choose a smaller image.',
		imagePixelLimit:
			'This image is too large to process safely. Choose a smaller image.',
		entryLimit:
			'This file exceeds the supported size or archive safety limits. Choose different media or export a smaller copy.',
		entryCountLimit:
			'This export contains too many files. Export only your Instagram posts and try again.',
		archiveEntryLimit:
			'This export contains too many files. Export only your Instagram posts and try again.',
		centralDirectoryLimit:
			'This export has too much directory data. Export only your Instagram posts and try again.',
		metadataLimit:
			'This export contains too much post data. Request a smaller Instagram export.',
		postLimit:
			'This export contains too many posts. Request a smaller Instagram export.',
		ambiguousArchive:
			'This export contains conflicting files. Select an unmodified Instagram export.',
		unsupportedArchive:
			'This archive uses unsupported compression or encryption. Select an unencrypted Instagram ZIP export.'
	};
	return new InputError(
		messages[code] ||
			'This Instagram archive is invalid or damaged. Select another JSON-format ZIP export with post media included.'
	);
}
export async function openInstagramArchive(
	file: File,
	signal?: AbortSignal
): Promise<InstagramArchive> {
	signal?.throwIfAborted();
	const worker = new Worker(new URL('./archive.worker.ts', import.meta.url), {
		type: 'module'
	});
	let next = 0;
	let closed = false;
	const pending = new Map<
		number,
		{
			resolve: (value: unknown) => void;
			reject: (error: unknown) => void;
			cleanup: () => void;
		}
	>();
	const close = () => {
		if (closed) return;
		closed = true;
		worker.terminate();
		signal?.removeEventListener('abort', close);
		for (const operation of pending.values()) {
			operation.cleanup();
			operation.reject(new DOMException('Import closed.', 'AbortError'));
		}
		pending.clear();
	};
	signal?.addEventListener('abort', close, { once: true });
	worker.onmessage = ({
		data
	}: {
		data: { id: number; value?: unknown; error?: string };
	}) => {
		const operation = pending.get(data.id);
		if (!operation) return;
		pending.delete(data.id);
		operation.cleanup();
		if (data.error) operation.reject(userError(data.error));
		else operation.resolve(data.value);
	};
	worker.onerror = (event) => {
		event.preventDefault();
		for (const operation of pending.values()) {
			operation.cleanup();
			operation.reject(userError('invalidArchive'));
		}
		pending.clear();
		close();
	};
	const request = <T>(
		action: 'open' | 'media' | 'preview',
		body: Record<string, unknown>,
		abort?: AbortSignal
	): Promise<T> => {
		if (closed) return Promise.reject(userError('archiveClosed'));
		abort?.throwIfAborted();
		const id = ++next;
		return new Promise<T>((resolve, reject) => {
			const cancel = () => {
				worker.postMessage({ id, action: 'cancel' });
				pending.delete(id);
				reject(new DOMException('Import cancelled.', 'AbortError'));
			};
			abort?.addEventListener('abort', cancel, { once: true });
			pending.set(id, {
				resolve: (value) => resolve(value as T),
				reject,
				cleanup: () => abort?.removeEventListener('abort', cancel)
			});
			worker.postMessage({ id, action, ...body });
		});
	};
	try {
		const manifest = await request<{ posts: InstagramPost[]; skipped: number }>(
			'open',
			{ file },
			signal
		);
		return {
			...manifest,
			media: (post, abort, selection?: MediaSelection) =>
				request<InstagramMedia>(
					'media',
					{ sourceKey: post.sourceKey, selection },
					abort
				),
			preview: (post, abort) =>
				request<Blob | undefined>(
					'preview',
					{ sourceKey: post.sourceKey },
					abort
				),
			close
		};
	} catch (error) {
		close();
		throw error;
	}
}
