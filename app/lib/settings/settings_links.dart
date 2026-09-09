import 'package:craftsky_app/shared/link/external_link.dart';

final settingsTermsUri = Uri.https('craftsky.social', '/terms');
final settingsPrivacyUri = Uri.https('craftsky.social', '/privacy');
final Uri communityGuidelinesUri = Uri.parse(
  'https://docs.google.com/document/d/'
  '1BXqycv4IvnsVGWAhK1FZ94XF6hrNiW4FVFFtfjGyT94/edit?usp=sharing',
);
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
