<script lang="ts">
	let { name }: { name: string } = $props();
	const paths: Record<string, string> = {
		home: 'm3 10 9-7 9 7v10a1 1 0 0 1-1 1h-6v-7H9v7H4a1 1 0 0 1-1-1z',
		search: 'M21 21l-5-5M19 10a8 8 0 1 1-16 0 8 8 0 0 1 16 0',
		messages: 'M21 11a9 9 0 0 1-9 9H3l2-5a9 9 0 1 1 16-4',
		activity: 'M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4',
		profile: 'M16 7a4 4 0 1 1-8 0 4 4 0 0 1 8 0M4 21v-2a8 8 0 0 1 16 0v2',
		saved: 'M6 3h12v18l-6-4-6 4z',
		scheduled: 'M12 7v5l3 2M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0',
		drafts: 'M14 2H4v20h16V8zM14 2v6h6M8 12h8M8 16h6',
		settings:
			'm9 3-1 3-3 1-2 4 2 2-1 3 3 3 3-1 2 2 4-2 1-3 3-1v-4l-3-1-1-3h-4zM15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0',
		menu: 'M3 6h18M3 12h18M3 18h18',
		plus: 'M12 5v14M5 12h14'
	};
</script>

<svg
	width="24"
	height="24"
	viewBox="0 0 24 24"
	fill="none"
	stroke="currentColor"
	stroke-width="1.8"
	stroke-linecap="round"
	stroke-linejoin="round"
	aria-hidden="true"><path d={paths[name] || paths.home} /></svg
>
