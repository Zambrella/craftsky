<script lang="ts">
	import AuthenticatedPage from '#lib/features/shell/AuthenticatedPage.svelte';
	let { data } = $props();
</script>

<AuthenticatedPage {data} />
