import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:material_ui/material_ui.dart';

export 'generated/app_localizations.dart';

// gen-l10n still emits legacy design delegates in Flutter 3.47. Keep the
// generated translations, and use the standalone design system delegates.
const appLocalizationsDelegates = <LocalizationsDelegate<dynamic>>[
  AppLocalizations.delegate,
  ...GlobalMaterialLocalizations.delegates,
];
