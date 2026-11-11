<script lang="ts">
	import { getContext } from 'svelte';
	import type { CraftskyAPI } from '#lib/api/client.ts';
	import SettingsPageLayout from './SettingsPageLayout.svelte';
	import Connections from './Connections.svelte';
	let {
		api,
		section,
		profileId
	}: { api: CraftskyAPI; section: string; profileId?: string } = $props();
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
	<Connections {api} {section} {profileId} />
</SettingsPageLayout>
