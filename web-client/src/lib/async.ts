/** Stop waiting even when a dependency ignores AbortSignal; still observe its rejection. */
export function abortable<T>(
	work: Promise<T>,
	signal: AbortSignal
): Promise<T> {
	return new Promise((resolve, reject) => {
		const abort = () =>
			reject(signal.reason ?? new DOMException('Cancelled', 'AbortError'));
		if (signal.aborted) {
			void work.catch(() => {});
			abort();
			return;
		}
		signal.addEventListener('abort', abort, { once: true });
		work
			.then(resolve, reject)
			.finally(() => signal.removeEventListener('abort', abort));
	});
}
export async function withDeadline<T>(
	milliseconds: number,
	parent: AbortSignal | undefined,
	action: (signal: AbortSignal) => Promise<T>
): Promise<T> {
	const controller = new AbortController();
	const cancel = () => controller.abort(parent?.reason);
	if (parent?.aborted) cancel();
	else parent?.addEventListener('abort', cancel, { once: true });
	const timer = setTimeout(
		() =>
			controller.abort(new DOMException('Request timed out', 'TimeoutError')),
		milliseconds
	);
	try {
		controller.signal.throwIfAborted();
		return await abortable(action(controller.signal), controller.signal);
	} finally {
		clearTimeout(timer);
		parent?.removeEventListener('abort', cancel);
	}
}
