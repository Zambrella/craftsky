/** Worker-only local logging. Never imports the browser SDK or consumes browser consent. */
const kinds = new Set([
	'Error',
	'TypeError',
	'RangeError',
	'SyntaxError',
	'URIError',
	'AbortError',
	'TimeoutError',
	'UpstreamError'
]);
export type ServerOperation =
	| 'worker.unhandled'
	| 'public.read'
	| 'auth.request'
	| 'auth.session'
	| 'auth.callback'
	| 'appview.proxy';
export type ServerStage =
	'configuration' | 'fetch' | 'response' | 'decode' | 'request';
export function logServerFailure(
	error: unknown,
	operation: ServerOperation,
	stage: ServerStage = 'request',
	details: { status?: number; code?: string; requestId?: string } = {}
) {
	try {
		const causes = [],
			seen = new Set<unknown>();
		let cause = error;
		while (cause instanceof Error && !seen.has(cause) && causes.length < 6) {
			seen.add(cause);
			// Select only deployed Worker coordinates; no exception prose, function names,
			// URLs, local paths, source lines, request/session objects or arbitrary fields.
			const frames = (cause.stack || '')
				.split('\n')
				.flatMap((line) => {
					const match =
						/(?:^|[\s/(])((?:worker|index|_worker)\.js):(\d+):(\d+)\)?$/.exec(
							line
						);
					return match
						? [
								{
									file: match[1],
									line: Number(match[2]),
									column: Number(match[3])
								}
							]
						: [];
				})
				.slice(0, 100);
			causes.push({
				kind: kinds.has(cause.name) ? cause.name : 'Error',
				frames
			});
			cause = cause.cause;
		}
		const status =
			Number.isInteger(details.status) &&
			details.status! >= 400 &&
			details.status! <= 599
				? details.status
				: undefined;
		const expected = status !== undefined && status < 500;
		(expected ? console.warn : console.error)(
			JSON.stringify({
				message: expected
					? 'Expected Worker failure'
					: 'Worker operation failed',
				operation,
				stage,
				causes,
				...(status ? { httpStatus: status } : {}),
				...(details.code && /^[a-z][a-z0-9_]{0,63}$/.test(details.code)
					? { errorCode: details.code }
					: {}),
				...(details.requestId &&
				/^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$/i.test(
					details.requestId
				)
					? { appViewRequestId: details.requestId }
					: {})
			})
		);
	} catch {
		/* Logging cannot alter HTTP behavior. */
	}
}
