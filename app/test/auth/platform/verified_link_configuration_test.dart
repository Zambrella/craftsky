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
}
