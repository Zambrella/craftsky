import 'package:craftsky_app/settings/settings_links.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('legal destinations are canonical HTTPS URLs', () {
    expect(settingsTermsUri, Uri.https('craftsky.social', '/terms'));
    expect(settingsPrivacyUri, Uri.https('craftsky.social', '/privacy'));
    expect(
      settingsSupportUri.toString(),
      'https://userinput.app/s/did:plc:lmmx63zcns6gewgxqfdt4kof/'
      '3mpr5izppvt2k?lang=en',
    );
  });

  test(
    'link launcher false or exception maps to a safe false result',
    () async {
      expect(
        await tryLaunchSettingsLink(
          settingsTermsUri,
          (_) async => false,
        ),
        isFalse,
      );
      expect(
        await tryLaunchSettingsLink(
          settingsPrivacyUri,
          (_) async => throw StateError('secret platform detail'),
        ),
        isFalse,
      );
      expect(
        await tryLaunchSettingsLink(settingsTermsUri, (_) async => true),
        isTrue,
      );
    },
  );
}
