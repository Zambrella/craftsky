import 'dart:async';
import 'dart:math' as math;
import 'dart:ui' as ui;

import 'package:flutter/widgets.dart';
import 'package:visibility_detector/visibility_detector.dart';

/// A cached fragment program, with a separate shader instance for each header.
/// Only this decoration repaints; profile text and controls stay outside it.
class ProfileShaderBackground extends StatefulWidget {
  const new({required this.style, super.key});

  final int style;

  /// Preload before displaying a profile or taking a deterministic capture.
  static Future<void> prepareAssets() async {
    await (_ProfileShaderBackgroundState._program ??=
        _ProfileShaderBackgroundState._loadProgram());
  }

  @override
  State<ProfileShaderBackground> createState() =>
      _ProfileShaderBackgroundState();
}

class _ProfileShaderBackgroundState extends State<ProfileShaderBackground>
    with SingleTickerProviderStateMixin, WidgetsBindingObserver {
  static Future<ui.FragmentProgram?>? _program;
  static ui.FragmentProgram? _loadedProgram;
  late final AnimationController _clock = AnimationController(
    vsync: this,
    duration: const Duration(seconds: 24),
  );
  final _visibilityKey = UniqueKey();
  ui.FragmentShader? _shader;
  bool _visible = true;
  bool _foreground = true;
  bool _motionAllowed = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _foreground =
        WidgetsBinding.instance.lifecycleState == null ||
        WidgetsBinding.instance.lifecycleState == AppLifecycleState.resumed;
    if (_loadedProgram case final program?) {
      _shader = program.fragmentShader();
    } else {
      unawaited(_load());
    }
  }

  static Future<ui.FragmentProgram?> _loadProgram() async {
    try {
      return _loadedProgram = await ui.FragmentProgram.fromAsset(
        'shaders/profile_background.frag',
      );
    } on Object {
      // Optional decoration: unsupported backends keep the static shading.
      return null;
    }
  }

  Future<void> _load() async {
    final program = await (_program ??= _loadProgram());
    if (!mounted || program == null) return;
    setState(() => _shader = program.fragmentShader());
    _syncMotion();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _motionAllowed =
        !MediaQuery.disableAnimationsOf(context) &&
        !MediaQuery.accessibleNavigationOf(context) &&
        TickerMode.valuesOf(context).enabled &&
        (ModalRoute.isCurrentOf(context) ?? true);
    _syncMotion();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _foreground = state == AppLifecycleState.resumed;
    _syncMotion();
  }

  void _syncMotion() {
    if (!mounted) return;
    if (_shader != null && _motionAllowed && _foreground && _visible) {
      if (!_clock.isAnimating) _clock.repeat();
    } else {
      _clock.stop();
      if (!_motionAllowed) _clock.value = 0;
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _clock.dispose();
    _shader?.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => ExcludeSemantics(
    child: IgnorePointer(
      child: VisibilityDetector(
        key: _visibilityKey,
        onVisibilityChanged: (info) {
          _visible = info.visibleFraction > 0;
          _syncMotion();
        },
        child: RepaintBoundary(
          child: CustomPaint(
            painter: _ProfileShaderPainter(
              shader: _shader,
              clock: _clock,
              style: widget.style,
            ),
          ),
        ),
      ),
    ),
  );
}

class _ProfileShaderPainter extends CustomPainter {
  new({required this.shader, required this.clock, required this.style})
    : super(repaint: clock);

  final ui.FragmentShader? shader;
  final Animation<double> clock;
  final int style;

  @override
  void paint(Canvas canvas, Size size) {
    if (size.isEmpty) return;
    final bounds = Offset.zero & size;
    if (shader case final effect?) {
      effect
        ..setFloat(0, size.width)
        ..setFloat(1, size.height)
        ..setFloat(2, clock.value * math.pi * 2)
        ..setFloat(3, style.toDouble());
      canvas.drawRect(bounds, Paint()..shader = effect);
    } else {
      canvas.drawRect(
        bounds,
        Paint()
          ..shader = ui.Gradient.linear(
            Offset.zero,
            Offset(size.width, size.height),
            const [Color(0x05000000), Color(0x33000000)],
          ),
      );
    }
  }

  @override
  bool shouldRepaint(_ProfileShaderPainter oldDelegate) =>
      oldDelegate.shader != shader || oldDelegate.style != style;
}
