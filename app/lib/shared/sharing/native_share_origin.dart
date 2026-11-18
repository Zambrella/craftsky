import 'package:flutter/widgets.dart';

/// Keeps the iPad popover anchor inside the visible view, even after scrolling.
Rect nativeShareOrigin(BuildContext context) {
  final renderObject = context.findRenderObject();
  final screen = Offset.zero & MediaQuery.sizeOf(context);
  if (renderObject is RenderBox && renderObject.hasSize) {
    final visible =
        (renderObject.localToGlobal(Offset.zero) & renderObject.size).intersect(
          screen,
        );
    if (!visible.isEmpty) return visible;
  }
  return Rect.fromCenter(center: screen.center, width: 1, height: 1);
}
