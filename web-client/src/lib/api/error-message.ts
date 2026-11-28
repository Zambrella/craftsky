import { APIError, MutationOutcomeUnknownError } from './transport';

/** Client-owned presentation matching Flutter's AppErrorMapper and English copy. */
export function apiErrorMessage(error: APIError): string {
	if (error instanceof MutationOutcomeUnknownError)
		return 'The action is still being confirmed. Retry the same action.';
	if (error.code === 'invalid_response')
		return 'Something went wrong. Please try again.';
	if (error.code === 'web_checkout_unavailable')
		return 'Web subscriptions are not available right now. Please try again later.';
	if (error.code === 'subscription_reconciliation_required')
		return 'Refresh your subscription status before opening checkout.';
	if (error.code === 'subscription_already_exists')
		return 'This account already has a subscription or a payment awaiting resolution. Refresh your subscription status.';
	if (error.code === 'web_checkout_pending')
		return 'This account already has a checkout in progress. Continue with that plan or cancel the pending purchase before choosing another.';
	if (error.code === 'web_management_unavailable')
		return 'Subscription management is temporarily unavailable. Please try again.';
	if (error.status === 0)
		return "You're offline. Check your connection and try again.";
	if (error.status === 401) return 'Please sign in again.';
	if (error.code === 'instagram_verification_required')
		return 'Verify or reactivate your Instagram account in Settings before importing.';
	if (error.code === 'instagram_post_already_imported')
		return 'This Instagram post has already been imported. Check its status in Instagram settings.';
	if (error.status === 403) return "You don't have permission to do that.";
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
