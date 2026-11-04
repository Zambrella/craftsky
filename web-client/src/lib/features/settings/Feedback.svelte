<script lang="ts">
	import type { Work } from './work.svelte.ts';
	import Snackbar from '../social/Snackbar.svelte';
	let { work, reload }: { work: Work; reload?: () => void } = $props();
</script>

{#if work.loading}<p class="loading" role="status">Loading…</p>{/if}
<Snackbar message={work.error} warning onDismiss={() => (work.error = '')}>
	{#if work.retry}<button
			type="button"
			disabled={work.busy}
			onclick={() => work.retry?.()}>Check and retry action</button
		>
	{:else if reload}<button
			type="button"
			disabled={work.busy || work.loading}
			onclick={reload}>Try again</button
		>{/if}
</Snackbar>{#if !work.error && work.retry}<p class="action-recovery">
		This action still needs confirmation. <button
			type="button"
			disabled={work.busy}
			onclick={() => work.retry?.()}>Check and retry action</button
		>
	</p>{/if}
<Snackbar message={work.notice} onDismiss={() => (work.notice = '')} />
