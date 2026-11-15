import { getContext } from 'svelte';
import type { SubscriptionAccess } from '#lib/api/types.ts';
import { usePrompt } from '../social/prompt';
import { comingSoon, subscriptionsEnabled } from './release-config';

/** Discoverable paid actions share guarded, focus-restoring paywall entry. */
export function useSubscriptionFeature() {
	const access =
		getContext<() => SubscriptionAccess | undefined>('craftsky-access');
	const ask = usePrompt();
	const present = getContext<(tier: 'plus' | 'business') => void>(
		'craftsky-feature-paywall'
	);
	function allowed(tier: 'plus' | 'business' = 'plus') {
		const current = access?.();
		return (
			subscriptionsEnabled() &&
			!!current?.givesAccess &&
			(current.effectiveTier === 'business' ||
				(tier === 'plus' && current.effectiveTier === 'plus'))
		);
	}
	return {
		allowed,
		async require(feature: string, tier: 'plus' | 'business' = 'plus') {
			if (allowed(tier)) return true;
			if (!subscriptionsEnabled()) {
				comingSoon(feature);
				return false;
			}
			const current = access?.();
			if (!current) {
				await ask({
					title: 'Subscription access unavailable',
					detail: `Subscription access is unavailable. Try again before using ${feature}.`,
					confirmLabel: 'OK'
				});
				return false;
			}
			present(tier);
			return false;
		}
	};
}
