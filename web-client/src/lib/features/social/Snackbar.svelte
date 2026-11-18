<script lang="ts">
	import { getContext, untrack, type Snippet } from 'svelte';
	import ToastSurface from './ToastSurface.svelte';
	import { toastContext, type ToastHub } from './toasts.svelte';
	let {
		message,
		warning = false,
		support = false,
		onDismiss,
		children
	}: {
		message: string;
		warning?: boolean;
		support?: boolean;
		onDismiss: () => void;
		children?: Snippet;
	} = $props();
	const hub = getContext<ToastHub | undefined>(toastContext);
	const source = Symbol('toast-source');
	$effect(() => {
		if (!hub) return;
		if (!message) {
			untrack(() => hub.remove(source));
			return;
		}
		untrack(() =>
			hub.show({ source, message, warning, support, onDismiss, children })
		);
		// Recovery controls belong to their screen. Confirmed successes survive navigation.
		return () => {
			if (warning) untrack(() => hub.remove(source));
		};
	});
</script>

{#if !hub && message}<ToastSurface
		{message}
		{warning}
		{support}
		{onDismiss}
		{children}
	/>{/if}
