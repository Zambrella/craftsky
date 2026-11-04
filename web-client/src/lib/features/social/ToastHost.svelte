<script lang="ts">
	import { onMount } from 'svelte';
	import ToastSurface from './ToastSurface.svelte';
	import { type ToastHub } from './toasts.svelte';
	let { hub, fullWidth = false }: { hub: ToastHub; fullWidth?: boolean } =
		$props();
	function portal(node: HTMLElement) {
		function move() {
			const target =
				[...document.querySelectorAll('dialog[open]')].at(-1) || document.body;
			if (node.parentElement !== target) target.append(node);
		}
		move();
		const observer = new MutationObserver(move);
		observer.observe(document.body, {
			subtree: true,
			attributes: true,
			attributeFilter: ['open'],
			childList: true
		});
		return {
			destroy() {
				observer.disconnect();
				node.remove();
			}
		};
	}
	onMount(() => {
		const receive = (event: Event) => {
			const { message, warning } = (
				event as CustomEvent<{ message: string; warning: boolean }>
			).detail;
			if (typeof message === 'string' && message)
				hub.show({
					source: Symbol('completed-action'),
					message,
					warning: !!warning,
					onDismiss() {}
				});
		};
		window.addEventListener('craftsky-toast', receive);
		return () => window.removeEventListener('craftsky-toast', receive);
	});
</script>

<div
	use:portal
	style:--app-content-left={fullWidth ? '0px' : undefined}
	style:--app-content-width={fullWidth ? '100vw' : undefined}
	style:--toast-bottom={fullWidth
		? 'calc(16px + env(safe-area-inset-bottom))'
		: undefined}
>
	{#if hub.items[0]}{@const toast = hub.items[0]}
		{#key toast.source}<ToastSurface
				message={toast.message}
				warning={toast.warning}
				children={toast.children}
				onDismiss={() => hub.dismiss(toast.source)}
			/>{/key}
	{/if}
</div>
