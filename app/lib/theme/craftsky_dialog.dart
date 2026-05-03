import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// A branded confirm/alert dialog. Paper-cutout aesthetic: thick ink border,
/// chunky `r3` corners, hard-offset drop shadow drawn via stacked layers (the
/// same approach used by `ChunkyButton`).
///
/// Most callers should reach for [showCraftskyConfirmDialog],
/// [showCraftskyDestructiveConfirmDialog], or [showCraftskyAlertDialog]
/// rather than constructing this widget directly.
class CraftskyDialog extends StatelessWidget {
  const CraftskyDialog({
    required this.title,
    required this.body,
    required this.actions,
    super.key,
  });

  final String title;
  final Widget body;
  final List<Widget> actions;

  /// Maximum width on wide screens. Below this, the dialog tracks the
  /// available width minus [_horizontalInset] on each side.
  static const double _maxWidth = 360;

  /// Horizontal inset reserved on small screens so the 10px shadow never
  /// touches the edge.
  static const double _horizontalInset = 24;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final shadows = theme.extension<BrandShadowTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;

    final shadowOffset = shadows.dropLg.first.offset;
    final shadowColor = shadows.dropLg.first.color;
    final radius = BorderRadius.circular(radii.r3);

    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(
          horizontal: _horizontalInset,
          vertical: _horizontalInset,
        ),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: _maxWidth),
          child: IntrinsicHeight(
            child: Stack(
              children: [
                Positioned.fill(
                  child: Transform.translate(
                    offset: shadowOffset,
                    child: DecoratedBox(
                      decoration: BoxDecoration(
                        color: shadowColor,
                        borderRadius: radius,
                      ),
                    ),
                  ),
                ),
                Material(
                  color: Colors.transparent,
                  child: Container(
                    decoration: BoxDecoration(
                      color: swatches.paper3,
                      borderRadius: radius,
                      border: Border.all(color: colors.onSurface, width: 1.5),
                    ),
                    padding: EdgeInsets.all(spacing.sp5),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        Text(title, style: theme.textTheme.titleLarge),
                        SizedBox(height: spacing.sp4),
                        DefaultTextStyle.merge(
                          style: theme.textTheme.bodyMedium,
                          child: body,
                        ),
                        SizedBox(height: spacing.sp5),
                        Wrap(
                          alignment: WrapAlignment.end,
                          spacing: spacing.sp2,
                          runSpacing: spacing.sp2,
                          children: actions,
                        ),
                      ],
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
