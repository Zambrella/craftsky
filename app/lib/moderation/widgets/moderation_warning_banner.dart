import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

class ModerationWarningBanner extends StatelessWidget {
  const ModerationWarningBanner({required this.warningKind, super.key});

  final String warningKind;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final text = switch (warningKind) {
      'post' => l10n.moderationWarningPost,
      'profile' => l10n.moderationWarningProfile,
      'author' => l10n.moderationWarningAuthor,
      _ => l10n.moderationWarningPost,
    };
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    return Semantics(
      label: text,
      child: Container(
        padding: EdgeInsets.all(spacing.sp3),
        decoration: BoxDecoration(
          color: theme.colorScheme.secondaryContainer,
          borderRadius: BorderRadius.circular(radii.r3),
        ),
        child: Row(
          children: [
            Icon(
              Icons.info_outline,
              color: theme.colorScheme.onSecondaryContainer,
            ),
            SizedBox(width: spacing.sp2),
            Expanded(
              child: Text(
                text,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.onSecondaryContainer,
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
