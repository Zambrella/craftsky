import 'dart:async';

import 'package:craftsky_app/business/models/business_action.dart';
import 'package:craftsky_app/business/models/business_labels.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

class BusinessProfileSummary extends StatelessWidget {
  const BusinessProfileSummary({
    required this.business,
    this.launchExternal = launchExternalLink,
    this.confirmExternal = showOpenLinkDialog,
    super.key,
  });

  final BusinessProfile? business;
  final ExternalLinkLauncher launchExternal;
  final ExternalLinkConfirmer confirmExternal;

  static bool hasContent(BusinessProfile? business) {
    final tagline = business?.tagline?.trim();
    return (tagline != null && tagline.isNotEmpty) ||
        business?.location != null ||
        (business?.serviceArea?.trim().isNotEmpty ?? false) ||
        business?.primaryAction != null;
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final tagline = business?.tagline?.trim();
    final location = switch (business) {
      BusinessProfile(location: final location?) => BusinessLabels.location(
        location,
        l10n,
      ),
      BusinessProfile(serviceArea: final area?) when area.trim().isNotEmpty =>
        area.trim(),
      _ => null,
    };
    final action = business?.primaryAction;
    final destination = action == null
        ? null
        : BusinessActionFormatter.destination(action.destination);
    final presentation = action == null
        ? null
        : BusinessActionFormatter.presentation(action.type, l10n);

    final hasTagline = tagline != null && tagline.isNotEmpty;
    final hasLocation = location != null;
    final hasDetails = hasTagline || hasLocation;

    return SizedBox(
      width: double.infinity,
      child: Column(
        children: [
          if (hasTagline)
            Text(
              tagline,
              textAlign: TextAlign.center,
              style: theme.textTheme.titleMedium,
            ),
          if (hasLocation) ...[
            if (hasTagline) SizedBox(height: spacing.sp2),
            Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(
                  Icons.location_on_outlined,
                  size: 18,
                  color: Theme.of(context).colorScheme.onSurfaceVariant,
                ),
                SizedBox(width: spacing.sp1),
                Flexible(
                  child: Text(location, textAlign: TextAlign.center),
                ),
              ],
            ),
          ],
          if (action != null && destination != null) ...[
            if (hasDetails) SizedBox(height: spacing.sp2),
            OutlinedButton.icon(
              onPressed: () => unawaited(
                confirmAndLaunchExternalAction(
                  context,
                  uri: destination,
                  launchUrl: launchExternal,
                  confirmOpenLink: confirmExternal,
                ),
              ),
              icon: Icon(presentation!.icon),
              label: Text(presentation.label),
            ),
          ],
        ],
      ),
    );
  }
}
