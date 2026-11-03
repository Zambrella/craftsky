import { afterNavigate, beforeNavigate, goto } from '$app/navigation';
import { usePrompt, type PromptOptions } from './prompt';

/** Keep browser unload protection; style internal navigations with the native dialog. */
export function confirmNavigation(
	dirty: () => boolean,
	options: PromptOptions
) {
	const ask = usePrompt();
	let accepted = false;
	let pending = false;
	afterNavigate(() => {
		accepted = false;
	});
	beforeNavigate((nav) => {
		if (accepted || !dirty()) return;
		nav.cancel();
		// For document unloads the browser owns the required confirmation UI.
		if (nav.willUnload || pending || !nav.to) return;
		pending = true;
		void ask(options).then((result) => {
			pending = false;
			if (!result) return;
			accepted = true;
			if (nav.type === 'popstate') history.go(nav.delta);
			else
				void goto(nav.to!.url).finally(() => {
					accepted = false;
				});
		});
	});
}
