<script lang="ts">
	import type { CraftskyAPI } from '#lib/api/client.ts';
	import type { Session } from './model.ts';
	import Business from './Business.svelte';
	import './settings.css';
	let {
		api,
		session,
		productIndex
	}: {
		api: CraftskyAPI;
		session: Session;
		productIndex?: string;
	} = $props();
</script>

<section class="settings-workspace product-composer">
	<Business {api} {session} {productIndex} section="product" />
</section>

<style>
	.product-composer {
		width: 100%;
		height: 100dvh;
		min-height: 420px;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		padding: 0;
		background: var(--paper);
	}
</style>
