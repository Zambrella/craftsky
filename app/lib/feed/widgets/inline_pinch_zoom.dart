import 'dart:async';

import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';

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
  late final AnimationController _reset;
  late Animation<Matrix4> _resetMatrix;
  OverlayEntry? _entry;
  Matrix4 _matrix = Matrix4.identity();
  Offset _origin = Offset.zero;
  Offset _startCenter = Offset.zero;
  double _startDistance = 1;
  Size _size = Size.zero;

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
  void dispose() {
    _stopRouting();
    _removeOverlay();
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

    // Stop a drag that the feed or carousel has already accepted. Do not
    // interfere with their ordinary one-finger scrolling or page swipes.
    for (final axis in [Axis.vertical, Axis.horizontal]) {
      final scrollable = Scrollable.maybeOf(context, axis: axis);
      if (scrollable != null) {
        _scrollHolds.add(scrollable.position.hold(() {}));
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
    }
    if (_entry == null || _pointers.length >= 2 || _reset.isAnimating) return;
    for (final hold in _scrollHolds) {
      hold.cancel();
    }
    _scrollHolds.clear();
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
    for (final hold in _scrollHolds) {
      hold.cancel();
    }
    _scrollHolds.clear();
    _entry?.remove();
    _entry?.dispose();
    _entry = null;
  }

  @override
  Widget build(BuildContext context) => Listener(
    behavior: HitTestBehavior.opaque,
    onPointerDown: _onDown,
    child: Opacity(opacity: _entry == null ? 1 : 0, child: widget.child),
  );
}
