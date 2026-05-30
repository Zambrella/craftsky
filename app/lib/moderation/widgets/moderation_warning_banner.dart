import 'package:craftsky_app/l10n/generated/app_localizations.dart';
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
    return Semantics(
      label: text,
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: theme.colorScheme.secondaryContainer,
          borderRadius: BorderRadius.circular(12),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(
              Icons.info_outline,
              color: theme.colorScheme.onSecondaryContainer,
            ),
            const SizedBox(width: 8),
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
