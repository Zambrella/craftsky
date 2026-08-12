import 'package:craftsky_app/shared/link/external_link.dart';

final settingsTermsUri = Uri.https('craftsky.social', '/terms');
final settingsPrivacyUri = Uri.https('craftsky.social', '/privacy');
final settingsSupportUri = Uri.https(
  'github.com',
  '/Zambrella/craftsky/discussions',
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
