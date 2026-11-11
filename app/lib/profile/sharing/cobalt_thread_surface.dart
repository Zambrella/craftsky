import 'dart:async';
import 'dart:math' as math;
import 'dart:ui' as ui;

import 'package:craftsky_app/profile/sharing/share_card_motion.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';

class CobaltThreadSurface extends StatefulWidget {
  const new({this.frames, super.key});

  final ValueListenable<ShareCardFrame>? frames;

  @override
  State<CobaltThreadSurface> createState() => _CobaltThreadSurfaceState();
}

class _CobaltThreadSurfaceState extends State<CobaltThreadSurface> {
  // Programs can be shared; each card owns and disposes its shader instance.
  static Future<ui.FragmentProgram?>? _program;
  ui.FragmentShader? _shader;

  @override
  void initState() {
    super.initState();
    unawaited(_load());
  }

  Future<void> _load() async {
    final program = await (_program ??= _loadProgram());
    if (mounted && program != null) {
      setState(() => _shader = program.fragmentShader());
    }
  }

  static Future<ui.FragmentProgram?> _loadProgram() async {
    try {
      return await ui.FragmentProgram.fromAsset(
        'shaders/share_card_threads.frag',
      );
    } on Object {
      // Backends without runtime effects retain the static embossed artwork.
      return null;
    }
  }

  @override
  void dispose() {
    _shader?.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => ExcludeSemantics(
    child: IgnorePointer(
      child: CustomPaint(
        painter: _ThreadsPainter(shader: _shader, frames: widget.frames),
      ),
    ),
  );
}

class _ThreadsPainter extends CustomPainter {
  new({required this.shader, required this.frames}) : super(repaint: frames);

  final ui.FragmentShader? shader;
  final ValueListenable<ShareCardFrame>? frames;

  @override
  void paint(Canvas canvas, Size size) {
    final effect = shader;
    final frame = frames?.value ?? const ShareCardFrame();
    if (effect != null) {
      effect
        ..setFloat(0, size.width)
        ..setFloat(1, size.height)
        ..setFloat(2, frame.phase)
        ..setFloat(3, frame.tilt.dx)
        ..setFloat(4, frame.tilt.dy);
      canvas.drawRect(Offset.zero & size, Paint()..shader = effect);
      return;
    }
    canvas.drawColor(BrandColors.cobalt, BlendMode.src);
    final paint = Paint()
      ..color = BrandColors.paper.withValues(alpha: 0.12)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 0.7;
    for (var thread = 0; thread < 8; thread++) {
      final path = Path();
      for (var x = 0.0; x <= size.width; x += 2) {
        final y =
            size.height * (0.13 + thread * 0.02) +
            math.sin(x / size.width * 8) * size.height * 0.075;
        if (x == 0) {
          path.moveTo(x, y);
        } else {
          path.lineTo(x, y);
        }
      }
      canvas.drawPath(path, paint);
    }
  }

  @override
  bool shouldRepaint(_ThreadsPainter oldDelegate) =>
      oldDelegate.shader != shader || oldDelegate.frames != frames;
}
