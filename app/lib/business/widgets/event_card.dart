import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_formatters.dart';
import 'package:craftsky_app/business/models/business_labels.dart';
import 'package:craftsky_app/business/widgets/business_image.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

class EventCard extends StatelessWidget {
  const EventCard({
    required this.event,
    required this.onTap,
    this.trailing,
    super.key,
  });

  final BusinessEvent event;
  final VoidCallback onTap;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    final display = BusinessFormatters.event(event, l10n);
    final roles = event.roles
        .map((role) => BusinessLabels.eventRole(role, l10n))
        .join(', ');

    return Semantics(
      button: true,
      child: CraftskyCard(
        child: InkWell(
          onTap: onTap,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (event.image case final image?)
                Semantics(
                  image: true,
                  label: image.alt,
                  child: AspectRatio(
                    key: const Key('event-card-image'),
                    aspectRatio: 16 / 9,
                    child: BusinessImage(
                      image: image,
                      networkUrl: image.thumb,
                      fit: BoxFit.cover,
                    ),
                  ),
                ),
              Padding(
                padding: EdgeInsets.all(spacing.sp3),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Expanded(
                          child: Text(
                            event.name,
                            style: theme.textTheme.titleLarge?.copyWith(
                              fontWeight: FontWeight.w800,
                            ),
                          ),
                        ),
                        if (trailing case final action?)
                          SizedBox.square(
                            dimension: spacing.sp6,
                            child: action,
                          )
                        else
                          const Icon(CraftskyIconsBold.next),
                      ],
                    ),
                    SizedBox(height: spacing.sp2),
                    _EventCardLine(
                      icon: CraftskyIcons.date,
                      text: display.date,
                    ),
                    SizedBox(height: spacing.sp1),
                    _EventCardLine(
                      icon: CraftskyIcons.schedule,
                      text: display.time,
                    ),
                    if (event.venueName case final venue?) ...[
                      SizedBox(height: spacing.sp1),
                      _EventCardLine(
                        icon: CraftskyIcons.location,
                        text: venue,
                      ),
                    ],
                    if (event.mode case final mode?) ...[
                      SizedBox(height: spacing.sp1),
                      _EventCardLine(
                        icon: CraftskyIcons.people,
                        text: BusinessLabels.eventMode(mode, l10n),
                      ),
                    ],
                    if (roles.isNotEmpty) ...[
                      SizedBox(height: spacing.sp1),
                      _EventCardLine(
                        icon: CraftskyIcons.businessIdentity,
                        text: roles,
                      ),
                    ],
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _EventCardLine extends StatelessWidget {
  const _EventCardLine({required this.icon, required this.text});

  final IconData icon;
  final String text;

  @override
  Widget build(BuildContext context) {
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Icon(icon, size: 18),
        SizedBox(width: spacing.sp2),
        Expanded(child: Text(text)),
      ],
    );
  }
}
