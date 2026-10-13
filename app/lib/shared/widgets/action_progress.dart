import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:material_ui/material_ui.dart';

/// Keeps an action's size and label while replacing its visible content.
class ActionProgress extends StatelessWidget {
  const new({
    required this.isLoading,
    required this.child,
    this.size = 18,
    super.key,
  });

  final bool isLoading;
  final Widget child;
  final double size;

  @override
  Widget build(BuildContext context) {
    // A confirmation or editor route is waiting for the user, not displaying
    // work on its obscured source control. Resume progress when it returns.
    if (!isLoading || ModalRoute.isCurrentOf(context) == false) return child;
    return Stack(
      alignment: Alignment.center,
      children: [
        Visibility(
          visible: false,
          maintainState: true,
          maintainAnimation: true,
          maintainSize: true,
          maintainSemantics: true,
          child: child,
        ),
        // Disabled button foregrounds can be too faint (or white on grey).
        // Use the branded indicator's primary colour for visible progress.
        StitchProgressIndicator(size: size),
      ],
    );
  }
}
