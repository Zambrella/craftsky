import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Full-screen error fallback for the profile page. Rendered when the
/// initial profile fetch fails and there's no cached value to show.
class ProfilePageError extends StatelessWidget {
  const ProfilePageError({
    required this.error,
    required this.onRetry,
    super.key,
  });

  final Object error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    return Center(
      child: Padding(
        padding: EdgeInsets.all(spacing.sp6),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              l10n.profileLoadErrorTitle,
              style: theme.textTheme.headlineSmall,
              textAlign: TextAlign.center,
            ),
            SizedBox(height: spacing.sp2),
            Text(
              '$error',
              // `outline` carries the brand's ink3 (tertiary text) per
              // the ColorScheme override in app_theme.dart.
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.outline,
              ),
              textAlign: TextAlign.center,
            ),
            // 20px sits between sp4(16) and sp5(24); intentional
            // breathing room above the retry button.
            const SizedBox(height: 20),
            ChunkyButton(
              onPressed: onRetry,
              child: Text(l10n.profileLoadErrorRetry),
            ),
          ],
        ),
      ),
    );
  }
}
