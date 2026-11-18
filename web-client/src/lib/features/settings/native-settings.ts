import currencyScales from './currencies.json';
import {
	backgrounds,
	colours,
	record,
	text,
	type GrowthPoint
} from './model.ts';
import type { BillingState } from '#lib/api/types.ts';

export { characterCount, requireRecordText } from './text-limits.ts';

/** Flutter's ProfileCustomisation.fromMap resolves unknown open values to defaults. */
export function effectiveCustomisation(value: unknown) {
	const custom = record(value),
		colour = text(custom.colour),
		background = text(custom.profileBackground) || text(custom.background);
	return {
		colour: Object.hasOwn(colours, colour) ? colour : 'cobalt',
		background: backgrounds.includes(background) ? background : 'none'
	};
}
const currencyNames = new Intl.DisplayNames(['en'], { type: 'currency' });
export const currencyOptions = Object.keys(currencyScales)
	.sort()
	.map((code) => ({
		code,
		label: `${code} - ${currencyNames.of(code) || code}`
	}));

export type TierState =
	| 'empty'
	| 'active'
	| 'cancelled'
	| 'dormant'
	| 'stale'
	| 'pending'
	| 'anomaly'
	| 'unavailable';
/** Keep native presentation precedence for unresolved provider state. */
export function tierState(
	state: BillingState,
	tier: 'plus' | 'business'
): TierState {
	const pairs = state.licenses
		.filter((license) => license.tier === tier)
		.map((license) => ({
			license,
			subscription: state.subscriptions.find(
				(value) => value.id === license.subscriptionId
			)
		}));
	if (!pairs.length) return 'empty';
	if (
		pairs.some(
			({ license, subscription }) =>
				!subscription ||
				(license.anomaly && license.anomaly !== 'none') ||
				(subscription.anomaly && subscription.anomaly !== 'none')
		)
	)
		return 'anomaly';
	if (
		state.reconciliationStale ||
		state.requestedGeneration > state.reconciledGeneration
	)
		return 'stale';
	if (pairs.some(({ subscription }) => subscription!.pendingPayment))
		return 'pending';
	if (
		pairs.some(
			({ subscription }) =>
				!['active', 'cancelled', 'expired'].includes(subscription!.status)
		)
	)
		return 'unavailable';
	if (
		pairs.some(
			({ subscription }) =>
				subscription!.status === 'cancelled' && subscription!.givesAccess
		)
	)
		return 'cancelled';
	return pairs.some(({ subscription }) => subscription!.givesAccess)
		? 'active'
		: 'dormant';
}
export const tierStateLabel: Record<TierState, string> = {
	empty: 'No current subscription',
	active: 'Active',
	cancelled: 'Canceled; access continues until the current period ends',
	dormant: 'Not currently providing access',
	stale: 'Subscription status needs refreshing',
	pending: 'Payment or reconciliation is pending',
	anomaly: 'Subscription state needs support',
	unavailable: 'Status unavailable'
};

/** Same padded integer axis, daily gaps and point markers as FollowerGrowthChart. */
export function followerChart(
	source: readonly GrowthPoint[],
	period = '30d',
	width = 600
) {
	let points = source;
	if (period === '1y') {
		const months = new Map<string, GrowthPoint>();
		for (const point of source) {
			const month = point.date.slice(0, 7) + '-01';
			const previous = months.get(month);
			months.set(month, {
				date: month,
				count: point.count ?? previous?.count ?? null
			});
		}
		points = [...months.values()];
	}
	const observed = points.filter(
		(point): point is GrowthPoint & { count: number } =>
			point.count !== null && Number.isFinite(point.count)
	);
	if (!observed.length) return undefined;
	const minimum = Math.min(...observed.map((point) => point.count)),
		maximum = Math.max(...observed.map((point) => point.count));
	const interval = Math.max(1, Math.ceil((maximum - minimum) / 4));
	const lower = Math.max(
			0,
			Math.floor((minimum - interval) / interval) * interval
		),
		upper = Math.ceil((maximum + interval) / interval) * interval;
	const x = (index: number) =>
		58 + (index / Math.max(1, points.length - 1)) * Math.max(40, width - 80);
	const y = (count: number) =>
		204 - ((count - lower) / Math.max(1, upper - lower)) * 184;
	const ticks = Array.from(
		{ length: Math.round((upper - lower) / interval) + 1 },
		(_, index) => ({
			count: lower + index * interval,
			y: y(lower + index * interval)
		})
	);
	const dots: { x: number; y: number }[] = [],
		paths: string[] = [];
	let current = '';
	points.forEach((point, index) => {
		if (point.count === null || !Number.isFinite(point.count)) {
			if (current) paths.push(current);
			current = '';
			return;
		}
		const position = { x: x(index), y: y(point.count) };
		dots.push(position);
		current += `${current ? ' L' : 'M'}${position.x.toFixed(2)},${position.y.toFixed(2)}`;
	});
	if (current) paths.push(current);
	const dates = [
		...new Set([0, Math.floor(points.length / 2), points.length - 1])
	].map((index) => ({ date: points[index].date, x: x(index) }));
	return { ticks, dots, paths, dates, lower, upper, interval };
}

const moderationLabels: Record<string, Record<string, string>> = {
	reason: {
		harassment: 'Harassment',
		hate: 'Hate or hateful conduct',
		spam: 'Spam',
		misleading: 'Misleading content',
		suspected_ai_generated: 'Suspected AI-generated content',
		adult_or_graphic: 'Adult or graphic content',
		impersonation: 'Impersonation',
		off_topic: 'Off-topic content',
		intellectual_property: 'Intellectual property',
		other: 'Other policy issue',
		illegal_content: 'Other illegal content',
		fraud_or_scam: 'Fraud or scam',
		self_harm: 'Self-harm policy violation',
		unknown: 'Policy decision'
	},
	effect: {
		formalWarning: 'Formal warning',
		visibilityWarn: 'Viewer warning',
		visibilityHide: 'Content hidden',
		visibilityTakedown: 'Content removed from CraftSky',
		strike: 'Account strike',
		severeSuspension: 'Severe suspension',
		unknown: 'Policy action'
	},
	action: {
		apply: 'Applied',
		expire: 'Expired',
		negate: 'Overturned',
		restore: 'Access restored',
		unknown: 'No longer active'
	},
	appeal: {
		pending: 'Appeal under review',
		upheld: 'Decision upheld after appeal',
		changed: 'Decision changed after appeal',
		none: 'An appeal can be requested by email.',
		unknown: 'Appeal status updated'
	}
};
/** Controlled native labels also give future enum values a safe, readable fallback. */
export function moderationLabel(
	kind: 'reason' | 'effect' | 'action' | 'appeal',
	value: unknown
) {
	const labels = moderationLabels[kind];
	return labels[typeof value === 'string' ? value : ''] || labels.unknown;
}
