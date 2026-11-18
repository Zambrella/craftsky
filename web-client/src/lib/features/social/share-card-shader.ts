const vertex = `
attribute vec2 position;
varying vec2 uv;
void main() { uv = (position + 1.0) * 0.5; gl_Position = vec4(position, 0.0, 1.0); }
`;
const fragment = `
#ifdef GL_FRAGMENT_PRECISION_HIGH
precision highp float;
#else
precision mediump float;
#endif
varying vec2 uv;
uniform sampler2D pattern;
uniform vec2 tilt;
float roundedRect(vec2 p, vec2 halfSize, float radius) {
  vec2 q = abs(p) - halfSize + radius;
  return length(max(q, 0.0)) + min(max(q.x, q.y), 0.0) - radius;
}
void main() {
  vec2 point = vec2(uv.x, 1.0 - uv.y) * vec2(480.0, 600.0);
  vec2 centred = point - vec2(240.0, 300.0);
  float edge = roundedRect(centred / 600.0, vec2(238.5, 298.5) / 600.0, 20.5 / 600.0) * 600.0;
  float rim = 1.0 - smoothstep(0.6, 2.5, abs(edge));
  float bevel = 1.0 - smoothstep(2.5, 6.0, -edge);
  float angle = -0.8 + (tilt.x - tilt.y) * 6.0;
  vec2 direction = vec2(cos(angle), sin(angle));
  vec2 normal = (centred / 600.0) / max(length(centred / 600.0), 0.001);
  float glint = pow(max(dot(normal, direction), 0.0), 20.0)
      + pow(max(dot(normal, -direction), 0.0), 28.0) * 0.4;
  glint = min(glint, 1.0);
  vec2 light = vec2(clamp(0.28 - tilt.y * 7.0, 0.05, 0.95), clamp(0.25 + tilt.x * 7.0, 0.05, 0.95)) * vec2(480.0, 600.0);
  float distanceToLight = length((point - light) / 298.0);
  float foil = (1.0 - smoothstep(0.0, 1.0, distanceToLight));
  foil = foil * foil * texture2D(pattern, uv).a * 0.3;
  float rimAlpha = rim * (0.22 + glint * 0.76);
  float bevelAlpha = bevel * 0.24 * (1.0 - rimAlpha);
  vec3 paper = vec3(245.0, 239.0, 228.0) / 255.0;
  vec3 silver = mix(vec3(155.0, 194.0, 230.0) / 255.0, paper, glint);
  vec3 colour = silver * rimAlpha + vec3(21.0, 53.0, 214.0) / 255.0 * 0.65 * bevelAlpha;
  float edgeAlpha = rimAlpha + bevelAlpha;
  gl_FragColor = vec4(colour + paper * foil * (1.0 - edgeAlpha), edgeAlpha + foil * (1.0 - edgeAlpha));
}
`;

/** Optional decoration: unsupported/lost contexts leave the complete PNG visible. */
export function createShareCardShader(
	canvas: HTMLCanvasElement,
	mask: HTMLCanvasElement
) {
	const gl = canvas.getContext('webgl', {
		alpha: true,
		antialias: false,
		depth: false,
		premultipliedAlpha: true
	});
	if (!gl) return undefined;
	const shaders: WebGLShader[] = [];
	const program = gl.createProgram(),
		buffer = gl.createBuffer(),
		texture = gl.createTexture();
	let lost = false;
	const onLost = () => {
		lost = true;
		canvas.hidden = true;
	};
	const destroy = () => {
		canvas.removeEventListener('webglcontextlost', onLost);
		gl.deleteTexture(texture);
		gl.deleteBuffer(buffer);
		gl.deleteProgram(program);
		for (const shader of shaders) gl.deleteShader(shader);
		gl.getExtension('WEBGL_lose_context')?.loseContext();
	};
	try {
		if (!program || !buffer || !texture) {
			destroy();
			return undefined;
		}
		for (const [type, source] of [
			[gl.VERTEX_SHADER, vertex],
			[gl.FRAGMENT_SHADER, fragment]
		] as const) {
			const shader = gl.createShader(type);
			if (!shader) {
				destroy();
				return undefined;
			}
			shaders.push(shader);
			gl.shaderSource(shader, source);
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
		gl.bindTexture(gl.TEXTURE_2D, texture);
		gl.pixelStorei(gl.UNPACK_FLIP_Y_WEBGL, true);
		gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, mask);
		gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
		gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR);
		gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
		gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
		gl.uniform1i(gl.getUniformLocation(program, 'pattern'), 0);
		const tilt = gl.getUniformLocation(program, 'tilt');
		canvas.addEventListener('webglcontextlost', onLost);
		return {
			draw(x: number, y: number) {
				if (lost) return;
				gl.viewport(0, 0, canvas.width, canvas.height);
				gl.uniform2f(tilt, (x * Math.PI) / 180, (y * Math.PI) / 180);
				gl.drawArrays(gl.TRIANGLES, 0, 6);
			},
			destroy
		};
	} catch {
		destroy();
		return undefined;
	}
}
