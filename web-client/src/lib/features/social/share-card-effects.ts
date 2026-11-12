import { createShareCardFoilMask } from './profile-share-card';
import { createShareCardShader } from './share-card-shader';

const limit = 5;
const bound = (value: number) => Math.max(-limit, Math.min(limit, value));

/** No device sensors; only pointer input and a once-per-open introduction. */
export function shareCardEffects(stage: HTMLElement, enabled: boolean) {
	const card = stage.querySelector<HTMLElement>('.artwork')!;
	const canvas = document.createElement('canvas');
	canvas.setAttribute('aria-hidden', 'true');
	canvas.className = 'share-card-foil';
	Object.assign(canvas.style, {
		position: 'absolute',
		inset: '0',
		width: '100%',
		height: '100%',
		pointerEvents: 'none'
	});
	card.append(canvas);
	let shader: ReturnType<typeof createShareCardShader>;
	let disposed = false,
		frame = 0,
		last = 0,
		introStart = 0,
		introPlayed = false,
		intro = false;
	let x = 0,
		y = 0,
		targetX = 0,
		targetY = 0;
	let pointer: number | undefined,
		pointerX = 0,
		pointerY = 0;
	const reduced = matchMedia('(prefers-reduced-motion: reduce)');
	const active = () => enabled && !document.hidden && !reduced.matches;
	const paint = () => {
		card.style.transform = `perspective(900px) rotateX(${x}deg) rotateY(${y}deg)`;
		shader?.draw(x, y);
	};
	function tick(time: number) {
		frame = 0;
		if (!active()) {
			reset();
			return;
		}
		const dt = last ? Math.min((time - last) / 1000, 0.1) : 0;
		last = time;
		let goalX = targetX,
			goalY = targetY;
		if (intro) {
			if (!introStart) introStart = time;
			const progress = Math.min((time - introStart) / 1500, 1);
			const envelope = Math.sin(progress * Math.PI);
			goalX += Math.sin(progress * Math.PI * 2) * envelope * 4;
			goalY += Math.sin(progress * Math.PI * 3) * envelope * 3.25;
			if (progress === 1) intro = false;
		}
		const amount = 1 - Math.exp(-dt * 9);
		x += (bound(goalX) - x) * amount;
		y += (bound(goalY) - y) * amount;
		if (!intro && Math.abs(goalX - x) + Math.abs(goalY - y) < 0.001) {
			x = targetX;
			y = targetY;
			last = 0;
			paint();
			return;
		}
		paint();
		wake();
	}
	function wake() {
		if (!frame && active()) frame = requestAnimationFrame(tick);
	}
	function reset() {
		cancelAnimationFrame(frame);
		frame = 0;
		last = 0;
		intro = false;
		if (pointer !== undefined && stage.hasPointerCapture(pointer))
			stage.releasePointerCapture(pointer);
		pointer = undefined;
		stage.classList.remove('dragging');
		x = y = targetX = targetY = 0;
		paint();
	}
	function sync() {
		stage.classList.toggle('motion-enabled', active());
		if (!active()) {
			reset();
			return;
		}
		if (!introPlayed) {
			introPlayed = true;
			intro = true;
			introStart = 0;
		}
		wake();
	}
	function down(event: PointerEvent) {
		if (!active() || !event.isPrimary || event.button !== 0) return;
		intro = false;
		pointer = event.pointerId;
		pointerX = event.clientX;
		pointerY = event.clientY;
		targetX = x;
		targetY = y;
		stage.setPointerCapture(pointer);
		stage.classList.add('dragging');
	}
	function move(event: PointerEvent) {
		if (!active()) return;
		const rect = stage.getBoundingClientRect();
		if (!rect.width || !rect.height) return;
		if (pointer === event.pointerId) {
			targetX = bound(
				targetX - ((event.clientY - pointerY) / rect.height) * limit * 4
			);
			targetY = bound(
				targetY + ((event.clientX - pointerX) / rect.width) * limit * 4
			);
			pointerX = event.clientX;
			pointerY = event.clientY;
		} else if (pointer === undefined && event.pointerType === 'mouse') {
			intro = false;
			targetX = bound(
				-((event.clientY - rect.top) / rect.height - 0.5) * limit * 2
			);
			targetY = bound(
				((event.clientX - rect.left) / rect.width - 0.5) * limit * 2
			);
		} else return;
		wake();
	}
	function release(event: PointerEvent) {
		if (pointer !== event.pointerId) return;
		pointer = undefined;
		stage.classList.remove('dragging');
		if (stage.hasPointerCapture(event.pointerId))
			stage.releasePointerCapture(event.pointerId);
		targetX = targetY = 0;
		wake();
	}
	function leave() {
		if (pointer === undefined) {
			intro = false;
			targetX = targetY = 0;
			wake();
		}
	}
	function resize() {
		const ratio = Math.min(devicePixelRatio || 1, 2);
		canvas.width = Math.max(1, Math.round(stage.clientWidth * ratio));
		canvas.height = Math.max(1, Math.round(stage.clientHeight * ratio));
		paint();
	}
	const observer = new ResizeObserver(resize);
	observer.observe(stage);
	stage.addEventListener('pointerdown', down);
	stage.addEventListener('pointermove', move);
	stage.addEventListener('pointerup', release);
	stage.addEventListener('pointercancel', release);
	stage.addEventListener('lostpointercapture', release);
	stage.addEventListener('pointerleave', leave);
	reduced.addEventListener('change', sync);
	document.addEventListener('visibilitychange', sync);
	// Absence of WebGL/assets affects decoration only; the PNG and tilt still work.
	void createShareCardFoilMask()
		.then((mask) => {
			if (disposed) return;
			shader = createShareCardShader(canvas, mask);
			canvas.hidden = !shader;
			stage.dataset.foil = shader ? 'webgl' : 'static';
			resize();
		})
		.catch(() => {
			if (!disposed) {
				canvas.hidden = true;
				stage.dataset.foil = 'static';
			}
		});
	sync();
	return {
		update(value: boolean) {
			enabled = value;
			sync();
		},
		destroy() {
			disposed = true;
			reset();
			observer.disconnect();
			shader?.destroy();
			canvas.remove();
			stage.removeEventListener('pointerdown', down);
			stage.removeEventListener('pointermove', move);
			stage.removeEventListener('pointerup', release);
			stage.removeEventListener('pointercancel', release);
			stage.removeEventListener('lostpointercapture', release);
			stage.removeEventListener('pointerleave', leave);
			reduced.removeEventListener('change', sync);
			document.removeEventListener('visibilitychange', sync);
		}
	};
}
