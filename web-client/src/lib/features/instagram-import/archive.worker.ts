import { inspectInstagramArchive } from './engine';
import type {
	InstagramArchiveSource,
	InstagramPost,
	MediaSelection
} from './types';
let archive: InstagramArchiveSource | undefined;
let postsByKey = new Map<string, InstagramPost>();
const operations = new Map<number, AbortController>();
let queue = Promise.resolve();
type Request = {
	id: number;
	action: 'open' | 'media' | 'preview' | 'cancel';
	file?: File;
	sourceKey?: string;
	selection?: MediaSelection;
};
self.onmessage = ({ data }: MessageEvent<Request>) => {
	if (data.action === 'cancel') {
		operations.get(data.id)?.abort();
		return;
	}
	const controller = new AbortController();
	operations.set(data.id, controller);
	// Serialize archive reads so thumbnail requests cannot multiply extraction memory.
	queue = queue.then(async () => {
		try {
			controller.signal.throwIfAborted();
			let value: unknown;
			if (data.action === 'open') {
				archive?.close();
				archive = await inspectInstagramArchive(data.file!, controller.signal);
				postsByKey = new Map(
					archive.posts.map((post) => [post.sourceKey, post])
				);
				value = { posts: archive.posts, skipped: archive.skipped };
			} else {
				const post = data.sourceKey && postsByKey.get(data.sourceKey);
				if (!post || !archive) throw new Error('archiveClosed');
				value =
					data.action === 'media'
						? await archive.media(post, controller.signal, data.selection)
						: await archive.previewSource(post, controller.signal);
			}
			self.postMessage({ id: data.id, value });
		} catch (error) {
			const message = error instanceof Error ? error.message : '';
			const code = /^[a-zA-Z]{1,40}$/.test(message)
				? message
				: 'invalidArchive';
			self.postMessage({
				id: data.id,
				error: controller.signal.aborted ? 'cancelled' : code
			});
		} finally {
			operations.delete(data.id);
		}
	});
};
