import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';

class ScheduledStagingProgress extends StatelessWidget {
  const ScheduledStagingProgress({
    required this.completed,
    required this.total,
    this.creating = false,
    super.key,
  });

  final int completed;
  final int total;
  final bool creating;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final safeTotal = total.clamp(0, 4);
    final safeCompleted = completed.clamp(0, safeTotal);
    final label = creating || safeTotal == 0
        ? l10n.scheduledPostCreating
        : l10n.scheduledPostStagingProgress(
            (safeCompleted + 1).clamp(1, safeTotal),
            safeTotal,
          );
    return Semantics(
      liveRegion: true,
      label: label,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          ExcludeSemantics(child: Text(label)),
          const SizedBox(height: 8),
          LinearProgressIndicator(
            value: creating || safeTotal == 0
                ? null
                : safeCompleted / safeTotal,
          ),
        ],
      ),
    );
  }
}
