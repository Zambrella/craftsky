import { describe, expect, it } from 'vitest';
import { NotificationBadge } from './notification-badge.svelte.ts';
function badge() {
	const badge = new NotificationBadge();
	badge.accept(3, badge.capture());
	return badge;
}
describe('optimistic notification badge', () => {
	it('clears immediately and rejects stale polls before and during acknowledgement', () => {
		const state = badge(),
			oldPoll = state.capture();
		const finish = state.beginSeen(),
			pendingPoll = state.capture();
		expect(state.count).toBe(0);
		state.accept(3, oldPoll);
		state.accept(3, pendingPoll);
		expect(state.count).toBe(0);
		finish(true);
		state.accept(3, pendingPoll);
		expect(state.count).toBe(0);
		state.accept(1, state.capture());
		expect(state.count).toBe(1);
	});
	it('restores the count if viewing or acknowledgement fails', () => {
		const state = badge();
		state.beginSeen()(false);
		expect(state.count).toBe(3);
	});
	it('keeps a successful acknowledgement when overlapping views settle out of order', () => {
		const state = badge(),
			first = state.beginSeen(),
			second = state.beginSeen();
		first(true);
		second(false);
		expect(state.count).toBe(0);
	});
	it('restores the original count when all overlapping views fail', () => {
		const state = badge(),
			first = state.beginSeen(),
			second = state.beginSeen();
		first(false);
		second(false);
		second(true);
		expect(state.count).toBe(3);
	});
	it('isolates a late previous-account acknowledgement', () => {
		const previous = badge(),
			current = badge(),
			finish = previous.beginSeen();
		finish(true);
		expect(previous.count).toBe(0);
		expect(current.count).toBe(3);
	});
});
