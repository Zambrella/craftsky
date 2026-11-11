#include <flutter/runtime_effect.glsl>

uniform vec2 uSize;
uniform float uPhase;
uniform vec2 uTilt;
out vec4 fragColor;

// Embossed fibres catch a slow moving light. This paints only the cobalt
// paper behind the identity and QR; no sampling or distortion of content.
void main() {
  vec2 uv = FlutterFragCoord().xy / uSize;
  float wave = sin(uv.x * 8.0 + sin(uPhase) * 0.18) * 0.075;
  float thread = pow(0.5 + 0.5 * cos((uv.y - wave) * 310.0), 32.0);
  float band = smoothstep(0.04, 0.16, uv.y) *
      (1.0 - smoothstep(0.37, 0.55, uv.y));
  vec2 light = vec2(0.5 + sin(uPhase) * 0.45, 0.2) + uTilt * 3.0;
  float shine = exp(-dot(uv - light, uv - light) * 9.0);
  float strength = thread * band * (0.05 + shine * 0.13);
  vec3 cobalt = vec3(21.0, 53.0, 214.0) / 255.0;
  vec3 paper = vec3(245.0, 239.0, 228.0) / 255.0;
  fragColor = vec4(mix(cobalt, paper, strength), 1.0);
}
