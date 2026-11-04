<script lang="ts">
	import { onMount } from 'svelte';
	import { applyConsent } from './client.ts';
	onMount(() => {
		void applyConsent();
		const update = () => {
			void applyConsent();
		};
		window.addEventListener('craftsky-diagnostics-change', update);
		window.addEventListener('storage', update);
		return () => {
			window.removeEventListener('craftsky-diagnostics-change', update);
			window.removeEventListener('storage', update);
		};
	});
</script>
