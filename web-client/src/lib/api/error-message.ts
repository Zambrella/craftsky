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
		return 'This account already has a checkout in progress. Continue with that plan or wait for the checkout to expire before choosing another.';
	if (error.code === 'web_management_unavailable')
		return 'Subscription management is not available here yet. Use the management link in your Stripe receipt.';
	if (error.status === 0)
		return "You're offline. Check your connection and try again.";
	if (error.status === 401) return 'Please sign in again.';
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
