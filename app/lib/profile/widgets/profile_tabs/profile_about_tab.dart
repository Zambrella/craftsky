import 'package:craftsky_app/business/models/business_labels.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

/// Business details and member-since content for the business-only About tab.
class ProfileAboutTab extends StatelessWidget {
  const ProfileAboutTab({required this.profile, super.key});

  final Profile profile;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    final business = profile.business;

    // 20px sits between sp4(16) and sp5(24) — used as the section
    // gap. Kept as a literal because it doesn't map to a SpacingTheme
    // token.
    const sectionGap = SizedBox(height: 20);

    final children = <Widget>[
      if (business != null) ...[
        if (business.businessTypes.isNotEmpty) ...[
          _BusinessDetailSection(
            heading: l10n.businessTypesHeading,
            values: business.businessTypes
                .map((value) => BusinessLabels.openValue(value, l10n))
                .toList(growable: false),
          ),
          sectionGap,
        ],
        if (business.offerings.isNotEmpty) ...[
          _BusinessDetailSection(
            heading: l10n.businessOfferingsHeading,
            values: business.offerings
                .map((value) => BusinessLabels.openValue(value, l10n))
                .toList(growable: false),
          ),
          sectionGap,
        ],
        if (business.serviceArea?.trim() case final serviceArea?
            when serviceArea.isNotEmpty && business.location != null) ...[
          _BusinessTextSection(
            heading: l10n.businessServiceAreaHeading,
            value: serviceArea,
          ),
          sectionGap,
        ],
        if (business.hoursNote?.trim() case final hours?
            when hours.isNotEmpty) ...[
          _BusinessTextSection(
            heading: l10n.businessHoursHeading,
            value: hours,
          ),
          sectionGap,
        ],
      ],
      if (profile.createdAt != null) ...[
        Text(l10n.profileAboutJoinedHeading, style: theme.textTheme.labelSmall),
        SizedBox(height: spacing.sp1),
        Text(
          // Pass the active locale so e.g. "April 2026" / "avril 2026"
          // follow the user's language rather than the intl default.
          DateFormat.yMMMM(l10n.localeName).format(profile.createdAt!),
          style: theme.textTheme.bodyMedium,
        ),
      ],
    ];

    return SliverPadding(
      padding: EdgeInsets.all(spacing.sp4),
      sliver: SliverList.builder(
        itemCount: children.length,
        itemBuilder: (context, index) => children[index],
      ),
    );
  }
}

class _BusinessDetailSection extends StatelessWidget {
  const _BusinessDetailSection({required this.heading, required this.values});

  final String heading;
  final List<String> values;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(heading, style: theme.textTheme.labelSmall),
        SizedBox(height: spacing.sp2),
        Wrap(
          spacing: spacing.sp2,
          runSpacing: spacing.sp1,
          children: [for (final value in values) Chip(label: Text(value))],
        ),
      ],
    );
  }
}

class _BusinessTextSection extends StatelessWidget {
  const _BusinessTextSection({required this.heading, required this.value});

  final String heading;
  final String value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(heading, style: theme.textTheme.labelSmall),
        SizedBox(height: spacing.sp1),
        Text(value, style: theme.textTheme.bodyMedium),
      ],
    );
  }
}
