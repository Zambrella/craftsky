import { page } from '$app/state';
import { authenticatedContext } from './page-context';

/** Same opt-in release switch as Flutter's SUBSCRIPTIONS_ENABLED. */
export const subscriptionsEnabled = () =>
	authenticatedContext(page.data)?.subscriptionsEnabled === true;

export function comingSoon(feature: string) {
	window.dispatchEvent(
		new CustomEvent('craftsky-coming-soon', { detail: feature })
	);
}
