import type { Snippet } from 'svelte';
export const toastContext = Symbol('action-toasts');
export type Toast = {
	source: symbol;
	message: string;
	warning: boolean;
	onDismiss: () => void;
	children?: Snippet;
	support?: boolean;
};
export class ToastHub {
	items = $state<Toast[]>([]);
	private owner: string | undefined;
	scope(owner: string) {
		if (owner === this.owner) return;
		this.owner = owner;
		this.items = [];
	}
	show(toast: Toast) {
		const index = this.items.findIndex((item) => item.source === toast.source);
		if (
			index < 0 &&
			this.items.some(
				(item) =>
					item.message === toast.message && item.warning === toast.warning
			)
		)
			return;
		if (index >= 0)
			this.items = this.items.map((item, i) => (i === index ? toast : item));
		else
			this.items =
				this.items.length < 5
					? [...this.items, toast]
					: [this.items[0], ...this.items.slice(-3), toast];
		// An action failure must be visible immediately, ahead of confirmations.
		const failure = this.items.findIndex((item) => item.warning);
		if (failure > 0)
			this.items = [
				this.items[failure],
				...this.items.filter((_, i) => i !== failure)
			];
	}
	remove(source: symbol) {
		this.items = this.items.filter((item) => item.source !== source);
	}
	dismiss(source: symbol) {
		const item = this.items.find((item) => item.source === source);
		this.remove(source);
		item?.onDismiss();
	}
}
/** For accepted actions whose originating screen is about to navigate away. */
export function notify(message: string, warning = false, support = false) {
	if (typeof window !== 'undefined')
		window.dispatchEvent(
			new CustomEvent('craftsky-toast', {
				detail: { message, warning, support }
			})
		);
}
