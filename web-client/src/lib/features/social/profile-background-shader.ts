// Use exactly the same effect on Flutter and the web; only the platform entry
// point differs. Vite bundles the source, with no runtime asset/network fetch.
import source from '../../../../../app/shaders/profile_background.frag?raw';

export function profileShaderStyle(background?: string): number | undefined {
	switch (background) {
		case 'animated-flow':
			return 0;
		case 'animated-waves':
			return 1;
		case 'animated-weave':
			return 2;
		case 'animated-grain':
			return 3;
		case 'animated-marble':
			return 4;
		case 'animated-textile':
			return 5;
		default:
			return undefined;
	}
}

const vertex = `attribute vec2 position;
void main() { gl_Position = vec4(position, 0.0, 1.0); }`;
const fragment = `
#ifdef GL_FRAGMENT_PRECISION_HIGH
precision highp float;
#else
precision mediump float;
#endif
${source
	.replace('#include <flutter/runtime_effect.glsl>', '')
	.replace('out vec4 fragColor;', '')
	.replaceAll(
		'FlutterFragCoord().xy',
		'vec2(gl_FragCoord.x, uSize.y - gl_FragCoord.y)'
	)
	.replaceAll('fragColor', 'gl_FragColor')}`;

/** Optional, bounded decoration; an unavailable GPU retains the CSS still. */
function createShader(canvas: HTMLCanvasElement, style: number) {
	const gl = canvas.getContext('webgl', {
		alpha: true,
		antialias: false,
		depth: false,
		premultipliedAlpha: true
	});
	if (!gl) return undefined;
	const shaders: WebGLShader[] = [];
	const program = gl.createProgram(),
		buffer = gl.createBuffer();
	const onLost = (event: Event) => {
		event.preventDefault();
		canvas.hidden = true;
	};
	const destroy = () => {
		canvas.removeEventListener('webglcontextlost', onLost);
		gl.deleteBuffer(buffer);
		gl.deleteProgram(program);
		for (const shader of shaders) gl.deleteShader(shader);
		gl.getExtension('WEBGL_lose_context')?.loseContext();
	};
	try {
		if (!program || !buffer) {
			destroy();
			return undefined;
		}
		for (const [type, text] of [
			[gl.VERTEX_SHADER, vertex],
			[gl.FRAGMENT_SHADER, fragment]
		] as const) {
			const shader = gl.createShader(type);
			if (!shader) {
				destroy();
				return undefined;
			}
			shaders.push(shader);
			gl.shaderSource(shader, text);
			gl.compileShader(shader);
			if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
				destroy();
				return undefined;
			}
			gl.attachShader(program, shader);
		}
		gl.linkProgram(program);
		if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
			destroy();
			return undefined;
		}
		gl.useProgram(program);
		gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
		gl.bufferData(
			gl.ARRAY_BUFFER,
			new Float32Array([-1, -1, 1, -1, -1, 1, -1, 1, 1, -1, 1, 1]),
			gl.STATIC_DRAW
		);
		const position = gl.getAttribLocation(program, 'position');
		gl.enableVertexAttribArray(position);
		gl.vertexAttribPointer(position, 2, gl.FLOAT, false, 0, 0);
		gl.uniform1f(gl.getUniformLocation(program, 'uStyle'), style);
		const size = gl.getUniformLocation(program, 'uSize'),
			phase = gl.getUniformLocation(program, 'uPhase');
		canvas.hidden = false;
		canvas.addEventListener('webglcontextlost', onLost);
		return {
			draw(value: number) {
				if (gl.isContextLost()) return false;
				gl.viewport(0, 0, canvas.width, canvas.height);
				gl.uniform2f(size, canvas.width, canvas.height);
				gl.uniform1f(phase, value);
				gl.drawArrays(gl.TRIANGLES, 0, 6);
				return true;
			},
			destroy
		};
	} catch {
		destroy();
		return undefined;
	}
}

/** Animate only visible, foreground headers; reduced motion draws phase zero. */
export function mountProfileShader(canvas: HTMLCanvasElement, style: number) {
	const shader = createShader(canvas, style);
	if (!shader) {
		canvas.hidden = true;
		return () => {};
	}
	const reduced = matchMedia('(prefers-reduced-motion: reduce)');
	let visible = false,
		frame = 0,
		lastTime = 0,
		elapsed = 0,
		lastPaint = 0;
	const stop = () => {
		cancelAnimationFrame(frame);
		frame = 0;
		lastTime = 0;
	};
	const tick = (now: number) => {
		if (lastTime) elapsed += Math.min(now - lastTime, 100);
		lastTime = now;
		// Slow shading needs at most 30fps, independently of display refresh rate.
		if (now - lastPaint >= 1000 / 30) {
			lastPaint = now;
			if (!shader.draw(((elapsed % 24000) / 24000) * Math.PI * 2)) {
				stop();
				return;
			}
		}
		frame = requestAnimationFrame(tick);
	};
	const sync = () => {
		stop();
		if (reduced.matches) {
			elapsed = 0;
			shader.draw(0);
		} else if (visible && !document.hidden) frame = requestAnimationFrame(tick);
	};
	const resize = new ResizeObserver(() => {
		const bounds = canvas.getBoundingClientRect(),
			scale = Math.min(devicePixelRatio || 1, 1.5);
		canvas.width = Math.max(
			1,
			Math.min(1024, Math.round(bounds.width * scale))
		);
		canvas.height = Math.max(
			1,
			Math.min(512, Math.round(bounds.height * scale))
		);
		shader.draw(reduced.matches ? 0 : (elapsed / 24000) * Math.PI * 2);
	});
	const intersection = new IntersectionObserver(([entry]) => {
		visible = entry.isIntersecting;
		sync();
	});
	resize.observe(canvas);
	intersection.observe(canvas);
	reduced.addEventListener('change', sync);
	document.addEventListener('visibilitychange', sync);
	return () => {
		stop();
		resize.disconnect();
		intersection.disconnect();
		reduced.removeEventListener('change', sync);
		document.removeEventListener('visibilitychange', sync);
		shader.destroy();
	};
}
