import { getContext, onDestroy } from 'svelte';
import { page } from '$app/state';

type Entry = { y: number; state: Map<string, unknown>; loading: number };
const entry = (): Entry => ({ y: 0, state: new Map(), loading: 0 });
export const pageHistoryContext = Symbol('page-history');
export type PageLoad<T> = {
	isCurrent: () => boolean;
	save: (value: T) => void;
};
export type PageView<T> = {
	value: T | undefined;
	cachedTab: boolean;
	save: (value: T) => void;
	beginLoad: () => PageLoad<T>;
	onInvalidate: (refresh: () => void) => () => void;
	ready: () => void;
};

/** Account-scoped, in-memory state for history entries and adjacent list tabs. */
export class PageHistory {
	private entries = new Map<number, Entry>();
	private tabs = new Map<string, Entry>();
	private revisions = new Map<string, number>();
	private listeners = new Map<string, Set<() => void>>();
	private restoringTab: Entry | undefined;
	private tabNavigation = false;
	private current = entry();
	private pending: Entry | undefined;
	private popping = false;
	private cancelRestore = () => {};
	leave(index: number, target: number, pop: boolean, tabNavigation = false) {
		this.tabNavigation = tabNavigation;
		if (!tabNavigation) this.tabs.clear();
		this.cancelRestore();
		this.popping = pop;
		this.current.y = window.scrollY;
		this.entries.set(index, this.current);
		this.pending = pop ? this.entries.get(target) || entry() : entry();
		if (!pop)
			for (const key of this.entries.keys())
				if (key > index) this.entries.delete(key);
		// Keep navigation state bounded; never persist private page data to storage.
		while (this.entries.size > 50)
			this.entries.delete(this.entries.keys().next().value!);
	}
	arrive(index: number, pop: boolean) {
		this.current =
			(pop ? this.entries.get(index) : undefined) ||
			this.pending ||
			this.current;
		this.pending = undefined;
		this.popping = false;
		this.entries.set(index, this.current);
		if (pop || this.restoringTab === this.current) this.restore(this.current);
		this.restoringTab = undefined;
	}
	view<T>(key: string, tabKey?: string): PageView<T> {
		let revision = this.revisions.get(key) || 0;
		const owner =
			(this.popping && page.state.navigationIndex !== undefined
				? this.entries.get(page.state.navigationIndex)
				: undefined) ||
			this.pending ||
			this.current;
		// Tab links reuse loaded content; other navigations still fetch fresh data.
		if (tabKey !== undefined) {
			const cached = this.tabs.get(tabKey);
			if (
				this.tabNavigation &&
				!this.popping &&
				cached &&
				!owner.state.has(key)
			) {
				owner.state.set(key, cached.state.get(key));
				owner.y = cached.y;
				if (this.pending) this.restoringTab = owner;
				else this.restore(owner);
			}
			this.tabs.delete(tabKey);
			this.tabs.set(tabKey, owner);
			while (this.tabs.size > 20)
				this.tabs.delete(this.tabs.keys().next().value!);
		}
		owner.loading++;
		let finished = false;
		return {
			value: owner.state.get(key) as T | undefined,
			cachedTab: tabKey !== undefined && this.tabNavigation,
			beginLoad: () => {
				const requestRevision = this.revisions.get(key) || 0;
				const isCurrent = () =>
					(this.revisions.get(key) || 0) === requestRevision;
				return {
					isCurrent,
					save: (value: T) => {
						if (!isCurrent()) return;
						revision = requestRevision;
						owner.state.set(key, value);
					}
				};
			},
			onInvalidate: (refresh: () => void) => {
				const listeners = this.listeners.get(key) || new Set();
				this.listeners.set(key, listeners);
				listeners.add(refresh);
				return () => {
					listeners.delete(refresh);
					if (!listeners.size) this.listeners.delete(key);
				};
			},
			save: (value: T) => {
				if ((this.revisions.get(key) || 0) === revision)
					owner.state.set(key, value);
			},
			ready: () => {
				if (!finished) {
					owner.loading--;
					finished = true;
				}
			}
		};
	}
	/** Policy changes invalidate offscreen entries as well as adjacent tabs. */
	invalidate(key: string) {
		this.revisions.set(key, (this.revisions.get(key) || 0) + 1);
		const owners = new Set([
			this.current,
			...this.entries.values(),
			...this.tabs.values()
		]);
		if (this.pending) owners.add(this.pending);
		if (this.restoringTab) owners.add(this.restoringTab);
		for (const owner of owners) {
			if (owner.state.delete(key)) owner.y = 0;
		}
		for (const refresh of this.listeners.get(key) || []) refresh();
	}
	private restore(owner: Entry) {
		let frame = 0;
		const stop = () => {
			cancelAnimationFrame(frame);
			observer.disconnect();
			clearTimeout(timeout);
			for (const name of ['wheel', 'touchstart', 'pointerdown', 'keydown'])
				window.removeEventListener(name, stop);
		};
		const attempt = () => {
			cancelAnimationFrame(frame);
			frame = requestAnimationFrame(() => {
				if (
					owner.loading ||
					document.querySelector(
						'.app-main .social-loading, .app-main .skeleton'
					)
				)
					return;
				if (document.documentElement.scrollHeight - innerHeight < owner.y - 1)
					return;
				window.scrollTo({ top: owner.y, behavior: 'instant' });
				stop();
			});
		};
		const observer = new MutationObserver(attempt);
		observer.observe(document.querySelector('.app-main') || document.body, {
			childList: true,
			subtree: true,
			attributes: true
		});
		const timeout = setTimeout(stop, 15000);
		for (const name of ['wheel', 'touchstart', 'pointerdown', 'keydown'])
			window.addEventListener(name, stop, { passive: true });
		this.cancelRestore = stop;
		attempt();
	}
	dispose() {
		this.cancelRestore();
		this.listeners.clear();
	}
}

export function rememberPage<T>(key: string, tabKey?: string) {
	const view = getContext<PageHistory | undefined>(pageHistoryContext)?.view<T>(
		key,
		tabKey
	);
	onDestroy(() => view?.ready());
	return view;
}
