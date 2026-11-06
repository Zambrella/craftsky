import { InputError } from '#lib/user-facing-error.ts';
import { APIError } from '#lib/api/transport.ts';
import type { ErrorEvent, Event, EventHint, StackFrame } from '@sentry/browser';
import type { Envelope, EventEnvelope, Transport } from '@sentry/core';

const nativeTypes = new Set([
	'Error',
	'TypeError',
	'RangeError',
	'ReferenceError',
	'SyntaxError',
	'URIError',
	'EvalError',
	'AggregateError',
	'UnhandledRejection',
	'APIError',
	'MutationOutcomeUnknownError',
	'QuotaExceededError',
	'UserFacingError',
	'AbortError',
	'TimeoutError',
	'NotAllowedError'
]);
const eventID = (value: unknown): value is string =>
	typeof value === 'string' && /^[a-f0-9]{32}$/.test(value);
const coordinate = (value: unknown): value is number =>
	typeof value === 'number' &&
	Number.isSafeInteger(value) &&
	value >= 0 &&
	value < 2147483647;

function filename(value: unknown): string {
	if (typeof value !== 'string') return '[redacted]';
	try {
		const url = new URL(value, 'https://app.craftsky.social'),
			path = url.pathname;
		// Only fixed application code paths are retained. Route/resource URLs, query strings,
		// file URLs, local home directories and external dependencies are never sent.
		if (value.startsWith('file:')) return '[redacted]';
		if (
			url.origin !== 'https://app.craftsky.social' &&
			(typeof location === 'undefined' || url.origin !== location.origin) &&
			!['localhost', '127.0.0.1'].includes(url.hostname)
		)
			return '[redacted]';
		if (
			/^\/_app\/immutable\/(?:chunks|entry|nodes|workers)\/[A-Za-z0-9._-]+\.js$/.test(
				path
			)
		)
			return path;
		if (
			/^\/src\/(?:lib|routes)\/[A-Za-z0-9_./[\]-]+\.(?:ts|js|svelte)$/.test(
				path
			)
		)
			return path;
	} catch {}
	return '[redacted]';
}
function frame(value: StackFrame): StackFrame {
	return {
		filename: filename(value.filename),
		...(coordinate(value.lineno) ? { lineno: value.lineno } : {}),
		...(coordinate(value.colno) ? { colno: value.colno } : {}),
		...(typeof value.in_app === 'boolean' ? { in_app: value.in_app } : {})
	};
}
export function expectedFailure(error: unknown): boolean {
	if (error instanceof InputError) return true;
	if (error instanceof APIError)
		return error.status < 500 && error.code !== 'invalid_response';
	if (
		typeof DOMException !== 'undefined' &&
		error instanceof DOMException &&
		['AbortError', 'NotAllowedError'].includes(error.name)
	)
		return true;
	if (
		error instanceof TypeError &&
		[
			'Failed to fetch',
			'Load failed',
			'NetworkError when attempting to fetch resource.'
		].includes(error.message)
	)
		return true;
	return false;
}

