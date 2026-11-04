import { getContext, onDestroy } from 'svelte';
import { page } from '$app/state';

type Entry = { y: number; state: Map<string, unknown>; loading: number };
const entry = (): Entry => ({ y: 0, state: new Map(), loading: 0 });
export const pageHistoryContext = Symbol('page-history');

/** Account-scoped, in-memory view state for individual browser history entries. */
export class PageHistory {
	private entries = new Map<number, Entry>();
	private current = entry();
	private pending: Entry | undefined;
	private popping = false;
	private cancelRestore = () => {};
	leave(index: number, target: number, pop: boolean) {
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
		if (pop) this.restore(this.current);
	}
	view<T>(key: string) {
		const owner =
			(this.popping && page.state.navigationIndex !== undefined
				? this.entries.get(page.state.navigationIndex)
				: undefined) ||
			this.pending ||
			this.current;
		owner.loading++;
		let finished = false;
		return {
			value: owner.state.get(key) as T | undefined,
			save: (value: T) => owner.state.set(key, value),
			ready: () => {
				if (!finished) {
					owner.loading--;
					finished = true;
				}
			}
		};
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
	}
}

export function rememberPage<T>(key: string) {
	const view = getContext<PageHistory | undefined>(pageHistoryContext)?.view<T>(
		key
	);
	onDestroy(() => view?.ready());
	return view;
}
