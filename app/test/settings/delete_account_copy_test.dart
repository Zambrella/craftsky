import 'package:craftsky_app/l10n/generated/app_localizations_en.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('deletion confirmation explains the permanent CraftSky boundary', () {
    final copy = AppLocalizationsEn().deleteAccountBoundary('@alice.test');

    expect(copy, contains('@alice.test'));
    expect(copy, contains('all your CraftSky data from your PDS'));
    expect(copy, contains('all private data held by CraftSky'));
    expect(copy, contains('won’t delete your PDS, DID'));
    expect(copy, contains('wider AT Protocol account'));
    expect(
      copy,
      contains(
        '\n\nTo continue, you’ll need to authenticate with your PDS again.',
      ),
    );
  });
}
