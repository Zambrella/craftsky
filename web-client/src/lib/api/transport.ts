import { withDeadline } from '#lib/async.ts';
import type { Page, PageQuery, QueryValue } from './types';

export type Fetcher = typeof fetch;
export interface TransportOptions {
	headers?: HeadersInit | (() => HeadersInit);
}
export interface RequestOptions {
	query?: Record<string, QueryValue>;
	body?: unknown;
	headers?: HeadersInit;
	signal?: AbortSignal;
	operationKey?: string;
	expectedCid?: string;
	response?: 'json' | 'empty' | 'text' | 'bytes';
	validate?: (value: Record<string, unknown>) => boolean;
}

/** Carries only the public API error envelope; request bodies are never retained. */
export class APIError extends Error {
	readonly name: string = 'APIError';
	httpMethod?: string;
	responseKind?: 'appViewError' | 'json' | 'nonJson' | 'empty';
	constructor(
		readonly status: number,
		readonly code: string,
		message: string,
		readonly requestId?: string,
		readonly fields?: Record<string, string>,
		readonly retryAfter?: number,
		options?: ErrorOptions
	) {
		super(message, options);
	}
	get retryable(): boolean {
		return this.status === 0 || this.status === 429 || this.status >= 500;
	}
}

/** The write may have reached the PDS. Retry the original frozen request and key. */
export class MutationOutcomeUnknownError extends APIError {
	readonly name = 'MutationOutcomeUnknownError';
	constructor(
		readonly operationKey: string,
		retryAfter = 1,
		readonly responsePending = false,
		options?: ErrorOptions
	) {
		super(
			202,
			'mutation_outcome_unknown',
			'The action is still being confirmed. Retry the same action.',
			undefined,
			undefined,
			retryAfter,
			options
		);
	}
	override get retryable(): boolean {
		return true;
	}
}
export function newOperationKey(): string {
	return crypto.randomUUID();
}
export function requireOperationKey(key: string): string {
	if (
		!/^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(
			key
		)
	)
		throw new TypeError('A canonical lowercase operation UUID is required');
	return key;
}
export function queryString(values: Record<string, QueryValue> = {}): string {
	const query = new URLSearchParams();
	for (const [key, value] of Object.entries(values)) {
		if (value === undefined || value === null) continue;
		if (Array.isArray(value)) {
			for (const item of value) query.append(key, item);
		} else query.set(key, String(value));
	}
	return query.size ? `?${query}` : '';
}
export const segment = (value: string): string => encodeURIComponent(value);
export const profileSegment = (value: string): string =>
	segment(value === 'me' ? 'me' : `@${value.replace(/^@/, '')}`);
export const pageQuery = ({ cursor, limit }: PageQuery = {}): Record<
	string,
	QueryValue
> => ({ cursor, limit });
const object = (value: unknown): value is Record<string, unknown> =>
	!!value && typeof value === 'object' && !Array.isArray(value);
const retrySeconds = (value: string | null): number | undefined =>
	value && /^\d+$/.test(value) ? Number(value) : undefined;

