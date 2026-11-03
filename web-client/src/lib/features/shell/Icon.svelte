<script lang="ts">
	import { nativeIconAliases, nativeIconPaths } from './native-icons.ts';
	let {
		name,
		size = 24,
		filled = false
	}: { name: string; size?: number; filled?: boolean } = $props();
	let icon = $derived(
		nativeIconPaths[nativeIconAliases[name] || name] || nativeIconPaths.info
	);
</script>

<svg
	width={size}
	height={size}
	viewBox="0 0 1024 1024"
	fill="currentColor"
	aria-hidden="true"
	focusable="false"
	><path d={filled ? icon.fill || icon.regular : icon.regular} /></svg
>
