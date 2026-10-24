import 'dart:math' as math;

import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:material_ui/material_ui.dart';

/// Morphs one ChunkyButton between full-width primary and compact secondary.
class AnimatedChunkyButton extends StatefulWidget {
  const new({
    required this.compact,
    required this.label,
    required this.compactLabel,
    required this.onPressed,
    required this.actionKey,
    this.focusNode,
    this.style,
    this.child,
    super.key,
  });

  final bool compact;
  final String label;
  final String compactLabel;
  final VoidCallback? onPressed;
  final Key actionKey;
  final FocusNode? focusNode;
  final ButtonStyle? style;
  final Widget? child;

  @override
  State<AnimatedChunkyButton> createState() => _AnimatedChunkyButtonState();
}

class _AnimatedChunkyButtonState extends State<AnimatedChunkyButton> {
  // Preserve press and label animation when the action key changes.
  final GlobalKey _buttonKey = GlobalKey();

  @override
  Widget build(BuildContext context) {
    final compact = widget.compact;
    final label = widget.label;
    final compactLabel = widget.compactLabel;
    final child = widget.child;
    final durations = Theme.of(context).extension<DurationTheme>()!;
    final duration = MediaQuery.disableAnimationsOf(context)
        ? Duration.zero
        : durations.medium;
    final defaults = const ChunkyButton(
      onPressed: null,
      child: SizedBox.shrink(),
    ).defaultStyleOf(context);
    final textStyle = defaults.textStyle!.resolve({})!;
    final direction = Directionality.of(context);
    final padding = defaults.padding!.resolve({})!.resolve(direction);
    final minimum = defaults.minimumSize!.resolve({})!;
    final text = TextPainter(
      text: TextSpan(text: compactLabel, style: textStyle),
      textDirection: direction,
      textScaler: MediaQuery.textScalerOf(context),
    )..layout();
    final compactWidth = math.max(
      minimum.width,
      text.width + padding.horizontal,
    );
    text.dispose();

    return LayoutBuilder(
      builder: (context, constraints) => TweenAnimationBuilder<double>(
        tween: Tween(end: compact ? 1 : 0),
        duration: duration,
        curve: durations.ease,
        builder: (context, progress, _) {
          final fullWidth = constraints.hasBoundedWidth
              ? constraints.maxWidth
              : compactWidth;
          final targetWidth = math.min(compactWidth, fullWidth);
          final width = fullWidth + (targetWidth - fullWidth) * progress;
          return Align(
            alignment: AlignmentDirectional.centerEnd,
            child: SizedBox(
              key: widget.actionKey,
              width: width,
              child: ChunkyButton(
                key: _buttonKey,
                onPressed: widget.onPressed,
                focusNode: widget.focusNode,
                style: compact ? null : widget.style,
                variantProgress: progress,
                child: Semantics(
                  label: compact
                      ? compactLabel
                      : (child == null ? label : null),
                  excludeSemantics: compact || child == null,
                  child: AnimatedSwitcher(
                    duration: duration,
                    switchInCurve: durations.ease,
                    switchOutCurve: durations.ease,
                    transitionBuilder: (child, animation) => FadeTransition(
                      opacity: animation,
                      alwaysIncludeSemantics: true,
                      child: child,
                    ),
                    layoutBuilder: (currentChild, previousChildren) => Stack(
                      alignment: Alignment.center,
                      children: [
                        for (final child in previousChildren)
                          ExcludeSemantics(child: child),
                        ?currentChild,
                      ],
                    ),
                    child: compact || child == null
                        ? Text(
                            compact ? compactLabel : label,
                            key: ValueKey(compact),
                            maxLines: 1,
                            softWrap: false,
                            overflow: TextOverflow.fade,
                          )
                        : child,
                  ),
                ),
              ),
            ),
          );
        },
      ),
    );
  }
}
