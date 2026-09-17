import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  const host = 'app.craftsky.social';

  test('Android accepts only the two verified HTTPS callback paths', () {
    final manifest = File(
      'android/app/src/main/AndroidManifest.xml',
    ).readAsStringSync();

    expect('android:autoVerify="true"'.allMatches(manifest), hasLength(2));
    expect('android:scheme="https"'.allMatches(manifest), hasLength(2));
    expect('android:host="$host"'.allMatches(manifest), hasLength(2));
    expect(manifest, contains('android:path="/auth/complete"'));
    expect(
      manifest,
      contains('android:path="/account-deletion/reauth-complete"'),
    );
    expect(manifest, isNot(contains('android:scheme="craftsky"')));
    expect(manifest, isNot(contains('android:scheme="craftsky-dev"')));
  });

  test('Android registers the code-only custom scheme in debug only', () {
    final debugManifest = File(
      'android/app/src/debug/AndroidManifest.xml',
    ).readAsStringSync();

    expect(
      'android:scheme="craftsky-dev"'.allMatches(debugManifest),
      hasLength(2),
    );
    expect(debugManifest, contains('android:path="/auth/complete"'));
    expect(
      debugManifest,
      contains('android:path="/account-deletion/reauth-complete"'),
    );
    expect(debugManifest, isNot(contains('android:autoVerify')));
    expect(debugManifest, isNot(contains('android:scheme="craftsky"')));
  });

  test('iOS accepts the callback host through associated domains only', () {
    final infoPlist = File('ios/Runner/Info.plist').readAsStringSync();
    final entitlements = File(
      'ios/Runner/Runner.entitlements',
    ).readAsStringSync();

    expect(infoPlist, isNot(contains('CFBundleURLTypes')));
    expect(entitlements, contains('com.apple.developer.associated-domains'));
    expect(entitlements, contains('<string>applinks:$host</string>'));
  });

  test('iOS registers craftsky-dev through the Debug plist only', () {
    final debugInfoPlist = File(
      'ios/Runner/Info-Debug.plist',
    ).readAsStringSync();
    final releaseInfoPlist = File('ios/Runner/Info.plist').readAsStringSync();
    final project = File(
      'ios/Runner.xcodeproj/project.pbxproj',
    ).readAsStringSync();

    expect(debugInfoPlist, contains('<key>CFBundleURLTypes</key>'));
    expect(debugInfoPlist, contains('<string>craftsky-dev</string>'));
    expect(debugInfoPlist, isNot(contains('<string>craftsky</string>')));
    expect(releaseInfoPlist, isNot(contains('CFBundleURLTypes')));
    expect(releaseInfoPlist, isNot(contains('craftsky-dev')));
    expect(project, contains('INFOPLIST_FILE = Runner/Info-Debug.plist;'));
    expect(
      'INFOPLIST_FILE = Runner/Info.plist;'.allMatches(project),
      hasLength(2),
      reason: 'Profile and Release must keep the scheme-free plist',
    );
  });

  test(
    'domain associations authorize the production application identities',
    () {
      final assetLinks =
          jsonDecode(
                File(
                  '../verified-links/.well-known/assetlinks.json',
                ).readAsStringSync(),
              )
              as List<dynamic>;
      final androidTarget =
          (assetLinks.single as Map<String, dynamic>)['target']
              as Map<String, dynamic>;

      expect(androidTarget['package_name'], 'social.craftsky.app');
      const androidFingerprintPrefix =
          'D7:A8:2E:30:70:D7:8B:7F:A6:3D:22:DA:EE:06:5C:51:ED:96:39:7A:';
      const androidFingerprint =
          '${androidFingerprintPrefix}A5:C7:A5:20:CF:79:1A:BE:3B:17:D7:0D';
      expect(androidTarget['sha256_cert_fingerprints'], [androidFingerprint]);

      final association =
          jsonDecode(
                File(
                  '../verified-links/.well-known/apple-app-site-association',
                ).readAsStringSync(),
              )
              as Map<String, dynamic>;
      final applinks = association['applinks'] as Map<String, dynamic>;
      final details =
          (applinks['details'] as List<dynamic>).single as Map<String, dynamic>;
      final components = details['components'] as List<dynamic>;

      expect(details['appIDs'], ['B6YZZCUZWS.social.craftsky.app']);
      expect(
        components.cast<Map<String, dynamic>>().map(
          (component) => component['/'],
        ),
        ['/auth/complete', '/account-deletion/reauth-complete'],
      );
    },
  );

  test('browser fallbacks do not execute scripts or leak referrers', () {
    for (final path in [
      '../verified-links/auth/complete.html',
      '../verified-links/account-deletion/reauth-complete.html',
    ]) {
      final page = File(path).readAsStringSync();
      expect(page, contains('<meta name="referrer" content="no-referrer">'));
      expect(page, isNot(contains('<script')));
      expect(page, isNot(contains('http://')));
      expect(page, isNot(contains('https://')));
    }
  });
}
