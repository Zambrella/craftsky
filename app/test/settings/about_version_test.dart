import 'package:craftsky_app/l10n/generated/app_localizations_en.dart';
import 'package:craftsky_app/settings/about_version.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final l10n = AppLocalizationsEn();

  test('About uses the shared localized version and build formatter', () {
    expect(
      buildVersionLabel(
        l10n,
        version: '1.2.3',
        buildNumber: '123',
      ),
      l10n.navigationBuildVersion('1.2.3', '123'),
    );
  });

  test('incomplete package metadata never produces malformed punctuation', () {
    expect(
      buildVersionLabel(l10n, version: '1.2.3', buildNumber: ''),
      '1.2.3',
    );
    expect(
      buildVersionLabel(l10n, version: ' 1.2.3 ', buildNumber: '   '),
      '1.2.3',
    );
    expect(buildVersionLabel(l10n, version: '', buildNumber: '123'), isNull);
    expect(buildVersionLabel(l10n, version: null, buildNumber: null), isNull);
  });
}