const operationNames = [
	'post.unsave',
	'drafts.manage',
	'push.open',
	'messages.draft.load',
	'messages.draft.save',
	'messages.report.save',
	'messages.send.prepare',
	'composer.media.edit',
	'messages.authorize',
	'post.chooseFolder',
	'post.createFolder',
	'post.like',
	'post.relationship',
	'post.remove',
	'post.report',
	'post.repost',
	'post.revealPost',
	'post.revealQuote',
	'post.save',
	'post.toggle',
	'post.togglePin',
	'composer',
	'composer.addAttachments',
	'composer.addImages',
	'composer.addVideo',
	'composer.discardDraft',
	'composer.initialize',
	'composer.perform',
	'composer.requestClose',
	'composer.retry',
	'composer.saveDraft',
	'composer.submit',
	'discovery.crafts',
	'discovery.members',
	'framework',
	'messages.checkSend',
	'messages.more',
	'messages.navigate',
	'messages.presented',
	'messages.search',
	'messages.send',
	'messages.setPolicy',
	'messages.start',
	'messages.sync',
	'page.import',
	'push.toggle',
	'settings.Account.action',
	'settings.Account.cancelDeletion',
	'settings.Account.confirmDeletion',
	'settings.Account.load',
	'settings.Account.more',
	'settings.Account.startDeletion',
	'settings.Billing.action',
	'settings.Billing.assign',
	'settings.Billing.load',
	'settings.Billing.refresh',
	'settings.Billing.unassign',
	'settings.Business.action',
	'settings.Business.deleteEvent',
	'settings.Business.deleteProfile',
	'settings.Business.load',
	'settings.Business.more',
	'settings.Business.saveEvent',
	'settings.Business.saveProfile',
	'settings.Business.useImage',
	'settings.Connections.action',
	'settings.Connections.load',
	'settings.Connections.more',
	'settings.Connections.remove',
	'settings.Instagram.accept',
	'settings.Instagram.action',
	'settings.Instagram.importFollowing',
	'settings.Instagram.load',
	'settings.Instagram.moreImports',
	'settings.Instagram.moreSuggestions',
	'settings.Instagram.readFile',
	'settings.Instagram.setDiscoverable',
	'settings.Instagram.update',
	'settings.Instagram.verify',
	'settings.Preferences.action',
	'settings.Preferences.changeIncoming',
	'settings.Preferences.changePush',
	'settings.Preferences.load',
	'settings.Preferences.notification',
	'settings.Preferences.replaceLanguages',
	'settings.Preferences.saveCustomisation',
	'settings.Preferences.setMessaging',
	'settings.SettingsWorkspace.action',
	'settings.SettingsWorkspace.loadMenu',
	'settings.SettingsWorkspace.signOut',
	'settings.load',
	'settings.save',
	'social',
	'social.Activity.action',
	'social.Activity.follow',
	'social.Activity.load',
	'social.Event.action',
	'social.Event.load',
	'social.Feed.action',
	'social.Feed.load',
	'social.MutualFollowers.action',
	'social.MutualFollowers.load',
	'social.Onboarding.action',
	'social.Onboarding.finish',
	'social.Profile.action',
	'social.Profile.business.profileEvents',
	'social.Profile.follow',
	'social.Profile.more',
	'social.Profile.profiles.get',
	'social.Profile.profiles.posts',
	'social.Profile.relationship',
	'social.Profile.report',
	'social.ProfileEditor.action',
	'social.ProfileEditor.profiles.me',
	'social.ProfileEditor.save',
	'social.Saved.action',
	'social.Saved.create',
	'social.Saved.load',
	'social.Saved.remove',
	'social.Saved.rename',
	'social.Search.action',
	'social.Search.load',
	'social.Search.suggest',
	'social.Tag.action',
	'social.Tag.load',
	'social.Thread.action',
	'social.Thread.load',
	'social.Thread.replies',
	'startup',
	'workspace'
] as const;
const operations = new Set<string>(operationNames);
export type DiagnosticOperation = (typeof operationNames)[number];
const uuid = (value: unknown): value is string =>
	typeof value === 'string' &&
	/^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$/i.test(
		value
	);
export function failureContext(
	error: unknown,
	operation?: DiagnosticOperation
) {
	return {
		...(operation && operations.has(operation) ? { operation } : {}),
		...(error instanceof APIError
			? {
					httpStatus: error.status,
					...(error.responseKind ? { responseKind: error.responseKind } : {}),
					...(error.httpMethod &&
					/^(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)$/.test(error.httpMethod)
						? { httpMethod: error.httpMethod }
						: {}),
					...(/^[a-z][a-z0-9_]{0,63}$/.test(error.code)
						? { errorCode: error.code }
						: {}),
					...(uuid(error.requestId)
						? { appViewRequestId: error.requestId }
						: {})
				}
			: {})
	};
}
function selectedContext(value: Record<string, unknown> = {}) {
	return {
		...(typeof value.responseKind === 'string' &&
		['appViewError', 'json', 'nonJson', 'empty'].includes(value.responseKind)
			? { responseKind: value.responseKind }
			: {}),
		...(typeof value.httpMethod === 'string' &&
		/^(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)$/.test(value.httpMethod)
			? { httpMethod: value.httpMethod }
			: {}),
		...(typeof value.operation === 'string' && operations.has(value.operation)
			? { operation: value.operation }
			: {}),
		...(Number.isInteger(value.httpStatus) &&
		Number(value.httpStatus) >= 0 &&
		Number(value.httpStatus) <= 599
			? { httpStatus: value.httpStatus }
			: {}),
		...(typeof value.errorCode === 'string' &&
		/^[a-z][a-z0-9_]{0,63}$/.test(value.errorCode)
			? { errorCode: value.errorCode }
			: {}),
		...(uuid(value.appViewRequestId)
			? { appViewRequestId: value.appViewRequestId }
			: {})
	};
}

