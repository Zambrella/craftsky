import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

class ScheduledPostCapacityWarning extends StatelessWidget {
  const ScheduledPostCapacityWarning({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final semanticColors = theme.extension<SemanticColorsTheme>()!;
    final message = l10n.scheduledPostCapacityWarning;

    return Semantics(
      label: message,
      liveRegion: true,
      container: true,
      child: ExcludeSemantics(
        child: Container(
          padding: EdgeInsets.all(spacing.sp3),
          decoration: BoxDecoration(
            color: semanticColors.warningSurface,
            border: Border.all(color: semanticColors.warning),
            borderRadius: BorderRadius.circular(radii.r3),
          ),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Icon(
                Icons.warning_amber_rounded,
                color: semanticColors.warning,
              ),
              SizedBox(width: spacing.sp2),
              Expanded(
                child: Text(message, style: theme.textTheme.bodyMedium),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
