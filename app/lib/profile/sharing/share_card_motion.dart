import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/foundation.dart';
import 'package:flutter/scheduler.dart';
import 'package:flutter/widgets.dart';
import 'package:sensors_plus/sensors_plus.dart';

@immutable
class ShareCardFrame {
  const new({this.tilt = Offset.zero});

  final Offset tilt;
}

/// A small, damped rotation, never more than 5 degrees in either axis.
/// Only the open, foreground route subscribes to the sensor or runs a ticker.
class ShareCardMotion extends StatefulWidget {
  const new({
    required this.builder,
    this.enabled = true,
    this.gyroscope,
    super.key,
  });

  final Widget Function(
    BuildContext context,
    ValueListenable<ShareCardFrame> frames,
  )
  builder;
  final bool enabled;
  final Stream<GyroscopeEvent> Function()? gyroscope;

  @override
  State<ShareCardMotion> createState() => _ShareCardMotionState();
}

class _ShareCardMotionState extends State<ShareCardMotion>
    with SingleTickerProviderStateMixin, WidgetsBindingObserver {
  static const double _limit = 5 * math.pi / 180;
  final ValueNotifier<ShareCardFrame> _frames = ValueNotifier(
    const ShareCardFrame(),
  );
  late final Ticker _ticker = createTicker(_tick);
  StreamSubscription<GyroscopeEvent>? _subscription;
  DateTime? _lastSample;
  Duration _lastTick = Duration.zero;
  Offset _target = Offset.zero;
  bool _running = false;
  bool _foreground = true;
  bool _introPending = true;
  bool _introActive = false;
  bool _dragging = false;
  Offset _dragTilt = Offset.zero;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _foreground =
        WidgetsBinding.instance.lifecycleState == null ||
        WidgetsBinding.instance.lifecycleState == AppLifecycleState.resumed;
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _sync();
  }

  @override
  void didUpdateWidget(ShareCardMotion oldWidget) {
    super.didUpdateWidget(oldWidget);
    _sync();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _foreground = state == AppLifecycleState.resumed;
    _sync();
  }

  void _sync() {
    final shouldRun =
        widget.enabled &&
        _foreground &&
        !MediaQuery.disableAnimationsOf(context) &&
        !MediaQuery.accessibleNavigationOf(context) &&
        TickerMode.valuesOf(context).enabled &&
        (ModalRoute.isCurrentOf(context) ?? true);
    final playIntro = _introPending && shouldRun;
    _introPending = false;
    if (_running == shouldRun) return;
    _running = shouldRun;
    if (!shouldRun) {
      _ticker.stop();
      unawaited(_subscription?.cancel());
      _subscription = null;
      _lastSample = null;
      _target = Offset.zero;
      _introActive = false;
      _dragging = false;
      _dragTilt = Offset.zero;
      _frames.value = const ShareCardFrame();
      return;
    }
    _introActive = playIntro;
    _wake();
    if (widget.gyroscope != null ||
        defaultTargetPlatform == TargetPlatform.iOS ||
        defaultTargetPlatform == TargetPlatform.android) {
      try {
        _subscription =
            (widget.gyroscope?.call() ??
                    gyroscopeEventStream(
                      samplingPeriod: const Duration(milliseconds: 32),
                    ))
                .listen(
                  _sample,
                  // Sensor absence/denial is an expected static-tilt fallback.
                  onError: (Object _) => _target = Offset.zero,
                  cancelOnError: true,
                );
      } on Object {
        _target = Offset.zero;
      }
    }
  }

  void _sample(GyroscopeEvent event) {
    if (!_running || _dragging || !event.x.isFinite || !event.y.isFinite) {
      return;
    }
    final previous = _lastSample;
    _lastSample = event.timestamp;
    if (previous == null) return;
    final dt = (event.timestamp.difference(previous).inMicroseconds / 1e6)
        .clamp(0.0, 0.1);
    _target = Offset(
      (_target.dx + event.x * dt * 0.7).clamp(-_limit, _limit),
      (_target.dy + event.y * dt * 0.7).clamp(-_limit, _limit),
    );
    if (_target.distanceSquared > 1e-8) _wake();
  }

  void _wake() {
    if (_ticker.isActive) return;
    _lastTick = Duration.zero;
    _ticker.start();
  }

  void _startDrag(DragStartDetails _) {
    if (!_running) return;
    _introActive = false;
    _dragging = true;
    _dragTilt = _frames.value.tilt;
    _target = Offset.zero;
    _lastSample = null;
    _wake();
  }

  void _updateDrag(DragUpdateDetails details, Size size) {
    if (!_running || !_dragging || size.isEmpty) return;
    _dragTilt = Offset(
      (_dragTilt.dx - details.delta.dy / size.height * _limit * 4).clamp(
        -_limit,
        _limit,
      ),
      (_dragTilt.dy + details.delta.dx / size.width * _limit * 4).clamp(
        -_limit,
        _limit,
      ),
    );
    _wake();
  }

  void _endDrag() {
    if (!_dragging) return;
    _dragging = false;
    _dragTilt = Offset.zero;
    _lastSample = null;
    _wake();
  }

  void _tick(Duration elapsed) {
    final dt = ((elapsed - _lastTick).inMicroseconds / 1e6).clamp(0.0, 0.1);
    _lastTick = elapsed;
    // A short, two-axis invitation to interact; never repeated on resume or
    // after the native share sheet closes. Input takes over immediately.
    var intro = Offset.zero;
    if (_introActive) {
      final progress = (elapsed.inMicroseconds / 1500000).clamp(0.0, 1.0);
      final envelope = math.sin(math.pi * progress);
      intro = Offset(
        math.sin(progress * math.pi * 2) * envelope * _limit * 0.8,
        math.sin(progress * math.pi * 3) * envelope * _limit * 0.65,
      );
      if (progress >= 1) _introActive = false;
    }
    final desired = _dragging ? _dragTilt : _target + intro;
    final bounded = Offset(
      desired.dx.clamp(-_limit, _limit),
      desired.dy.clamp(-_limit, _limit),
    );
    final tilt = Offset.lerp(
      _frames.value.tilt,
      bounded,
      1 - math.exp(-dt * 9),
    )!;
    _target *= math.exp(-dt * 1.8);
    final settled = (tilt - bounded).distanceSquared < 1e-8;
    if (!_introActive &&
        settled &&
        (_dragging ||
            (tilt.distanceSquared < 1e-8 && _target.distanceSquared < 1e-8))) {
      _target = Offset.zero;
      _frames.value = ShareCardFrame(tilt: _dragging ? _dragTilt : Offset.zero);
      _ticker.stop();
      return;
    }
    _frames.value = ShareCardFrame(tilt: tilt);
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    unawaited(_subscription?.cancel());
    _ticker.dispose();
    _frames.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => ValueListenableBuilder(
    valueListenable: _frames,
    child: LayoutBuilder(
      builder: (gestureContext, _) => MouseRegion(
        cursor: _running ? SystemMouseCursors.grab : MouseCursor.defer,
        child: GestureDetector(
          behavior: HitTestBehavior.opaque,
          onPanStart: _running ? _startDrag : null,
          onPanUpdate: _running
              ? (details) {
                  final box = gestureContext.findRenderObject();
                  if (box is RenderBox && box.hasSize) {
                    _updateDrag(details, box.size);
                  }
                }
              : null,
          onPanEnd: _running ? (_) => _endDrag() : null,
          onPanCancel: _running ? _endDrag : null,
          child: widget.builder(context, _frames),
        ),
      ),
    ),
    builder: (_, frame, child) => Transform(
      key: const Key('share-card-tilt'),
      alignment: Alignment.center,
      transform: Matrix4.identity()
        ..setEntry(3, 2, 0.001)
        ..rotateX(frame.tilt.dx)
        ..rotateY(frame.tilt.dy),
      child: child,
    ),
  );
}
