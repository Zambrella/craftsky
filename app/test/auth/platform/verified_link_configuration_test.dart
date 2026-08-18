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
}
