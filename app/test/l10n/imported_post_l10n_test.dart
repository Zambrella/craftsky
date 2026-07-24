import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-015 imported post provenance has localized accessible copy', () {
    final l10n = lookupAppLocalizations(const Locale('en'));

    expect(l10n.postImportedFromInstagram, 'Imported from Instagram');
  });
}
