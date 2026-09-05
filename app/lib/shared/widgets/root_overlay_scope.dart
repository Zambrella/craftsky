import 'package:flutter/widgets.dart';

class RootOverlayScope extends InheritedWidget {
  const RootOverlayScope({
    required this.overlayContext,
    required super.child,
    super.key,
  });

  final BuildContext overlayContext;

  static BuildContext overlayContextOf(BuildContext context) =>
      context
          .dependOnInheritedWidgetOfExactType<RootOverlayScope>()
          ?.overlayContext ??
      context;

  @override
  bool updateShouldNotify(RootOverlayScope oldWidget) =>
      overlayContext != oldWidget.overlayContext;
}
