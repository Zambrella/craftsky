import {
	safeEvent,
	failureContext,
	type DiagnosticOperation
} from './privacy.ts';
import type { StackFrame } from '@sentry/browser';

function frames(error: Error): StackFrame[] {
	if (typeof error.stack !== 'string') return [];
	return error.stack.split('\n').flatMap((line) => {
		// Select coordinates only. Function names, exception prose and non-code
		// URLs are never emitted; safeEvent applies the shared filename allowlist.
		const match =
			/(?:^|\s|@|\()((?:https?:\/\/|file:\/\/|\/)[^\s)]+):(\d+):(\d+)\)?$/.exec(
				line.trim()
			);
		return match
			? [
					{
						filename: match[1],
						lineno: Number(match[2]),
						colno: Number(match[3]),
						in_app: true
					}
				]
			: [];
	});
}

function selected(
	error: unknown,
	fallback?: { filename: string; lineno: number; colno: number }
) {
	const values = [],
		seen = new Set<unknown>();
	let cause = error;
	while (cause instanceof Error && !seen.has(cause) && values.length < 6) {
		seen.add(cause);
		values.push({ type: cause.name, stacktrace: { frames: frames(cause) } });
		cause = cause.cause;
	}
	if (!values.length)
		values.push({ type: 'UnhandledRejection', stacktrace: { frames: [] } });
	if (!values[0].stacktrace.frames.length && fallback)
		values[0].stacktrace.frames.push({ ...fallback, in_app: true });
	const event = safeEvent({ exception: { values } });
	return {
		message: 'Unexpected browser error',
		exceptions:
			event?.exception?.values?.map((value) => ({
				kind: value.type,
				frames:
					value.stacktrace?.frames?.filter(
						(frame) => frame.filename !== '[redacted]'
					) || []
			})) || []
	};
}

/** Local protection is independent of optional SDK loading and consent. */
export function logFailure(error: unknown, operation?: DiagnosticOperation) {
	try {
		console.error(
			JSON.stringify({
				...selected(error),
				...failureContext(error, operation)
			})
		);
	} catch {
		/* Diagnostics cannot change application behavior. */
	}
}

export function installLocalDiagnostics(target: Window = window): () => void {
	const log = (
		error: unknown,
		fallback?: { filename: string; lineno: number; colno: number }
	) => {
		try {
			console.error(JSON.stringify(selected(error, fallback)));
		} catch {
			// Selection, serialization and the console itself must not affect the app.
		}
	};
	const onError = (event: ErrorEvent) => {
		event.preventDefault();
		log(event.error, {
			filename: event.filename,
			lineno: event.lineno,
			colno: event.colno
		});
	};
	const onRejection = (event: PromiseRejectionEvent) => {
		event.preventDefault();
		log(event.reason);
	};
	target.addEventListener('error', onError);
	target.addEventListener('unhandledrejection', onRejection);
	return () => {
		target.removeEventListener('error', onError);
		target.removeEventListener('unhandledrejection', onRejection);
	};
}
