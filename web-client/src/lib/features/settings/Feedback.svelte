<script lang="ts">
	import type { Work } from './work.svelte.ts';
	let { work, reload }: { work: Work; reload?: () => void } = $props();
</script>

{#if work.loading}<p class="loading" role="status">Loading…</p>{/if}
{#if work.error}<div class="error" role="alert">
		<p>{work.error}</p>
		<div class="actions">
			{#if work.retry}<button
					disabled={work.busy}
					onclick={() => work.retry?.()}>Check and retry action</button
				>{:else if reload}<button
					disabled={work.busy || work.loading}
					onclick={reload}>Try again</button
				>{/if}
		</div>
	</div>{/if}
{#if work.notice}<p class="notice" role="status">{work.notice}</p>{/if}
