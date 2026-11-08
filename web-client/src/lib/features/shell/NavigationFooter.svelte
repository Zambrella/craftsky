<script lang="ts">
	import Icon from './Icon.svelte';
	const version = __CRAFTSKY_VERSION__.trim();
</script>

<div class="sidebar-about">
	<a href="https://craftsky.social/terms" target="_blank" rel="noreferrer"
		>Terms</a
	><a href="https://craftsky.social/privacy" target="_blank" rel="noreferrer"
		>Privacy</a
	><a
		href="https://userinput.app/s/did:plc:lmmx63zcns6gewgxqfdt4kof/3mpr5izppvt2k?lang=en"
		target="_blank"
		rel="noreferrer"
		class="sidebar-feedback"><Icon name="messages" size={20} />Feedback</a
	>
	{#if version}<small>{version}</small>{/if}
</div>
