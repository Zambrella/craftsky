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

/// A small, damped rotation, never more than 2.5 degrees in either axis.
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
  static const double _limit = 2.5 * math.pi / 180;
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
    if (_running == shouldRun) return;
    _running = shouldRun;
    if (!shouldRun) {
      _ticker.stop();
      unawaited(_subscription?.cancel());
      _subscription = null;
      _lastSample = null;
      _target = Offset.zero;
      _frames.value = const ShareCardFrame();
      return;
    }
    _lastTick = Duration.zero;
    _ticker.start();
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
    if (!_running || !event.x.isFinite || !event.y.isFinite) return;
    final previous = _lastSample;
    _lastSample = event.timestamp;
    if (previous == null) return;
    final dt = (event.timestamp.difference(previous).inMicroseconds / 1e6)
        .clamp(0.0, 0.1);
    _target = Offset(
      (_target.dx + event.x * dt * 0.35).clamp(-_limit, _limit),
      (_target.dy + event.y * dt * 0.35).clamp(-_limit, _limit),
    );
    if (!_ticker.isActive && _target.distanceSquared > 1e-8) {
      _lastTick = Duration.zero;
      _ticker.start();
    }
  }

  void _tick(Duration elapsed) {
    final dt = ((elapsed - _lastTick).inMicroseconds / 1e6).clamp(0.0, 0.1);
    _lastTick = elapsed;
    final tilt = Offset.lerp(
      _frames.value.tilt,
      _target,
      1 - math.exp(-dt * 9),
    )!;
    _target *= math.exp(-dt * 1.8);
    if (tilt.distanceSquared < 1e-8 && _target.distanceSquared < 1e-8) {
      _target = Offset.zero;
      if (_frames.value.tilt != Offset.zero) {
        _frames.value = const ShareCardFrame();
      }
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
    child: widget.builder(context, _frames),
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
