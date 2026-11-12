#include <flutter/runtime_effect.glsl>

uniform vec2 uSize;
uniform float uPhase;
uniform float uStyle;
uniform vec3 uAccent;
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

// Reference palettes follow the selected accent's hue; the original six
// continue to use their existing translucent pigments without any changes.
float square(float x) { return x * x; }

vec3 hsvColour(float h, float s, float v) {
  vec3 rgb = clamp(abs(fract(vec3(h) + vec3(0.0, 0.6666667, 0.3333333))
      * 6.0 - 3.0) - 1.0, 0.0, 1.0);
  return v * mix(vec3(1.0), rgb, s);
}

float selectedHue() {
  float hi = max(uAccent.r, max(uAccent.g, uAccent.b));
  float lo = min(uAccent.r, min(uAccent.g, uAccent.b));
  float d = hi - lo;
  if (d < 0.001) return 0.63;
  float h;
  if (hi == uAccent.r) h = (uAccent.g - uAccent.b) / d;
  else if (hi == uAccent.g) h = 2.0 + (uAccent.b - uAccent.r) / d;
  else h = 4.0 + (uAccent.r - uAccent.g) / d;
  return fract(h / 6.0 + 1.0);
}

vec3 referenceStyle(vec2 uv) {
  float h = selectedHue();
  vec3 deep = hsvColour(h - 0.025, 0.92, 0.30);
  vec3 blue = hsvColour(h - 0.015, 0.96, 0.65);
  vec3 cyan = hsvColour(h - 0.12, 0.94, 0.82);
  vec3 ice = hsvColour(h - 0.14, 0.25, 1.0);
  float grain = grainAt(FlutterFragCoord().xy) - 0.5;
  if (uStyle < 6.5) {
    // Ribbon bands: concentric, off-centre arcs with clean poster-like edges.
    vec2 centre = vec2(-0.22 + 0.07 * sin(uPhase),
        1.18 + 0.07 * cos(uPhase));
    float radius = length((uv - centre) * vec2(1.0, 0.82));
    float bands = radius * 9.0 + 0.35 * sin(uPhase);
    float index = mod(floor(bands), 5.0);
    vec3 colour = blue;
    if (index < 0.5) colour = deep;
    else if (index < 1.5) colour = cyan;
    else if (index < 2.5) colour = mix(cyan, ice, 0.48);
    else if (index < 3.5) colour = ice;
    // A narrow edge blend keeps arcs crisp without one-pixel stair stepping.
    float edge = smoothstep(0.0, 0.018, fract(bands));
    return mix(mix(blue, cyan, 0.35), colour, edge);
  } else if (uStyle < 7.5) {
    // Liquid silk: a broad translucent fold with smaller wisps along its seam.
    float curve = 0.78 - uv.y * 0.55
        + 0.12 * sin(uv.y * 5.0 + uPhase)
        + 0.07 * sin(uv.x * 4.0 + uv.y * 6.0 - uPhase);
    float fold = uv.x - curve;
    float veil = exp(-fold * fold * 9.0);
    float seam = exp(-fold * fold * 95.0);
    float softLight = smoothstep(-0.45, 0.45, fold);
    vec3 colour = mix(deep, cyan, 0.12 + softLight * 0.80);
    colour = mix(colour, ice, clamp(veil * 0.68 + seam * 0.25, 0.0, 0.96));
    float crease = exp(-square((fold + 0.10) * 15.0));
    colour *= 1.0 - crease * 0.34;
    float reflection = exp(-square((fold + 0.20) * 11.0));
    colour = mix(colour, ice, reflection * 0.28);
    vec2 hazeAt = uv - vec2(0.84, 0.02);
    float haze = exp(-dot(hazeAt, hazeAt) * 3.2);
    return mix(colour, ice, haze * 0.46);
  } else if (uStyle < 8.5) {
    // Light bloom: a defocused crescent with a pale, almost white foot.
    float curve = uv.x - (0.12 + 0.23 * sin(uv.y * 3.8 + 0.35 * sin(uPhase)));
    float glow = exp(-curve * curve * 28.0);
    vec2 centre = (uv - vec2(0.66 + 0.07 * sin(uPhase), 0.40)) * vec2(1.2, 1.0);
    float hollow = exp(-dot(centre, centre) * 5.0);
    vec3 colour = mix(deep * 0.8, cyan * 0.72, 0.16 + 0.15 * sin(uv.x * 3.0));
    colour = mix(colour, ice, clamp(glow * (1.0 - hollow * 0.60) * 1.10, 0.0, 1.0));
    float foot = exp(-dot((uv - vec2(0.20, 0.98)) * vec2(2.0, 1.3),
        (uv - vec2(0.20, 0.98)) * vec2(2.0, 1.3)) * 5.0);
    return mix(colour, vec3(0.98, 0.99, 0.91), foot * 0.80);
  } else if (uStyle < 9.5) {
    // Sunbeams: warm, soft rays crossing a cool field from the top right.
    vec2 p = (uv - vec2(1.06, -0.10)) * vec2(1.0, 0.80);
    float angle = atan(p.y, p.x);
    float rays = 0.0;
    for (int i = 0; i < 5; i++) {
      float f = float(i);
      float direction = 1.82 + f * 0.255 + 0.025 * sin(uPhase + f);
      float width = 0.024 + f * 0.006;
      float delta = (angle - direction) / width;
      rays += exp(-delta * delta) * (0.62 + 0.12 * sin(uPhase + f));
    }
    float distanceFromLight = length(p);
    float light = rays * 0.65 * exp(-distanceFromLight * 0.65)
        + exp(-distanceFromLight * distanceFromLight * 3.5) * 0.55;
    vec3 base = hsvColour(h - 0.11, 0.48, 0.62);
    vec3 peach = hsvColour(h + 0.43, 0.35, 1.0);
    return mix(base, peach, clamp(light, 0.0, 0.94));
  } else if (uStyle < 10.5) {
    // Lagoon: saturated cyan and deep blue clouds separated by a grainy fold.
    float boundary = uv.x + uv.y * 0.45 - 0.60
        + 0.18 * sin(uv.y * 3.5 + uPhase);
    float pool = smoothstep(-0.22, 0.32, boundary);
    vec3 colour = mix(deep * 0.60, cyan, pool);
    vec2 glowAt = uv - vec2(0.75 + 0.08 * sin(uPhase), 0.05);
    float glow = exp(-dot(glowAt, glowAt) * 5.0);
    return mix(colour, ice, glow * 0.90) + grain * 0.085;
  } else if (uStyle < 11.5) {
    // Nebula: dark centre, with quiet violet and teal light at the edges.
    vec3 night = hsvColour(h, 0.64, 0.095);
    vec2 a = (uv - vec2(0.82, -0.18 + 0.06 * sin(uPhase))) * vec2(1.3, 1.9);
    vec2 b = (uv - vec2(1.10, 1.14 + 0.07 * cos(uPhase))) * vec2(1.5, 1.3);
    vec2 c = (uv - vec2(0.42, 1.25)) * vec2(2.2, 1.7);
    vec3 colour = night
        + hsvColour(h + 0.16, 0.62, 0.48) * exp(-dot(a, a) * 3.0)
        + cyan * exp(-dot(b, b) * 2.4) * 0.70
        + hsvColour(h - 0.25, 0.94, 0.29) * exp(-dot(c, c) * 3.2);
    return colour;
  } else {
    // Prism: two neon halos with pink fringes, surrounded by black and grain.
    vec2 a = (uv - vec2(-0.06 + 0.06 * sin(uPhase), 1.10)) * vec2(0.95, 1.1);
    vec2 b = (uv - vec2(1.07 + 0.06 * cos(uPhase), -0.09)) * vec2(0.95, 1.1);
    float radius = min(length(a), length(b));
    float blueCore = 1.0 - smoothstep(0.12, 0.53, radius);
    float cyanRing = exp(-square((radius - 0.46) * 7.0));
    float pinkRing = exp(-square((radius - 0.65) * 9.0));
    vec3 colour = hsvColour(h + 0.02, 1.0, 1.0) * blueCore
        + hsvColour(h - 0.13, 1.0, 1.0) * cyanRing
        + hsvColour(h + 0.20, 0.68, 1.0) * pinkRing;
    float mask = max(blueCore, max(cyanRing, pinkRing));
    return colour + grain * (0.24 * mask + 0.008);
  }
}

void main() {
  vec2 uv = FlutterFragCoord().xy / max(uSize, vec2(1.0));
  if (uStyle >= 6.0) {
    fragColor = vec4(clamp(referenceStyle(uv), 0.0, 1.0), 1.0);
  } else if (uStyle < 0.5) {
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
