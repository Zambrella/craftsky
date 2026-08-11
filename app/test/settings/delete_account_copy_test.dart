import 'package:craftsky_app/l10n/generated/app_localizations_en.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('deletion confirmation states the complete permanent boundary', () {
    final copy = AppLocalizationsEn().deleteAccountBoundary('@alice.test');

    expect(copy, contains('@alice.test'));
    expect(copy, contains('CraftSky membership'));
    expect(copy, contains('private CraftSky data'));
    expect(copy, contains('social.craftsky.*'));
    expect(copy, contains('signed out of CraftSky on every device'));
    expect(copy, contains('cannot be undone'));
    expect(copy, contains('does not delete your AT Protocol account'));
    expect(copy, contains('does not delete your PDS account'));
    expect(copy, contains('records from other apps'));
    expect(copy, contains('does not directly delete blobs'));
    expect(copy, contains('does not wait for your PDS'));
    expect(copy, contains('the next time they connect'));
  });
}
