import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Display name (DM Serif Display) + optional pronouns + `@handle` block,
/// matching the order from the design mockup. Pronouns are currently a
/// placeholder — there's no wire field for them yet.
class ProfileIdentity extends StatelessWidget {
  const ProfileIdentity({
    required this.handle,
    this.displayName,
    this.pronouns,
    super.key,
  });

  final String handle;
  final String? displayName;
  final String? pronouns;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final name = (displayName?.isNotEmpty ?? false) ? displayName! : '@$handle';

    // `outline` carries the brand's ink3 (tertiary text) per the
    // ColorScheme override in app_theme.dart.
    final mutedInk = theme.colorScheme.outline;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Row(
          crossAxisAlignment: CrossAxisAlignment.baseline,
          textBaseline: TextBaseline.alphabetic,
          children: [
            Flexible(
              child: Text(
                name,
                style: theme.textTheme.headlineMedium,
                overflow: TextOverflow.ellipsis,
              ),
            ),
            if (pronouns != null) ...[
              SizedBox(width: spacing.sp2),
              Text(
                pronouns!,
                style: theme.textTheme.bodySmall?.copyWith(color: mutedInk),
              ),
            ],
          ],
        ),
        if (displayName?.isNotEmpty ?? false) ...[
          const SizedBox(height: 2),
          Text(
            '@$handle',
            // `onSurfaceVariant` (ink2) rather than `outline` (ink3) —
            // the @handle reads as a secondary identifier paired with
            // the display name, not tertiary metadata, so it wants
            // the darker secondary-text strength.
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ],
    );
  }
}
