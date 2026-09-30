import 'dart:async';

import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';

/// Lets a scrollable containing inline images suspend pull-to-refresh during a
/// pinch without changing ordinary one-finger drag behavior.
class InlinePinchZoomScope extends InheritedWidget {
  const InlinePinchZoomScope({
    required this.onPinchChanged,
    required super.child,
    super.key,
  });

  final ValueChanged<bool> onPinchChanged;

  static InlinePinchZoomScope? maybeOf(BuildContext context) =>
      context.dependOnInheritedWidgetOfExactType<InlinePinchZoomScope>();

  @override
  bool updateShouldNotify(InlinePinchZoomScope oldWidget) =>
      onPinchChanged != oldWidget.onPinchChanged;
}

/// Lifts an inline image above its scrollables when a second finger lands.
/// Pointer events are used deliberately: a scrollable may have already won the
/// gesture arena before the second finger arrives.
class InlinePinchZoom extends StatefulWidget {
  const InlinePinchZoom({
    required this.child,
    this.overlayContext,
    this.maxScale = 4,
    this.barrierColor = Colors.black12,
    this.resetDuration = const Duration(milliseconds: 300),
    super.key,
  });

  final Widget child;
  final BuildContext? overlayContext;
  final double maxScale;
  final Color barrierColor;
  final Duration resetDuration;

  @override
  State<InlinePinchZoom> createState() => _InlinePinchZoomState();
}

