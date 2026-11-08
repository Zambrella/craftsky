import { mount, unmount, onDestroy } from 'svelte';
import { goto } from '$app/navigation';
import PromptDialog from './PromptDialog.svelte';

export interface PromptOptions {
	title: string;
	detail: string;
	confirmLabel: string;
	danger?: boolean;
	cancelLabel?: string;
	href?: string;
	target?: string;
}

/** Native-style, focus-restoring confirmation, including when nested in an editor. */
export function prompt(
	options: PromptOptions,
	signal?: AbortSignal
): Promise<boolean> {
	if (signal?.aborted) return Promise.resolve(false);
	return new Promise((resolve) => {
		let settled = false;
		const cancel = () => finish(false);
		const finish = (accepted: boolean) => {
			if (settled) return;
			settled = true;
			signal?.removeEventListener('abort', cancel);
			void unmount(component).then(() => resolve(accepted));
		};
		const component = mount(PromptDialog, {
			target: document.body,
			props: { ...options, onResult: finish }
		});
		signal?.addEventListener('abort', cancel, { once: true });
	});
}

/** The final anchor click is a user gesture, so new-tab links survive popup blockers. */
export function confirmExternalLink(
	event: MouseEvent,
	anchor: EventTarget | null = event.currentTarget
) {
	if (!(anchor instanceof HTMLAnchorElement)) return;
	const href = anchor.href;
	const url = new URL(href);
	if (
		['https:', 'http:'].includes(url.protocol) &&
		url.hostname === 'app.craftsky.social' &&
		!url.port &&
		!url.username &&
		!url.password &&
		!url.pathname.startsWith('//')
	) {
		const destination = `${url.pathname}${url.search}${url.hash}`;
		anchor.href = destination;
		anchor.removeAttribute('target');
		// Use the router even when called by the shell's document-level listener.
		// Modified clicks retain the browser's native new-tab/window behavior.
		if (
			!event.button &&
			!event.metaKey &&
			!event.ctrlKey &&
			!event.shiftKey &&
			!event.altKey
		) {
			event.preventDefault();
			void goto(destination);
		}
		return;
	}
	if (url.origin === location.origin) return;
	event.preventDefault();
	if (!['https:', 'http:', 'mailto:'].includes(url.protocol)) return;
	void prompt({
		title: 'Open link?',
		detail: 'This will open outside CraftSky.',
		confirmLabel: 'Open link',
		href,
		target: anchor.target || undefined
	});
}

/** Cancel a pending choice if its owning screen/account is disposed. */
export function usePrompt() {
	const controller = new AbortController();
	onDestroy(() => controller.abort());
	return (options: PromptOptions) => prompt(options, controller.signal);
}
