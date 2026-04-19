import 'dart:ui' show lerpDouble;

import 'package:flutter/material.dart';

class SpacingTheme extends ThemeExtension<SpacingTheme> {
  const SpacingTheme({
    this.xs = 4,
    this.s = 8,
    this.m = 16,
    this.l = 24,
    this.xl = 32,
  });

  final double xs;
  final double s;
  final double m;
  final double l;
  final double xl;

  @override
  SpacingTheme copyWith({double? xs, double? s, double? m, double? l, double? xl}) {
    return SpacingTheme(
      xs: xs ?? this.xs,
      s: s ?? this.s,
      m: m ?? this.m,
      l: l ?? this.l,
      xl: xl ?? this.xl,
    );
  }

  @override
  SpacingTheme lerp(ThemeExtension<SpacingTheme>? other, double t) {
    if (other is! SpacingTheme) return this;
    return SpacingTheme(
      xs: lerpDouble(xs, other.xs, t)!,
      s: lerpDouble(s, other.s, t)!,
      m: lerpDouble(m, other.m, t)!,
      l: lerpDouble(l, other.l, t)!,
      xl: lerpDouble(xl, other.xl, t)!,
    );
  }
}

class RadiusTheme extends ThemeExtension<RadiusTheme> {
  const RadiusTheme({
    this.small = 4,
    this.medium = 8,
    this.large = 16,
  });

  final double small;
  final double medium;
  final double large;

  @override
  RadiusTheme copyWith({double? small, double? medium, double? large}) {
    return RadiusTheme(
      small: small ?? this.small,
      medium: medium ?? this.medium,
      large: large ?? this.large,
    );
  }

  @override
  RadiusTheme lerp(ThemeExtension<RadiusTheme>? other, double t) {
    if (other is! RadiusTheme) return this;
    return RadiusTheme(
      small: lerpDouble(small, other.small, t)!,
      medium: lerpDouble(medium, other.medium, t)!,
      large: lerpDouble(large, other.large, t)!,
    );
  }
}

class DurationTheme extends ThemeExtension<DurationTheme> {
  const DurationTheme({
    this.fast = const Duration(milliseconds: 150),
    this.medium = const Duration(milliseconds: 300),
    this.slow = const Duration(milliseconds: 500),
  });

  final Duration fast;
  final Duration medium;
  final Duration slow;

  @override
  DurationTheme copyWith({Duration? fast, Duration? medium, Duration? slow}) {
    return DurationTheme(
      fast: fast ?? this.fast,
      medium: medium ?? this.medium,
      slow: slow ?? this.slow,
    );
  }

  @override
  DurationTheme lerp(ThemeExtension<DurationTheme>? other, double t) {
    // Durations don't interpolate meaningfully; snap at midpoint.
    if (other is! DurationTheme) return this;
    return t < 0.5 ? this : other;
  }
}

class SemanticColorsTheme extends ThemeExtension<SemanticColorsTheme> {
  const SemanticColorsTheme({
    required this.error,
    required this.warning,
    required this.success,
    required this.info,
  });

  final Color error;
  final Color warning;
  final Color success;
  final Color info;

  @override
  SemanticColorsTheme copyWith({Color? error, Color? warning, Color? success, Color? info}) {
    return SemanticColorsTheme(
      error: error ?? this.error,
      warning: warning ?? this.warning,
      success: success ?? this.success,
      info: info ?? this.info,
    );
  }

  @override
  SemanticColorsTheme lerp(ThemeExtension<SemanticColorsTheme>? other, double t) {
    if (other is! SemanticColorsTheme) return this;
    return SemanticColorsTheme(
      error: Color.lerp(error, other.error, t)!,
      warning: Color.lerp(warning, other.warning, t)!,
      success: Color.lerp(success, other.success, t)!,
      info: Color.lerp(info, other.info, t)!,
    );
  }
}
