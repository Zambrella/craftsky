<script lang="ts">
	import { onMount } from 'svelte';
	import { applyConsent } from './client.ts';
	import { installLocalDiagnostics } from './local.ts';
	onMount(() => {
		const removeLocalDiagnostics = installLocalDiagnostics();
		void applyConsent();
		const update = () => {
			void applyConsent();
		};
		window.addEventListener('craftsky-diagnostics-change', update);
		window.addEventListener('storage', update);
		return () => {
			removeLocalDiagnostics();
			window.removeEventListener('craftsky-diagnostics-change', update);
			window.removeEventListener('storage', update);
		};
	});
</script>
