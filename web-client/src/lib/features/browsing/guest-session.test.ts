import { it, expect } from 'vitest';
import { GuestSession } from './guest-session.svelte';

// UT-003 / FR-004, FR-005 / AC-002.
it('shows the welcome prompt only until Explore within this guest tab session', () => {
	const values = new Map<string, string>();
	const storage = {
		getItem: (key: string) => values.get(key) ?? null,
		setItem: (key: string, value: string) => {
			values.set(key, value);
		}
	};
	const guest = new GuestSession(storage);
	guest.enter('/feed');
	expect(guest.promptOpen).toBe(true);
	guest.explore();
	expect(guest.promptOpen).toBe(false);
	guest.enter('/feed');
	expect(guest.promptOpen).toBe(false);
	const remount = new GuestSession(storage);
	remount.enter('/feed');
	expect(remount.promptOpen).toBe(false);
	remount.signIn();
	expect(remount.promptOpen).toBe(true);
	remount.explore();
	expect(remount.promptOpen).toBe(false);
	const newTab = new GuestSession();
	newTab.enter('/feed');
	expect(newTab.promptOpen).toBe(true);
});
it('keeps direct content links readable without an automatic prompt', () => {
	const guest = new GuestSession();
	guest.enter('/profiles/alice.test');
	expect(guest.promptOpen).toBe(false);
	guest.enter('/posts/did:plc:alice/first');
	expect(guest.promptOpen).toBe(false);
	guest.enter('/');
	expect(guest.promptOpen).toBe(true);
});
it('can explore when tab storage is unavailable', () => {
	const guest = new GuestSession({
		getItem: () => {
			throw Error('blocked');
		},
		setItem: () => {
			throw Error('blocked');
		}
	});
	guest.enter('/feed');
	guest.explore();
	guest.enter('/feed');
	expect(guest.promptOpen).toBe(false);
});
