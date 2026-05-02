import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Generic empty/placeholder body for tabs that don't have data wiring
/// yet (Projects, Reposts, Saved). Returns a [SliverFillRemaining] so
/// it fills whatever scrollable space is left below the header chrome.
/// Per the design-system voice, copy is warm rather than apologetic.
class ProfileEmptyTab extends StatelessWidget {
  const ProfileEmptyTab({required this.message, super.key});

  final String message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return SliverFillRemaining(
      hasScrollBody: false,
      child: Center(
        child: Padding(
          padding: EdgeInsets.all(spacing.sp6),
          child: Text(
            message,
            textAlign: TextAlign.center,
            // `outline` carries the brand's ink3 (tertiary text) per
            // the ColorScheme override in app_theme.dart.
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.outline,
            ),
          ),
        ),
      ),
    );
  }
}