/** The caller supplies account-bound authenticated transport. This class never owns credentials. */
export class APITransport {
	readonly basePath: string;
	constructor(
		private readonly fetcher: Fetcher,
		basePath = '/api/appview',
		private readonly options: TransportOptions = {}
	) {
		if (
			!basePath.startsWith('/') ||
			basePath.startsWith('//') ||
			/[?#]/.test(basePath)
		)
			throw new TypeError('API basePath must be a same-origin path');
		this.basePath = basePath.replace(/\/$/, '');
	}
	async request<T>(
		method: string,
		path: string,
		options: RequestOptions = {}
	): Promise<T> {
		try {
			return await withDeadline(120000, options.signal, (signal) =>
				this.performRequest<T>(method, path, { ...options, signal })
			);
		} catch (error) {
			const failure =
				options.operationKey &&
				error instanceof DOMException &&
				['TimeoutError', 'AbortError'].includes(error.name)
					? new MutationOutcomeUnknownError(
							requireOperationKey(options.operationKey),
							1,
							false,
							{ cause: error }
						)
					: error;
			if (failure instanceof APIError) failure.httpMethod = method;
			throw failure;
		}
	}
	private async performRequest<T>(
		method: string,
		path: string,
		options: RequestOptions
	): Promise<T> {
		const key =
			options.operationKey === undefined
				? undefined
				: requireOperationKey(options.operationKey);
		const supplied =
			typeof this.options.headers === 'function'
				? this.options.headers()
				: this.options.headers;
		const headers = new Headers(supplied);
		headers.set('Accept', 'application/json');
		for (const [name, value] of new Headers(options.headers))
			headers.set(name, value);
		if (key) headers.set('Idempotency-Key', key);
		if (options.expectedCid !== undefined)
			headers.set('If-Match', options.expectedCid);
		let body: BodyInit | undefined;
		if (options.body !== undefined) {
			if (
				options.body instanceof Blob ||
				options.body instanceof ArrayBuffer ||
				options.body instanceof Uint8Array ||
				options.body instanceof FormData
			) {
				body = options.body as BodyInit;
			} else {
				headers.set('Content-Type', 'application/json');
				body = JSON.stringify(options.body);
			}
		}
		let response: Response;
		try {
			response = await this.fetcher(
				`${this.basePath}${path}${queryString(options.query)}`,
				{
					method,
					headers,
					body,
					signal: options.signal,
					credentials: 'same-origin',
					redirect: 'manual',
					cache: 'no-store'
				}
			);
		} catch (error) {
			if (key)
				throw new MutationOutcomeUnknownError(key, 1, false, { cause: error });
			if (error instanceof DOMException && error.name === 'AbortError')
				throw error;
			throw new APIError(
				0,
				'network_unavailable',
				'Could not connect to CraftSky. Please try again.',
				undefined,
				undefined,
				undefined,
				{ cause: error }
			);
		}
		const retryAfter = retrySeconds(response.headers.get('Retry-After'));
		const rawRequestId = response.headers.get('X-Request-ID');
		const headerRequestId =
			rawRequestId &&
			/^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$/i.test(
				rawRequestId
			)
				? rawRequestId
				: undefined;
		let value: unknown;
		let responseKind: APIError['responseKind'];
		try {
			if (options.response === 'bytes' && response.ok)
				return (await response.arrayBuffer()) as T;
			if (options.response === 'text' && response.ok)
				return (await response.text()) as T;
			let text: string;
			try {
				text = await response.text();
				responseKind = text ? 'nonJson' : 'empty';
			} catch (error) {
				if (key)
					throw new MutationOutcomeUnknownError(key, 1, false, {
						cause: error
					});
				if (error instanceof DOMException && error.name === 'AbortError')
					throw error;
				throw new APIError(
					0,
					'network_unavailable',
					'Could not read the CraftSky response. Please try again.',
					headerRequestId,
					undefined,
					undefined,
					{ cause: error }
				);
			}
			if (text) {
				responseKind = 'nonJson';
				try {
					value = JSON.parse(text);
					responseKind =
						object(value) && typeof value.error === 'string'
							? 'appViewError'
							: 'json';
				} catch (cause) {
					if (key)
						throw new MutationOutcomeUnknownError(key, 1, false, { cause });
					throw new APIError(
						response.status,
						'invalid_response',
						'CraftSky returned an unreadable response.',
						headerRequestId,
						undefined,
						undefined,
						{ cause }
					);
				}
			}
			if (response.status === 202 && key) {
				const pending =
					object(value) &&
					Object.keys(value).length === 1 &&
					value.status === 'ambiguous' &&
					retryAfter !== undefined;
				throw new MutationOutcomeUnknownError(
					key,
					Math.max(1, Math.min(retryAfter ?? 1, 5)),
					pending
				);
			}
			if (!response.ok) {
				if (
					key &&
					(response.status >= 500 ||
						(response.status === 429 &&
							object(value) &&
							value.error === 'pds_rate_limited'))
				) {
					if (!object(value) || value.error !== 'video_blob_missing')
						throw new MutationOutcomeUnknownError(key, retryAfter ?? 1);
				}
				const envelope = object(value) ? value : {};
				const fields =
					object(envelope.fields) &&
					Object.values(envelope.fields).every((v) => typeof v === 'string')
						? (envelope.fields as Record<string, string>)
						: undefined;
				throw new APIError(
					response.status,
					typeof envelope.error === 'string'
						? envelope.error
						: 'request_failed',
					typeof envelope.message === 'string'
						? envelope.message
						: 'The request could not be completed.',
					typeof envelope.requestId === 'string' &&
						/^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$/i.test(
							envelope.requestId
						)
						? envelope.requestId
						: headerRequestId,
					fields,
					retryAfter
				);
			}
			if (options.response === 'empty') {
				if (
					(key && response.status !== 204) ||
					(text && response.status !== 202)
				) {
					if (key) throw new MutationOutcomeUnknownError(key);
					throw new APIError(
						response.status,
						'invalid_response',
						'Unexpected response to the action.',
						headerRequestId
					);
				}
				return undefined as T;
			}
			if (
				value === undefined ||
				!object(value) ||
				(options.validate && !options.validate(value))
			) {
				if (key) throw new MutationOutcomeUnknownError(key);
				throw new APIError(
					response.status,
					'invalid_response',
					'CraftSky returned an incomplete response.',
					headerRequestId
				);
			}
			return value as T;
		} catch (error) {
			if (error instanceof APIError) error.responseKind = responseKind;
			throw error;
		}
	}
	get<T>(
		path: string,
		query?: Record<string, QueryValue>,
		signal?: AbortSignal
	): Promise<T> {
		return this.request('GET', path, { query, signal });
	}
	async page<T>(
		path: string,
		query: Record<string, QueryValue> = {},
		signal?: AbortSignal
	): Promise<Page<T>> {
		const result = await this.get<Page<T>>(path, query, signal);
		if (
			!Array.isArray(result.items) ||
			(result.cursor !== undefined &&
				result.cursor !== null &&
				typeof result.cursor !== 'string')
		)
			throw new APIError(
				200,
				'invalid_response',
				'CraftSky returned an incomplete page.'
			);
		return result;
	}
}
