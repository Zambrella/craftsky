<script lang="ts">
	let { size = 24 }: { size?: number } = $props();
</script>

<svg
	xmlns="http://www.w3.org/2000/svg"
	width={size}
	height={size}
	viewBox="0 0 24 24"
	fill="none"
	aria-hidden="true"
	focusable="false"
	><path d="M3.5 10.85H19.8L22 12L19.8 13.15H3.5Z" fill="currentColor" /><circle
		cx="3"
		cy="12"
		r="2"
		fill="currentColor"
	/><path
		fill-rule="evenodd"
		d="M12 1.5C10.7 1.5 10 2.6 10 4.5L10.8 16L12 22.5L13.2 16L14 4.5C14 2.6 13.3 1.5 12 1.5ZM12 3.1C11.55 3.1 11.3 3.6 11.3 4.5V6.2C11.3 7.4 12.7 7.4 12.7 6.2V4.5C12.7 3.6 12.45 3.1 12 3.1Z"
		fill="currentColor"
	/></svg
>
