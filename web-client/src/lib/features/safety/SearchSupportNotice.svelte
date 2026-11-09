<script lang="ts">
	import guide from './guide-data.json';
	import { shouldOfferSearchSupport } from './support-search';
	let { query }: { query: string } = $props();
	let dismissed = $state(false);
</script>

{#if !dismissed && shouldOfferSearchSupport(query)}<aside aria-label="Support">
		<p role="status">{guide.searchNotice}</p>
		<div>
			<a href="/safety">Find support</a><button
				type="button"
				onclick={() => (dismissed = true)}>Dismiss</button
			>
		</div>
	</aside>{/if}

<style>
	aside {
		padding: 16px;
		border: 1.5px solid var(--ink);
		border-radius: 14px;
		margin: 16px 0;
	}
	p {
		line-height: 1.5;
	}
	div {
		display: flex;
		gap: 16px;
		align-items: center;
		flex-wrap: wrap;
	}
	a {
		text-decoration: underline;
	}
	button {
		background: transparent;
		color: inherit;
		border: 0;
		box-shadow: none;
		padding: 8px;
	}
</style>
