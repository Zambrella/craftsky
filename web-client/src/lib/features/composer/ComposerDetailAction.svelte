<script lang="ts">
	let { ondone, disabled = false }: { ondone: () => void; disabled?: boolean } =
		$props();

	let keyboardVisible = $state(false);

	function fitViewport(button: HTMLButtonElement) {
		const viewport = window.visualViewport;
		const page = button.parentElement;
		if (!viewport || !page) return;

		function update() {
			if (!viewport || !page) return;
			const bottom = viewport.offsetTop + viewport.height;
			keyboardVisible =
				viewport.scale === 1 && window.innerHeight - bottom > 100;
			if (keyboardVisible)
				page.style.setProperty('--composer-detail-height', `${bottom}px`);
			else page.style.removeProperty('--composer-detail-height');
		}
		update();
		viewport.addEventListener('resize', update);
		viewport.addEventListener('scroll', update);
		return {
			destroy() {
				viewport.removeEventListener('resize', update);
				viewport.removeEventListener('scroll', update);
				page.style.removeProperty('--composer-detail-height');
			}
		};
	}

	function done(event: MouseEvent & { currentTarget: HTMLButtonElement }) {
		if (!keyboardVisible) {
			ondone();
			return;
		}
		event.currentTarget.focus({ preventScroll: true });
		const { virtualKeyboard } = navigator as Navigator & {
			virtualKeyboard?: { hide(): void };
		};
		virtualKeyboard?.hide();
	}
</script>

<button
	use:fitViewport
	type="button"
	inputmode="none"
	class="detail-action"
	class:compact={keyboardVisible}
	{disabled}
	onpointerdown={(event) => {
		if (keyboardVisible) event.preventDefault();
	}}
	onclick={done}>Done</button
>

<style>
	.detail-action {
		position: absolute;
		z-index: 1;
		inset-inline-end: 16px;
		bottom: calc(16px + env(safe-area-inset-bottom));
		width: calc(100% - 32px);
		min-height: 48px;
		padding: 8px 24px;
		border: 2px solid var(--ink);
		border-radius: 999px;
		background: var(--primary);
		color: var(--on-primary, white);
		box-shadow: 3px 3px var(--shadow);
		font-weight: 800;
	}
	.detail-action.compact {
		width: auto;
		background: var(--surface);
		color: var(--ink);
	}
</style>
