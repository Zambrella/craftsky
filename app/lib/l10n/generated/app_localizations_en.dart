// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'Craftsky';

  @override
  String get homeSubtitle => 'Scaffold ready';

  @override
  String homeVersionLabel(String version) {
    return 'v$version';
  }

  @override
  String get initializationFailedTitle => 'Initialization Failed';

  @override
  String get retryButton => 'Retry';

  @override
  String get routingErrorTitle => 'Something went wrong';

  @override
  String get goHomeButton => 'Go home';
}
