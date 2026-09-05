import 'package:craftsky_app/business/models/business_labels.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';

abstract final class BusinessActionFormatter {
  static const List<String> knownTypes = BusinessLabels.actions;

  static BusinessActionPresentation presentation(
    String type,
    AppLocalizations l10n,
  ) => BusinessActionPresentation(
    label: BusinessLabels.action(type, l10n),
    icon: type == 'email'
        ? CraftskyIconsBold.email
        : CraftskyIconsBold.externalLink,
  );

  static Uri? destination(String? value) =>
      hydratedExternalActionUri(value, allowMailto: true);
}

final class BusinessActionPresentation {
  const BusinessActionPresentation({required this.label, required this.icon});

  final String label;
  final IconData icon;
}
