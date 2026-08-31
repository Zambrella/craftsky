import 'dart:async';

import 'package:craftsky_app/business/models/business_action.dart';
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

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final tagline = business?.tagline?.trim();
    final action = business?.primaryAction;
    final destination = action == null
        ? null
        : BusinessActionFormatter.destination(action.destination);
    final presentation = action == null
        ? null
        : BusinessActionFormatter.presentation(action.type, l10n);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          l10n.businessProfileLabel,
          style: Theme.of(context).textTheme.labelLarge,
        ),
        if (tagline != null && tagline.isNotEmpty) ...[
          SizedBox(height: spacing.sp1),
          Text(tagline),
        ],
        if (action != null && destination != null) ...[
          SizedBox(height: spacing.sp2),
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
    );
  }
}
