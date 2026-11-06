<script lang="ts">
	import AuthenticatedPage from '#lib/features/shell/AuthenticatedPage.svelte';
	import GuestPage from '#lib/features/browsing/GuestPage.svelte';
	import type { PageData } from './$types';
	let { data }: { data: PageData } = $props();
</script>

{#if data.guest}<GuestPage path={data.guest.path} />{:else}<AuthenticatedPage
		{data}
	/>{/if}
