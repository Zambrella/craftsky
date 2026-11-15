import { getContext } from 'svelte';
import { goto } from '$app/navigation';
import type { SubscriptionAccess } from '#lib/api/types.ts';
import { usePrompt } from '../social/prompt';
import { comingSoon, subscriptionsEnabled } from './release-config';

/** Discoverable paid actions use the same guarded, focus-restoring prompt. */
export function useSubscriptionFeature() {
	const access =
		getContext<() => SubscriptionAccess | undefined>('craftsky-access');
	const ask = usePrompt();
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
			const accepted = await ask({
				title: `${feature} requires ${tier === 'business' ? 'Business' : 'Plus'}`,
				detail:
					tier === 'business'
						? `Subscribe to Business to use ${feature}.`
						: `Subscribe to Plus or Business to use ${feature}.`,
				confirmLabel: 'Learn about subscriptions'
			});
			if (accepted) await goto('/settings/subscriptions');
			return false;
		}
	};
}
