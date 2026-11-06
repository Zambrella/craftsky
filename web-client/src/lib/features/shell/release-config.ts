import { page } from '$app/state';

/** Same opt-in release switch as Flutter's SUBSCRIPTIONS_ENABLED. */
export const subscriptionsEnabled = () =>
	page.data.subscriptionsEnabled === true;

export function comingSoon(feature: string) {
	window.dispatchEvent(
		new CustomEvent('craftsky-coming-soon', { detail: feature })
	);
}
