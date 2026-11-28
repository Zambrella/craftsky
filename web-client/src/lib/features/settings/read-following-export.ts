import { abortable } from '#lib/async.ts';
import { InputError, UserFacingError } from '#lib/user-facing-error.ts';
import type { ParsedFollowing } from './model';

export async function readFollowingExport(
	file: File,
	signal: AbortSignal
): Promise<ParsedFollowing> {
	signal.throwIfAborted();
	const worker = new Worker(new URL('./following.worker.ts', import.meta.url), {
		type: 'module'
	});
	const unreadable = () =>
		new UserFacingError('Could not read this Instagram export.');
	try {
		return await abortable(
			new Promise<ParsedFollowing>((resolve, reject) => {
				worker.onerror = (event) => {
					event.preventDefault();
					reject(unreadable());
				};
				worker.onmessageerror = () => reject(unreadable());
				worker.onmessage = ({
					data
				}: MessageEvent<{ result?: ParsedFollowing; error?: string }>) => {
					if (data.result) resolve(data.result);
					else reject(data.error ? new InputError(data.error) : unreadable());
				};
				worker.postMessage(file);
			}),
			signal
		);
	} finally {
		worker.terminate();
	}
}