class _InlinePinchZoomState extends State<InlinePinchZoom>
    with SingleTickerProviderStateMixin {
  final Map<int, Offset> _pointers = {};
  bool _routing = false;
  final List<ScrollHoldController> _scrollHolds = [];
  final Map<ScrollPosition, VoidCallback> _scrollLocks = {};
  late final AnimationController _reset;
  late Animation<Matrix4> _resetMatrix;
  OverlayEntry? _entry;
  Matrix4 _matrix = Matrix4.identity();
  Offset _origin = Offset.zero;
  Offset _startCenter = Offset.zero;
  double _startDistance = 1;
  Size _size = Size.zero;
  ValueChanged<bool>? _onPinchChanged;
  bool _pinchActive = false;

  @override
  void initState() {
    super.initState();
    _reset = AnimationController(vsync: this, duration: widget.resetDuration)
      ..addListener(() => _entry?.markNeedsBuild())
      ..addStatusListener((status) {
        if (status != AnimationStatus.completed) return;
        _removeOverlay();
        setState(() {});
      });
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _onPinchChanged = InlinePinchZoomScope.maybeOf(context)?.onPinchChanged;
  }

  @override
  void dispose() {
    if (_pinchActive) {
      final callback = _onPinchChanged;
      scheduleMicrotask(() => callback?.call(false));
      _pinchActive = false;
    }
    _stopRouting();
    _removeOverlay();
    _releaseScrolls();
    _reset.dispose();
    super.dispose();
  }

  void _onDown(PointerDownEvent event) {
    if (_pointers.containsKey(event.pointer)) return;
    if (_pointers.isEmpty) {
      GestureBinding.instance.pointerRouter.addGlobalRoute(_onGlobalEvent);
      _routing = true;
    }
    _pointers[event.pointer] = event.position;
    if (_pointers.length != 2 || _entry != null) return;

    _notifyPinchChanged(true);
    // Stop an accepted drag and keep both scrollables at their current offset.
    // A second pointer can still start a new drag after hold() is called.
    for (final axis in [Axis.vertical, Axis.horizontal]) {
      final scrollable = Scrollable.maybeOf(context, axis: axis);
      if (scrollable != null) {
        final position = scrollable.position;
        _scrollHolds.add(position.hold(() {}));
        final pixels = position.pixels;
        void keepPosition() {
          if (position.pixels != pixels) position.jumpTo(pixels);
        }

        position.addListener(keepPosition);
        _scrollLocks[position] = keepPosition;
      }
    }

    final box = context.findRenderObject()! as RenderBox;
    final overlay = Overlay.of(widget.overlayContext ?? context);
    final overlayBox = overlay.context.findRenderObject()! as RenderBox;
    _origin = box.localToGlobal(Offset.zero, ancestor: overlayBox);
    _size = box.size;
    _setBaseline();
    _matrix = Matrix4.identity();
    _entry = OverlayEntry(builder: _buildOverlay);
    overlay.insert(_entry!);
    setState(() {});
  }

  void _onGlobalEvent(PointerEvent event) {
    if (!mounted) return;
    if (event is PointerDownEvent) {
      if (_pointers.containsKey(event.pointer) || _pointers.isEmpty) return;
      final box = context.findRenderObject()! as RenderBox;
      if (box.size.contains(box.globalToLocal(event.position))) {
        _onDown(event);
      }
    } else if (event is PointerMoveEvent) {
      _onMove(event);
    } else if (event is PointerUpEvent || event is PointerCancelEvent) {
      _onEnd(event);
    }
  }

  void _setBaseline() {
    final points = _pointers.values.take(2).toList();
    _startCenter = (points[0] + points[1]) / 2;
    _startDistance = (points[0] - points[1]).distance;
  }

  void _onMove(PointerMoveEvent event) {
    if (!_pointers.containsKey(event.pointer)) return;
    _pointers[event.pointer] = event.position;
    if (_entry == null || _reset.isAnimating || _pointers.length != 2) return;
    final points = _pointers.values.take(2).toList();
    final center = (points[0] + points[1]) / 2;
    final scale =
        ((points[0] - points[1]).distance /
                _startDistance.clamp(1, double.infinity))
            .clamp(1.0, widget.maxScale);
    final focal = _startCenter - _origin;
    _matrix = Matrix4.identity()
      ..translateByDouble(
        center.dx - _startCenter.dx,
        center.dy - _startCenter.dy,
        0,
        1,
      )
      ..translateByDouble(focal.dx, focal.dy, 0, 1)
      ..scaleByDouble(scale, scale, 1, 1)
      ..translateByDouble(-focal.dx, -focal.dy, 0, 1);
    _entry!.markNeedsBuild();
  }

  void _onEnd(PointerEvent event) {
    if (_pointers.remove(event.pointer) == null) return;
    if (_pointers.isEmpty) {
      _stopRouting();
      if (_entry == null) {
        _releaseScrolls();
        _notifyPinchChanged(false);
      }
    }
    if (_entry == null || _pointers.length >= 2 || _reset.isAnimating) return;
    _resetMatrix = Matrix4Tween(begin: _matrix, end: Matrix4.identity())
        .animate(
          CurvedAnimation(parent: _reset, curve: Curves.fastOutSlowIn),
        );
    unawaited(_reset.forward(from: 0));
  }

  void _stopRouting() {
    if (!_routing) return;
    GestureBinding.instance.pointerRouter.removeGlobalRoute(_onGlobalEvent);
    _routing = false;
  }

  Widget _buildOverlay(BuildContext context) => IgnorePointer(
    child: Stack(
      children: [
        ModalBarrier(color: widget.barrierColor),
        Positioned(
          left: _origin.dx,
          top: _origin.dy,
          width: _size.width,
          height: _size.height,
          child: Transform(
            alignment: Alignment.topLeft,
            transform: _reset.isAnimating ? _resetMatrix.value : _matrix,
            child: widget.child,
          ),
        ),
      ],
    ),
  );

  void _removeOverlay() {
    final wasZooming = _entry != null;
    _entry?.remove();
    _entry?.dispose();
    _entry = null;
    if (_pointers.isEmpty) {
      _releaseScrolls();
      if (wasZooming) _notifyPinchChanged(false);
    }
  }

  void _notifyPinchChanged(bool active) {
    if (_pinchActive == active) return;
    _pinchActive = active;
    _onPinchChanged?.call(active);
  }

  void _releaseScrolls() {
    // Remove the guards before cancelling holds, which can resume scrolling.
    for (final entry in _scrollLocks.entries) {
      entry.key.removeListener(entry.value);
    }
    _scrollLocks.clear();
    for (final hold in _scrollHolds) {
      hold.cancel();
    }
    _scrollHolds.clear();
  }

  @override
  Widget build(BuildContext context) => Listener(
    behavior: HitTestBehavior.opaque,
    onPointerDown: _onDown,
    child: Opacity(opacity: _entry == null ? 1 : 0, child: widget.child),
  );
}
