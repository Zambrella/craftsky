import 'package:material_ui/material_ui.dart';

/// Wraps [child] in a [MediaQuery] that clamps the text scaler to
/// `[minTextScaleFactor, maxTextScaleFactor]`.
class TextScaleFactorClamper extends StatelessWidget {
  const new({
    required this.child,
    this.minTextScaleFactor = 1.0,
    this.maxTextScaleFactor = 1.5,
    super.key,
  });

  final Widget child;
  final double minTextScaleFactor;
  final double maxTextScaleFactor;

  @override
  Widget build(BuildContext context) {
    final mediaQuery = MediaQuery.of(context);
    final scaler = mediaQuery.textScaler.clamp(
      minScaleFactor: minTextScaleFactor,
      maxScaleFactor: maxTextScaleFactor,
    );
    return MediaQuery(
      data: mediaQuery.copyWith(textScaler: scaler),
      child: child,
    );
  }
}
