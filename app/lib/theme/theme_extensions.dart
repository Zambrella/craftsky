import 'dart:ui' show lerpDouble;

import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:flutter/material.dart';

/// 8-point grid with a 4-point half-step — `--sp-1` through `--sp-9` in the
/// CraftSky design system.
class SpacingTheme extends ThemeExtension<SpacingTheme> {
  const SpacingTheme({
    this.sp1 = 4,
    this.sp2 = 8,
    this.sp3 = 12,
    this.sp4 = 16,
    this.sp5 = 24,
    this.sp6 = 32,
    this.sp7 = 48,
    this.sp8 = 64,
    this.sp9 = 96,
  });

  final double sp1;
  final double sp2;
  final double sp3;
  final double sp4;
  final double sp5;
  final double sp6;
  final double sp7;
  final double sp8;
  final double sp9;

  @override
  SpacingTheme copyWith({
    double? sp1,
    double? sp2,
    double? sp3,
    double? sp4,
    double? sp5,
    double? sp6,
    double? sp7,
    double? sp8,
    double? sp9,
  }) {
    return SpacingTheme(
      sp1: sp1 ?? this.sp1,
      sp2: sp2 ?? this.sp2,
      sp3: sp3 ?? this.sp3,
      sp4: sp4 ?? this.sp4,
      sp5: sp5 ?? this.sp5,
      sp6: sp6 ?? this.sp6,
      sp7: sp7 ?? this.sp7,
      sp8: sp8 ?? this.sp8,
      sp9: sp9 ?? this.sp9,
    );
  }

  @override
  SpacingTheme lerp(ThemeExtension<SpacingTheme>? other, double t) {
    if (other is! SpacingTheme) return this;
    return SpacingTheme(
      sp1: lerpDouble(sp1, other.sp1, t)!,
      sp2: lerpDouble(sp2, other.sp2, t)!,
      sp3: lerpDouble(sp3, other.sp3, t)!,
      sp4: lerpDouble(sp4, other.sp4, t)!,
      sp5: lerpDouble(sp5, other.sp5, t)!,
      sp6: lerpDouble(sp6, other.sp6, t)!,
      sp7: lerpDouble(sp7, other.sp7, t)!,
      sp8: lerpDouble(sp8, other.sp8, t)!,
      sp9: lerpDouble(sp9, other.sp9, t)!,
    );
  }
}

/// Corner radii — mostly square, chunky-rounded for signature moments.
/// `r0` = full-bleed photos, `r4` = statement buttons, `rPill` = chips/avatars.
class RadiusTheme extends ThemeExtension<RadiusTheme> {
  const RadiusTheme({
    this.r0 = 0,
    this.r1 = 2,
    this.r2 = 6,
    this.r3 = 14,
    this.r4 = 22,
    this.rPill = 999,
  });

  final double r0;
  final double r1;
  final double r2;
  final double r3;
  final double r4;
  final double rPill;

  @override
  RadiusTheme copyWith({
    double? r0,
    double? r1,
    double? r2,
    double? r3,
    double? r4,
    double? rPill,
  }) {
    return RadiusTheme(
      r0: r0 ?? this.r0,
      r1: r1 ?? this.r1,
      r2: r2 ?? this.r2,
      r3: r3 ?? this.r3,
      r4: r4 ?? this.r4,
      rPill: rPill ?? this.rPill,
    );
  }

  @override
  RadiusTheme lerp(ThemeExtension<RadiusTheme>? other, double t) {
    if (other is! RadiusTheme) return this;
    return RadiusTheme(
      r0: lerpDouble(r0, other.r0, t)!,
      r1: lerpDouble(r1, other.r1, t)!,
      r2: lerpDouble(r2, other.r2, t)!,
      r3: lerpDouble(r3, other.r3, t)!,
      r4: lerpDouble(r4, other.r4, t)!,
      rPill: lerpDouble(rPill, other.rPill, t)!,
    );
  }
}

/// Motion durations from the design system.
/// Default ease is out; `ease-pop` is a springy bounce for buttons and likes.
class DurationTheme extends ThemeExtension<DurationTheme> {
  const DurationTheme({
    this.fast = const Duration(milliseconds: 120),
    this.medium = const Duration(milliseconds: 220),
    this.modal = const Duration(milliseconds: 320),
    this.ease = const Cubic(0.22, 0.61, 0.36, 1),
    this.easePop = const Cubic(0.34, 1.56, 0.64, 1),
  });

  final Duration fast;
  final Duration medium;
  final Duration modal;
  final Curve ease;
  final Curve easePop;

  @override
  DurationTheme copyWith({
    Duration? fast,
    Duration? medium,
    Duration? modal,
    Curve? ease,
    Curve? easePop,
  }) {
    return DurationTheme(
      fast: fast ?? this.fast,
      medium: medium ?? this.medium,
      modal: modal ?? this.modal,
      ease: ease ?? this.ease,
      easePop: easePop ?? this.easePop,
    );
  }

  @override
  DurationTheme lerp(ThemeExtension<DurationTheme>? other, double t) {
    if (other is! DurationTheme) return this;
    // Durations and curves don't interpolate meaningfully; snap at midpoint.
    return t < 0.5 ? this : other;
  }
}

