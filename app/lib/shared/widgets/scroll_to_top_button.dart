import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

class ScrollToTopButton extends StatelessWidget {
  const ScrollToTopButton({
    required this.visible,
    required this.onPressed,
    required this.tooltip,
    super.key,
  });

  final bool visible;
  final VoidCallback onPressed;
  final String tooltip;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final durations = theme.extension<DurationTheme>()!;
    final duration = MediaQuery.disableAnimationsOf(context)
        ? Duration.zero
        : durations.medium;
    return AnimatedSwitcher(
      duration: duration,
      switchInCurve: durations.easePop,
      switchOutCurve: durations.ease,
      transitionBuilder: (child, animation) => FadeTransition(
        opacity: animation,
        child: ScaleTransition(scale: animation, child: child),
      ),
      child: visible
          ? FloatingActionButton.small(
              key: const Key('scroll-to-top-button'),
              heroTag: null,
              tooltip: tooltip,
              backgroundColor: theme.colorScheme.surfaceContainerHigh,
              foregroundColor: theme.colorScheme.onSurfaceVariant,
              onPressed: onPressed,
              child: const Icon(CraftskyIconsBold.moveUp),
            )
          : const SizedBox.shrink(key: Key('scroll-to-top-hidden')),
    );
  }
}
