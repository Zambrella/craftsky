<script lang="ts">
	import { page } from '$app/state';
	import AppShell from '#lib/features/shell/AppShell.svelte';
	import Feed from '#lib/features/social/Feed.svelte';
	import Tag from '#lib/features/social/Tag.svelte';
	import Saved from '#lib/features/social/Saved.svelte';
	import Activity from '#lib/features/social/Activity.svelte';
	let { data } = $props();
	let path = $derived(data.path);
	let section = $derived(path[0] || 'home');
</script>

{#snippet loading()}<p class="empty" role="status">Loading…</p>{/snippet}
{#snippet loadError()}<section class="notice" role="alert">
		This page could not load. <button onclick={() => location.reload()}
			>Reload</button
		>
	</section>{/snippet}

{#key data.session.id}<AppShell session={data.session} accounts={data.accounts}
		>{#snippet children(
			api
		)}{#key (section === 'home' ? '/app/home' : page.url.pathname) + page.url.search}
				{#if section === 'home'}<Feed
						{api}
						did={data.session.did}
					/>{:else if section === 'search'}{#await import('#lib/features/social/Search.svelte')}{@render loading()}{:then Module}<Module.default
							{api}
							did={data.session.did}
						/>{:catch}{@render loadError()}{/await}{:else if section === 'profile' && path[1] === 'edit'}{#await import('#lib/features/social/ProfileEditor.svelte')}{@render loading()}{:then Module}<Module.default
							{api}
							session={data.session}
						/>{:catch}{@render loadError()}{/await}{:else if section === 'profile'}{#await import('#lib/features/social/Profile.svelte')}{@render loading()}{:then Module}<Module.default
							{api}
							session={data.session}
							id={path[1] || data.session.did}
						/>{:catch}{@render loadError()}{/await}{:else if section === 'posts' && path.length >= 3}{#await import('#lib/features/social/Thread.svelte')}{@render loading()}{:then Module}<Module.default
							{api}
							session={data.session}
							did={path[1]}
							rkey={path[2]}
							section={path[3]}
						/>{:catch}{@render loadError()}{/await}{:else if (section === 'tags' || section === 'tag') && path[1]}<Tag
						{api}
						did={data.session.did}
						tag={path[1]}
					/>{:else if section === 'saved'}<Saved
						{api}
						did={data.session.did}
					/>{:else if section === 'activity'}<Activity
						{api}
						did={data.session.did}
					/>{:else if section === 'messages'}{#await import('#lib/features/messages/MessagesWorkspace.svelte')}{@render loading()}{:then Module}<Module.default
							{api}
							{path}
							session={data.session}
						/>{:catch}{@render loadError()}{/await}{:else if section === 'settings'}{#await import('#lib/features/settings/SettingsWorkspace.svelte')}{@render loading()}{:then Module}<Module.default
							{api}
							{path}
							session={data.session}
						/>{:catch}{@render loadError()}{/await}{:else if section === 'compose'}{#await import('#lib/features/composer/ComposerWorkspace.svelte')}{@render loading()}{:then Module}<Module.default
							{api}
							{path}
							session={data.session}
						/>{:catch}{@render loadError()}{/await}{:else if section === 'drafts' || section === 'scheduled'}{#await import('#lib/features/drafts/DraftsWorkspace.svelte')}{@render loading()}{:then Module}<Module.default
							{api}
							{path}
							session={data.session}
						/>{:catch}{@render loadError()}{/await}{:else if section === 'events' && path.length >= 3}{#await import('#lib/features/social/Event.svelte')}{@render loading()}{:then Module}<Module.default
							{api}
							did={path[1]}
							rkey={path[2]}
							viewerDid={data.session.did}
						/>{:catch}{@render loadError()}{/await}{:else}<section
						class="panel stack"
					>
						<h1>Page not found</h1>
						<a href="/app/home">Back to your feed</a>
					</section>{/if}
			{/key}{/snippet}</AppShell
	>{/key}
