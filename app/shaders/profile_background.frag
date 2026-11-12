#include <flutter/runtime_effect.glsl>

uniform vec2 uSize;
uniform float uPhase;
uniform float uStyle;
out vec4 fragColor;

// Slow, seamless 24-second compositions. Dark, translucent pigments
// retain the chosen profile colour and readable light header controls.
// Stationary grain avoids temporal sparkle while the colour moves beneath it.
float grainAt(vec2 p) {
  return fract(sin(dot(floor(p), vec2(127.1, 311.7))) * 43758.5453);
}

float softNoise(vec2 p) {
  vec2 cell = floor(p);
  vec2 f = fract(p);
  f = f * f * (3.0 - 2.0 * f);
  return mix(mix(grainAt(cell), grainAt(cell + vec2(1.0, 0.0)), f.x),
      mix(grainAt(cell + vec2(0.0, 1.0)), grainAt(cell + vec2(1.0)), f.x), f.y);
}

void main() {
  vec2 uv = FlutterFragCoord().xy / max(uSize, vec2(1.0));
  if (uStyle < 0.5) {
    // Aurora: wide, overlapping teal and violet ribbons, softly undulating.
    float upper = uv.y - (0.24 + 0.18 * sin(uv.x * 3.5 + uPhase));
    float lower = uv.y - (0.74 + 0.16 * cos(uv.x * 2.8 - uPhase));
    float teal = exp(-upper * upper * 15.0);
    float violet = exp(-lower * lower * 13.0);
    float drift = 0.5 + 0.5 * sin(uv.x * 3.0 + uv.y * 2.0 + uPhase);
    vec3 cool = mix(vec3(0.015, 0.10, 0.25),
        vec3(0.015, 0.24, 0.18), drift);
    vec3 tint = mix(cool, vec3(0.24, 0.035, 0.30),
        violet / max(teal + violet, 0.001));
    float alpha = 0.28 + 0.40 * max(teal, violet);
    fragColor = vec4(tint * alpha, alpha);
  } else if (uStyle < 1.5) {
    // Two drifting pools of cool and warm dye, mixed over the selected colour.
    vec2 a = uv - vec2(0.28 + 0.20 * sin(uPhase),
        0.35 + 0.20 * cos(uPhase));
    vec2 b = uv - vec2(0.72 + 0.16 * cos(uPhase),
        0.65 + 0.22 * sin(uPhase));
    float cool = exp(-dot(a, a) * 8.0);
    float warm = exp(-dot(b, b) * 9.0);
    float pigment = warm / max(cool + warm, 0.001);
    vec3 tint = mix(vec3(0.015, 0.12, 0.22),
        vec3(0.30, 0.035, 0.15), pigment);
    float alpha = 0.25 + 0.42 * max(cool, warm);
    fragColor = vec4(tint * alpha, alpha);
  } else if (uStyle < 2.5) {
    // Sunset: a slow mesh of copper, berry and indigo, with three moving centres.
    vec2 a = uv - vec2(0.18 + 0.15 * cos(uPhase),
        0.28 + 0.20 * sin(uPhase));
    vec2 b = uv - vec2(0.70 + 0.18 * sin(uPhase),
        0.24 + 0.16 * cos(uPhase));
    vec2 c = uv - vec2(0.48 + 0.24 * cos(uPhase),
        0.80 + 0.12 * sin(uPhase));
    float copper = exp(-dot(a, a) * 5.0);
    float berry = exp(-dot(b, b) * 6.0);
    float indigo = exp(-dot(c, c) * 5.5);
    float total = max(copper + berry + indigo, 0.001);
    vec3 tint = (vec3(0.44, 0.13, 0.025) * copper
        + vec3(0.30, 0.025, 0.14) * berry
        + vec3(0.045, 0.035, 0.19) * indigo) / total;
    float alpha = 0.32 + 0.40 * max(copper, max(berry, indigo));
    fragColor = vec4(tint * alpha, alpha);
  } else if (uStyle < 3.5) {
    // Pigment: saturated dye clouds on a fine, stationary paper grain.
    vec2 p = uv * vec2(3.2, 2.3);
    p += vec2(0.35 * sin(uPhase), 0.30 * cos(uPhase));
    float cloud = softNoise(p + vec2(softNoise(p + vec2(2.7)), 0.0));
    float mixValue = smoothstep(0.28, 0.72, cloud);
    vec3 tint = mix(vec3(0.005, 0.17, 0.21),
        vec3(0.36, 0.025, 0.15), mixValue);
    float grain = grainAt(FlutterFragCoord().xy / 1.25) - 0.5;
    float alpha = clamp(0.18 + 0.66 * mixValue + grain * 0.10, 0.12, 0.89);
    fragColor = vec4(tint * alpha, alpha);
  } else if (uStyle < 4.5) {
    // Marble: flowing coloured contours, deep veins and a little surface grain.
    vec2 p = uv * vec2(2.8, 2.0)
        + vec2(0.28 * sin(uPhase), 0.28 * cos(uPhase));
    float warp = softNoise(p);
    float field = uv.x * 8.0 + uv.y * 4.0 + warp * 7.0 + uPhase;
    float band = 0.5 + 0.5 * sin(field);
    float vein = 1.0 - smoothstep(0.025, 0.14, abs(sin(field * 2.0)));
    vec3 tint = mix(vec3(0.035, 0.19, 0.22),
        vec3(0.30, 0.035, 0.25), band);
    tint = mix(tint, vec3(0.015, 0.01, 0.035), vein * 0.8);
    float grain = grainAt(FlutterFragCoord().xy / 1.4) - 0.5;
    float alpha = clamp(0.20 + 0.48 * band + 0.20 * vein + grain * 0.045,
        0.16, 0.91);
    fragColor = vec4(tint * alpha, alpha);
  } else {
    // Woven: the gradient drifts behind a fixed, softly crossed textile weave.
    float field = 0.5 + 0.5 * sin(uv.x * 4.0 + uv.y * 2.0
        + 1.2 * sin(uv.y * 3.0 - uPhase) + uPhase);
    vec3 tint = mix(vec3(0.015, 0.15, 0.20),
        vec3(0.34, 0.095, 0.055), field);
    vec2 fibres = FlutterFragCoord().xy / 3.5;
    float warp = 0.5 + 0.5 * sin(fibres.x * 6.2831853);
    float weft = 0.5 + 0.5 * sin(fibres.y * 6.2831853);
    float weave = warp * weft;
    float grain = grainAt(FlutterFragCoord().xy) - 0.5;
    float alpha = clamp(0.16 + 0.52 * field + 0.14 * weave + grain * 0.025,
        0.14, 0.85);
    fragColor = vec4(tint * alpha, alpha);
  }
}
