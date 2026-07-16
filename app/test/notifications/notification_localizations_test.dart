import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-017 notification surfaces have localized accessible copy', () {
    final l10n = lookupAppLocalizations(const Locale('en'));

    expect(l10n.notificationSettingsAction, isNotEmpty);
    expect(l10n.notificationCategoryEverythingElse, isNotEmpty);
    expect(l10n.notificationUnavailableRow, isNotEmpty);
    expect(l10n.notificationNewActivityCount(1), '1 new activity');
    expect(l10n.notificationNewActivityCount(100), '100 new activities');
  });
}
