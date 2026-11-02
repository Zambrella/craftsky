<script lang="ts">
	import type { Work } from './work.svelte';
	let { work }: { work: Work } = $props();
</script>

{#if work.busy}<p role="status" class="loading">Loading…</p>{/if}
{#if work.error}<div class="notice error" role="alert">
		<p>{work.error}</p>
		{#if work.retry}<button disabled={work.busy} onclick={() => work.retry?.()}
				>Retry</button
			>{/if}
	</div>{/if}
