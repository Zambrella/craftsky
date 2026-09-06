import 'package:craftsky_app/shared/link/external_link.dart';

final settingsTermsUri = Uri.https('craftsky.social', '/terms');
final settingsPrivacyUri = Uri.https('craftsky.social', '/privacy');
final Uri settingsSupportUri = Uri.parse(
  'https://userinput.app/s/did:plc:lmmx63zcns6gewgxqfdt4kof/'
  '3mpr5izppvt2k?lang=en',
);

Future<bool> tryLaunchSettingsLink(
  Uri uri,
  ExternalLinkLauncher launcher,
) async {
  try {
    return await launcher(uri);
  } on Object {
    return false;
  }
}
