import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_formatters.dart';
import 'package:craftsky_app/business/models/business_labels.dart';
import 'package:craftsky_app/business/widgets/business_image.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

class EventCard extends StatelessWidget {
  const EventCard({required this.event, required this.onTap, super.key});

  final BusinessEvent event;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final l10n = AppLocalizations.of(context);
    final display = BusinessFormatters.event(event, l10n);
    final roles = event.roles
        .map((role) => BusinessLabels.eventRole(role, l10n))
        .join(', ');

    return Semantics(
      button: true,
      child: Material(
        color: theme.colorScheme.surfaceContainerLow,
        clipBehavior: Clip.antiAlias,
        borderRadius: BorderRadius.circular(radii.r2),
        child: InkWell(
          onTap: onTap,
          child: DecoratedBox(
            decoration: BoxDecoration(
              border: Border.all(color: theme.colorScheme.outlineVariant),
              borderRadius: BorderRadius.circular(radii.r2),
            ),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                if (event.image case final image?)
                  Semantics(
                    image: true,
                    label: image.alt,
                    child: SizedBox.square(
                      dimension: 112,
                      child: BusinessImage(
                        image: image,
                        networkUrl: image.thumb,
                        fit: BoxFit.cover,
                      ),
                    ),
                  ),
                Expanded(
                  child: Padding(
                    padding: EdgeInsets.all(spacing.sp3),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(event.name, style: theme.textTheme.titleMedium),
                        SizedBox(height: spacing.sp1),
                        Text(display.date),
                        Text(display.time),
                        if (roles.isNotEmpty) Text(roles),
                        if (event.mode case final mode?)
                          Text(BusinessLabels.eventMode(mode, l10n)),
                        if (event.venueName case final venue?) Text(venue),
                      ],
                    ),
                  ),
                ),
                Padding(
                  padding: EdgeInsets.all(spacing.sp2),
                  child: const Icon(Icons.chevron_right),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
