import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
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
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              "That didn't load.",
              style: theme.textTheme.headlineSmall,
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            Text(
              '$error',
              style: theme.textTheme.bodySmall?.copyWith(
                color: BrandColors.ink3,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 20),
            ChunkyButton(onPressed: onRetry, child: const Text('Try again')),
          ],
        ),
      ),
    );
  }
}