/// Hard-offset "paper-on-paper" drop shadows — the signature CraftSky move.
/// `sm` = buttons/small chips, `md` = cards/hero elements, `lg` = posters.
class BrandShadowTheme extends ThemeExtension<BrandShadowTheme> {
  const BrandShadowTheme({
    this.drop = const [BoxShadow(color: BrandColors.ink, offset: Offset(6, 6))],
    this.dropSm = const [
      BoxShadow(color: BrandColors.ink, offset: Offset(3, 3)),
    ],
    this.dropLg = const [
      BoxShadow(color: BrandColors.ink, offset: Offset(10, 10)),
    ],
    this.paper1 = const [
      BoxShadow(color: Color(0x0A161210), offset: Offset(0, 2)),
      BoxShadow(
        color: Color(0x14161210),
        offset: Offset(0, 8),
        blurRadius: 20,
      ),
    ],
    this.paper2 = const [
      BoxShadow(
        color: Color(0x29161210),
        offset: Offset(0, 20),
        blurRadius: 40,
      ),
    ],
  });

  final List<BoxShadow> drop;
  final List<BoxShadow> dropSm;
  final List<BoxShadow> dropLg;
  final List<BoxShadow> paper1;
  final List<BoxShadow> paper2;

  @override
  BrandShadowTheme copyWith({
    List<BoxShadow>? drop,
    List<BoxShadow>? dropSm,
    List<BoxShadow>? dropLg,
    List<BoxShadow>? paper1,
    List<BoxShadow>? paper2,
  }) {
    return BrandShadowTheme(
      drop: drop ?? this.drop,
      dropSm: dropSm ?? this.dropSm,
      dropLg: dropLg ?? this.dropLg,
      paper1: paper1 ?? this.paper1,
      paper2: paper2 ?? this.paper2,
    );
  }

  @override
  BrandShadowTheme lerp(ThemeExtension<BrandShadowTheme>? other, double t) {
    if (other is! BrandShadowTheme) return this;
    return t < 0.5 ? this : other;
  }
}

/// Supporting paper swatches — used as coloured cutout backgrounds behind
/// imagery, chips, and large surface variety. Never text colour.
class BrandSwatchTheme extends ThemeExtension<BrandSwatchTheme> {
  const BrandSwatchTheme({
    this.paper = BrandColors.paper,
    this.paper2 = BrandColors.paper2,
    this.paper3 = BrandColors.paper3,
    this.butter = BrandColors.butter,
    this.clay = BrandColors.clay,
    this.moss = BrandColors.moss,
    this.sky = BrandColors.sky,
    this.lilac = BrandColors.lilac,
    this.wip = BrandColors.butter,
    this.done = BrandColors.moss,
    this.like = BrandColors.red,
    this.borderHair = BrandColors.borderHair,
  });

  final Color paper;
  final Color paper2;
  final Color paper3;

  final Color butter;
  final Color clay;
  final Color moss;
  final Color sky;
  final Color lilac;

  final Color wip;
  final Color done;
  final Color like;

  final Color borderHair;

  @override
  BrandSwatchTheme copyWith({
    Color? paper,
    Color? paper2,
    Color? paper3,
    Color? butter,
    Color? clay,
    Color? moss,
    Color? sky,
    Color? lilac,
    Color? wip,
    Color? done,
    Color? like,
    Color? borderHair,
  }) {
    return BrandSwatchTheme(
      paper: paper ?? this.paper,
      paper2: paper2 ?? this.paper2,
      paper3: paper3 ?? this.paper3,
      butter: butter ?? this.butter,
      clay: clay ?? this.clay,
      moss: moss ?? this.moss,
      sky: sky ?? this.sky,
      lilac: lilac ?? this.lilac,
      wip: wip ?? this.wip,
      done: done ?? this.done,
      like: like ?? this.like,
      borderHair: borderHair ?? this.borderHair,
    );
  }

  @override
  BrandSwatchTheme lerp(ThemeExtension<BrandSwatchTheme>? other, double t) {
    if (other is! BrandSwatchTheme) return this;
    return BrandSwatchTheme(
      paper: Color.lerp(paper, other.paper, t)!,
      paper2: Color.lerp(paper2, other.paper2, t)!,
      paper3: Color.lerp(paper3, other.paper3, t)!,
      butter: Color.lerp(butter, other.butter, t)!,
      clay: Color.lerp(clay, other.clay, t)!,
      moss: Color.lerp(moss, other.moss, t)!,
      sky: Color.lerp(sky, other.sky, t)!,
      lilac: Color.lerp(lilac, other.lilac, t)!,
      wip: Color.lerp(wip, other.wip, t)!,
      done: Color.lerp(done, other.done, t)!,
      like: Color.lerp(like, other.like, t)!,
      borderHair: Color.lerp(borderHair, other.borderHair, t)!,
    );
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
  SemanticColorsTheme copyWith({
    Color? error,
    Color? warning,
    Color? success,
    Color? info,
  }) {
    return SemanticColorsTheme(
      error: error ?? this.error,
      warning: warning ?? this.warning,
      success: success ?? this.success,
      info: info ?? this.info,
    );
  }

  @override
  SemanticColorsTheme lerp(
    ThemeExtension<SemanticColorsTheme>? other,
    double t,
  ) {
    if (other is! SemanticColorsTheme) return this;
    return SemanticColorsTheme(
      error: Color.lerp(error, other.error, t)!,
      warning: Color.lerp(warning, other.warning, t)!,
      success: Color.lerp(success, other.success, t)!,
      info: Color.lerp(info, other.info, t)!,
    );
  }
}
