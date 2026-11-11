<script lang="ts">
	import { getContext } from 'svelte';
	import SettingsPageLayout from './SettingsPageLayout.svelte';
	import { title } from './model.ts';
	import ConnectionsSkeleton from './ConnectionsSkeleton.svelte';
	let {
		section,
		profileId,
		failed = false
	}: { section: string; profileId?: string; failed?: boolean } = $props();
	const back = getContext<((event: MouseEvent) => void) | undefined>(
		'craftsky-back'
	);
</script>

<SettingsPageLayout
	{section}
	backHref={profileId
		? '/profiles/' + encodeURIComponent(profileId)
		: '/settings'}
	backLabel={profileId ? 'Back to profile' : 'Back to Settings'}
	onBack={profileId ? back : undefined}
>
	<h1 class="workspace-heading">{title(section)} (0)</h1>
	{#if failed}<section class="notice" role="alert">
			This page could not load. <button onclick={() => location.reload()}
				>Reload</button
			>
		</section>{:else}<ConnectionsSkeleton />{/if}
</SettingsPageLayout>
