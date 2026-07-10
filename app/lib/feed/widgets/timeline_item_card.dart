import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';

/// Adds timeline context, such as repost attribution, above a post card.
class TimelineItemCard extends StatelessWidget {
  const TimelineItemCard({
    required this.item,
    required this.child,
    super.key,
  });

  final TimelineItem item;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final reason = item.reason;
    if (reason == null) return child;

    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);
    final displayName = reason.by.displayName ?? reason.by.handle;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 10, 24, 0),
          child: Row(
            children: [
              Icon(
                Icons.repeat,
                size: 16,
                color: theme.colorScheme.outline,
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  l10n.postRepostedBy(displayName),
                  style: theme.textTheme.labelMedium?.copyWith(
                    color: theme.colorScheme.outline,
                  ),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ],
          ),
        ),
        child,
      ],
    );
  }
}
