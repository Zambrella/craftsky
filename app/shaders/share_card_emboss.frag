#include <flutter/runtime_effect.glsl>

uniform vec2 uSize;
uniform vec2 uTilt;
out vec4 fragColor;

float roundedRectDistance(vec2 point, vec2 halfSize, float radius) {
  vec2 corner = abs(point) - halfSize + vec2(radius);
  return length(max(corner, vec2(0.0))) +
      min(max(corner.x, corner.y), 0.0) - radius;
}

// Only the original fine embossed edge remains. The icon foil is painted
// separately beneath it; the centre of this effect is transparent.
void main() {
  vec2 centred = FlutterFragCoord().xy - uSize * 0.5;
  float edge = roundedRectDistance(
      centred, uSize * 0.5 - vec2(1.5), 20.5);
  float rim = 1.0 - smoothstep(0.6, 2.5, abs(edge));
  float bevel = 1.0 - smoothstep(2.5, 6.0, -edge);
  float angle = -0.8 + (uTilt.x - uTilt.y) * 6.0;
  vec2 direction = vec2(cos(angle), sin(angle));
  vec2 normal = centred / max(length(centred), 0.001);
  float reflection = pow(max(dot(normal, direction), 0.0), 20.0);
  float secondary = pow(max(dot(normal, -direction), 0.0), 28.0) * 0.4;
  float glint = min(reflection + secondary, 1.0);
  vec3 cobalt = vec3(21.0, 53.0, 214.0) / 255.0;
  vec3 paper = vec3(245.0, 239.0, 228.0) / 255.0;
  vec3 silver = mix(vec3(155.0, 194.0, 230.0) / 255.0, paper, glint);
  float rimAlpha = rim * (0.22 + glint * 0.76);
  float bevelAlpha = bevel * 0.24 * (1.0 - rimAlpha);
  // Flutter runtime effects require premultiplied colour.
  fragColor = vec4(silver * rimAlpha + cobalt * 0.65 * bevelAlpha,
      rimAlpha + bevelAlpha);
}
