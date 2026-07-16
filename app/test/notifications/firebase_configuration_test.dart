import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'AT-001 / REG-006 uses the shared native identity and Firebase project',
    () {
      final androidConfig =
          jsonDecode(
                File('android/app/google-services.json').readAsStringSync(),
              )
              as Map<String, dynamic>;
      final projectInfo = androidConfig['project_info'] as Map<String, dynamic>;
      final clients = androidConfig['client'] as List<dynamic>;
      final androidClient = clients.single as Map<String, dynamic>;
      final clientInfo = androidClient['client_info'] as Map<String, dynamic>;
      final androidInfo =
          clientInfo['android_client_info'] as Map<String, dynamic>;
      final iosConfig = File(
        'ios/Runner/GoogleService-Info.plist',
      ).readAsStringSync();

      expect(projectInfo['project_id'], 'craftsky-app');
      expect(androidInfo['package_name'], 'social.craftsky.app');
      expect(iosConfig, contains('<string>craftsky-app</string>'));
      expect(iosConfig, contains('<string>social.craftsky.app</string>'));

      final androidGradle = File(
        'android/app/build.gradle.kts',
      ).readAsStringSync();
      final iosProject = File(
        'ios/Runner.xcodeproj/project.pbxproj',
      ).readAsStringSync();
      expect(androidGradle, contains('namespace = "social.craftsky.app"'));
      expect(androidGradle, contains('applicationId = "social.craftsky.app"'));
      expect(
        iosProject,
        contains('PRODUCT_BUNDLE_IDENTIFIER = social.craftsky.app;'),
      );
    },
  );

  test('REG-006 binds one Android channel and enables iOS capabilities', () {
    final manifest = File(
      'android/app/src/main/AndroidManifest.xml',
    ).readAsStringSync();
    final strings = File(
      'android/app/src/main/res/values/strings.xml',
    ).readAsStringSync();
    final activity = File(
      'android/app/src/main/kotlin/social/craftsky/app/MainActivity.kt',
    ).readAsStringSync();
    final entitlements = File(
      'ios/Runner/Runner.entitlements',
    ).readAsStringSync();
    final info = File('ios/Runner/Info.plist').readAsStringSync();

    expect(manifest, contains('android.permission.POST_NOTIFICATIONS'));
    expect(
      manifest,
      contains('com.google.firebase.messaging.default_notification_channel_id'),
    );
    expect(manifest, contains('@string/default_notification_channel_id'));
    expect(strings, contains('name="default_notification_channel_id"'));
    expect(strings, contains('CraftSky notifications'));
    expect(activity, contains('NotificationManager.IMPORTANCE_DEFAULT'));
    expect(activity, contains('setSound'));
    expect(activity, contains('enableVibration(true)'));
    expect(entitlements, contains('<key>aps-environment</key>'));
    expect(info, contains('<string>remote-notification</string>'));
  });

  test('REG-001 and REG-005 keep Firebase inside the approved boundary', () {
    final dartFiles = Directory('lib')
        .listSync(recursive: true)
        .whereType<File>()
        .where((file) => file.path.endsWith('.dart'));
    for (final file in dartFiles) {
      final source = file.readAsStringSync();
      if (!source.contains('package:firebase_')) continue;
      expect(
        file.path == 'lib/firebase_options.dart' ||
            file.path.contains('/services/firebase_notification_'),
        isTrue,
        reason: 'Firebase import escaped adapter boundary: ${file.path}',
      );
    }

    final handler = File(
      'lib/notifications/services/firebase_notification_background_handler.dart',
    ).readAsStringSync();
    expect(handler, contains("@pragma('vm:entry-point')"));
    expect(
      handler,
      contains('Future<void> firebaseMessagingBackgroundHandler'),
    );
    expect(handler, isNot(contains('riverpod')));
    expect(handler, isNot(contains('go_router')));
    expect(handler, isNot(contains('Logger')));
  });

  test('UT-016 / AT-012 keeps provider presentation bounded', () {
    final adapter = File(
      'lib/notifications/services/firebase_notification_service.dart',
    ).readAsStringSync();

    expect(
      adapter,
      contains(
        'setForegroundNotificationPresentationOptions(\n'
        '        alert: false,\n'
        '        badge: false,\n'
        '        sound: false,',
      ),
    );
    expect(
      adapter,
      contains(
        'requestPermission(\n'
        '      alert: true,\n'
        '      badge: false,\n'
        '      sound: true,',
      ),
    );
    expect(adapter, isNot(contains('localNotification')));
    expect(adapter, isNot(contains('vibration')));
  });
}
