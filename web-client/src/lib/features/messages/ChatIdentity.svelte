<script lang="ts">
	import type { CraftskyAPI, ChatActor, Profile } from '#lib/api/client.ts';
	import CompactIdentity from '../social/CompactIdentity.svelte';
	let { api, peer }: { api: CraftskyAPI; peer: ChatActor } = $props();
	// Message state snapshots can replace peer without changing the person.
	const peerDid = $derived(peer.did);
	let profile = $state<Profile>();
	$effect(() => {
		const client = api,
			did = peerDid;
		let current = true;
		profile = undefined;
		void client.profiles
			.get(did)
			.then((value) => {
				if (current && value.did === did) profile = value;
			})
			.catch(() => {});
		return () => {
			current = false;
		};
	});
</script>

<CompactIdentity
	author={profile || { ...peer, handle: peer.handle || '', crafts: [] }}
/>
