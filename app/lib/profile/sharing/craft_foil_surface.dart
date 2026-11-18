import 'dart:async';
import 'dart:ui' as ui;

import 'package:craftsky_app/profile/data/crafts_catalog.dart';
import 'package:craftsky_app/profile/sharing/share_card_motion.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_svg/flutter_svg.dart';

/// Repeated craft glyphs act as a foil mask. A tilt-controlled light affects
/// only that mask, beneath the card's portrait, identity and QR code.
class CraftFoilSurface extends StatefulWidget {
  const new({this.frames, super.key});

  final ValueListenable<ShareCardFrame>? frames;

  /// Ensure the complete decoration is available before an image capture.
  static Future<void> prepareAssets() async {
    await Future.wait([
      _CraftFoilSurfaceState.patternAsset,
      _CraftFoilSurfaceState.edgeAsset,
    ]);
  }

  @override
  State<CraftFoilSurface> createState() => _CraftFoilSurfaceState();
}

class _CraftFoilSurfaceState extends State<CraftFoilSurface> {
  static Future<ui.FragmentProgram?>? _edgeProgram;
  static Future<ui.Picture?>? _patternAsset;
  // A single bounded mask is shared for the app lifetime, like the shader
  // program. Opening more cards does not allocate more copies of the pattern.
  static Future<ui.Picture?> get patternAsset =>
      _patternAsset ??= _buildPattern();
  static Future<ui.FragmentProgram?> get edgeAsset =>
      _edgeProgram ??= _loadEdgeProgram();
  ui.FragmentShader? _edgeShader;
  ui.Picture? _pattern;

  @override
  void initState() {
    super.initState();
    unawaited(_loadEdge());
    unawaited(_loadPattern());
  }

  Future<void> _loadEdge() async {
    final program = await edgeAsset;
    if (mounted && program != null) {
      setState(() => _edgeShader = program.fragmentShader());
    }
  }

  static Future<ui.FragmentProgram?> _loadEdgeProgram() async {
    try {
      return await ui.FragmentProgram.fromAsset(
        'shaders/share_card_emboss.frag',
      );
    } on Object {
      // A static hairline remains on backends without runtime effects.
      return null;
    }
  }

  Future<void> _loadPattern() async {
    final pattern = await patternAsset;
    if (mounted) setState(() => _pattern = pattern);
  }

  static Future<ui.Picture?> _buildPattern() async {
    final icons = <PictureInfo>[];
    try {
      for (final craft in canonicalSelectableCrafts) {
        final path = CraftIcon.assetPathFor(craft.id);
        if (path != null) {
          icons.add(await vg.loadPicture(SvgAssetLoader(path), null));
        }
      }
      if (icons.isEmpty) return null;
      final recorder = ui.PictureRecorder();
      final canvas = Canvas(recorder);
      // A staggered, deterministic sequence. Record the vectors once rather
      // than rebuilding dozens of SVG widgets on each motion frame.
      for (var row = 0; row < 11; row++) {
        for (var column = 0; column < 9; column++) {
          final icon = icons[(column + row * 2) % icons.length];
          final x = column * 68.0 - 24 + (row.isOdd ? 34 : 0);
          final y = row * 64.0 - 16;
          canvas
            ..save()
            ..translate(x, y)
            ..scale(28 / icon.size.width, 28 / icon.size.height)
            ..drawPicture(icon.picture)
            ..restore();
        }
      }
      return recorder.endRecording();
    } on Object {
      // Decoration is optional: asset/codec failure leaves a usable card.
      return null;
    } finally {
      for (final icon in icons) {
        icon.picture.dispose();
      }
    }
  }

  @override
  void dispose() {
    _edgeShader?.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => ExcludeSemantics(
    child: IgnorePointer(
      child: CustomPaint(
        key: const Key('share-card-craft-foil'),
        painter: _FoilPainter(
          edgeShader: _edgeShader,
          pattern: _pattern,
          frames: widget.frames,
        ),
      ),
    ),
  );
}

class _FoilPainter extends CustomPainter {
  new({required this.edgeShader, required this.pattern, required this.frames})
    : super(repaint: frames);

  final ui.FragmentShader? edgeShader;
  final ui.Picture? pattern;
  final ValueListenable<ShareCardFrame>? frames;

  @override
  void paint(Canvas canvas, Size size) {
    final bounds = Offset.zero & size;
    final tilt = frames?.value.tilt ?? Offset.zero;
    canvas.drawColor(BrandColors.cobalt, BlendMode.src);
    if (pattern case final mask?) {
      final light = Offset(
        size.width * (0.28 - tilt.dy * 12),
        size.height * (0.25 + tilt.dx * 12),
      );
      // The low ambient level keeps the sequence faint; a soft specular spot
      // reveals the individual glyphs as the card turns towards the light.
      final foil = Paint()
        ..blendMode = BlendMode.srcIn
        ..shader = ui.Gradient.radial(
          light,
          size.shortestSide * 0.62,
          [
            BrandColors.paper.withValues(alpha: 0.32),
            BrandColors.sky.withValues(alpha: 0.16),
            BrandColors.lilac.withValues(alpha: 0.06),
            BrandColors.paper.withValues(alpha: 0.028),
          ],
          [0, 0.22, 0.62, 1],
        );
      canvas
        ..saveLayer(bounds, Paint())
        ..drawPicture(mask)
        ..drawRect(bounds, foil)
        ..restore();
    }
    if (edgeShader case final effect?) {
      effect
        ..setFloat(0, size.width)
        ..setFloat(1, size.height)
        ..setFloat(2, tilt.dx)
        ..setFloat(3, tilt.dy);
      canvas.drawRect(bounds, Paint()..shader = effect);
    } else {
      canvas.drawRRect(
        RRect.fromRectAndRadius(
          bounds.deflate(1.5),
          const Radius.circular(20.5),
        ),
        Paint()
          ..color = BrandColors.paper.withValues(alpha: 0.18)
          ..style = PaintingStyle.stroke
          ..strokeWidth = 1.5,
      );
    }
  }

  @override
  bool shouldRepaint(_FoilPainter oldDelegate) =>
      oldDelegate.edgeShader != edgeShader ||
      oldDelegate.pattern != pattern ||
      oldDelegate.frames != frames;
}
