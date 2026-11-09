import { APIError, MutationOutcomeUnknownError } from './transport';

/** Client-owned presentation matching Flutter's AppErrorMapper and English copy. */
export function apiErrorMessage(error: APIError): string {
	if (error instanceof MutationOutcomeUnknownError)
		return 'The action is still being confirmed. Retry the same action.';
	if (error.code === 'invalid_response')
		return 'Something went wrong. Please try again.';
	if (error.status === 0)
		return "You're offline. Check your connection and try again.";
	if (error.status === 401) return 'Please sign in again.';
	if (error.code === 'instagram_verification_required')
		return 'Verify or reactivate your Instagram account in Settings before importing posts.';
	if (error.code === 'instagram_post_already_imported')
		return 'This Instagram post has already been imported. Check its status in Instagram settings.';
	if (error.code === 'not_found' || error.status === 404)
		return 'That content is no longer available.';
	if (error.status >= 500)
		return 'CraftSky is having trouble right now. Please try again.';
	if (error.status === 429)
		return 'Too many requests. Please wait a moment and try again.';
	return "That didn't work. Please try again.";
}

export function isCanceled(error: unknown): boolean {
	return error instanceof DOMException && error.name === 'AbortError';
}