/** Select a closed set of structural diagnostics, never attempt to scrub private prose. */
export function safeEvent(
	event: Event,
	hint: EventHint = {},
	enabled = true
): ErrorEvent | null {
	if (
		!enabled ||
		event.type !== undefined ||
		expectedFailure(hint.originalException) ||
		!event.exception?.values?.length
	)
		return null;
	const values = event.exception.values.slice(-6).map((error) => ({
		type: nativeTypes.has(error.type || '') ? error.type : 'Error',
		value: 'Unexpected browser error',
		...(error.mechanism
			? {
					mechanism: {
						type: [
							'generic',
							'onerror',
							'onunhandledrejection',
							'chained'
						].includes(error.mechanism.type)
							? error.mechanism.type
							: 'generic',
						...(typeof error.mechanism.handled === 'boolean'
							? { handled: error.mechanism.handled }
							: {}),
						...(coordinate(error.mechanism.exception_id)
							? { exception_id: error.mechanism.exception_id }
							: {}),
						...(coordinate(error.mechanism.parent_id)
							? { parent_id: error.mechanism.parent_id }
							: {}),
						...(error.mechanism.source === 'cause' ? { source: 'cause' } : {})
					}
				}
			: {}),
		...(error.stacktrace?.frames?.length
			? {
					stacktrace: { frames: error.stacktrace.frames.slice(-100).map(frame) }
				}
			: {})
	}));
	const diagnostic = selectedContext({
		...event.contexts?.diagnostic,
		...failureContext(hint.originalException)
	});
	const images = event.debug_meta?.images?.slice(0, 100).flatMap((image) =>
		image.type === 'sourcemap' &&
		uuid(image.debug_id) &&
		filename(image.code_file) !== '[redacted]'
			? [
					{
						type: 'sourcemap' as const,
						code_file: filename(image.code_file),
						debug_id: image.debug_id
					}
				]
			: []
	);
	return {
		type: undefined,
		platform: 'javascript',
		...(event.sdk?.name === 'sentry.javascript.browser' &&
		/^\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)?$/.test(event.sdk.version || '')
			? { sdk: { name: event.sdk.name, version: event.sdk.version } }
			: {}),
		level: event.level === 'fatal' ? 'fatal' : 'error',
		...(typeof event.release === 'string' &&
		/^craftsky-web@[0-9A-Za-z.+-]{1,100}$/.test(event.release)
			? { release: event.release }
			: {}),
		...(['development', 'preview', 'production', 'test'].includes(
			event.environment || ''
		)
			? { environment: event.environment }
			: {}),
		...(Object.keys(diagnostic).length ? { contexts: { diagnostic } } : {}),
		...(images?.length ? { debug_meta: { images } } : {}),
		...(eventID(event.event_id) ? { event_id: event.event_id } : {}),
		...(typeof event.timestamp === 'number' && Number.isFinite(event.timestamp)
			? { timestamp: event.timestamp }
			: {}),
		exception: { values }
	};
}

/** Attachments/logs/traces and scope data added after beforeSend are blocked at the wire boundary. */
export function privateTransport(
	inner: Transport,
	allowed: () => boolean
): Transport {
	return {
		send(envelope: Envelope) {
			if (!allowed() || !eventID(envelope[0].event_id))
				return Promise.resolve({ statusCode: 200 });
			const events: EventEnvelope[1] = [];
			for (const [header, body] of envelope[1]) {
				if (
					header.type !== 'event' ||
					!body ||
					typeof body !== 'object' ||
					body instanceof Uint8Array
				)
					continue;
				const value = safeEvent(body as Event, {}, allowed());
				if (value) events.push([{ type: 'event' }, value]);
			}
			if (!events.length) return Promise.resolve({ statusCode: 200 });
			return inner.send([
				{ event_id: envelope[0].event_id, sent_at: new Date().toISOString() },
				events
			]);
		},
		flush: (timeout) => inner.flush(timeout)
	};
}
