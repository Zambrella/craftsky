import 'dart:io';

import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_id.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('Sentry and auth secrets are not committed in app source/config', () {
    final paths = [
      'pubspec.yaml',
      ...Directory(
        'lib',
      ).listSync(recursive: true).whereType<File>().map((file) => file.path),
    ];
    final forbidden = RegExp(
      r'(SENTRY_AUTH_TOKEN\s*=|Authorization:\s*Bearer|Cookie:\s*|pds[_-]?token|appview[_-]?session[_-]?token)',
      caseSensitive: false,
    );
    final offenders = <String>[];

    for (final path in paths) {
      final text = File(path).readAsStringSync();
      if (forbidden.hasMatch(text)) offenders.add(path);
    }

    expect(offenders, isEmpty);
  });

  test('REG-002 notification source has no direct diagnostic sink', () {
    final files = Directory(
      'lib/notifications',
    ).listSync(recursive: true).whereType<File>();
    final forbiddenSink = RegExp(
      r'\b(print|debugPrint|log)\s*\(|Sentry\.|addBreadcrumb\s*\(|captureException\s*\(|analytics\.',
    );

    final offenders = [
      for (final file in files)
        if (forbiddenSink.hasMatch(file.readAsStringSync())) file.path,
    ];

    expect(offenders, isEmpty);
  });

  test('REG-002 notification stringification redacts IDs and payload copy', () {
    const notificationSentinel = '018f47a2-4b0e-7f39-a621-9f6f6c75e312';
    const routingSentinel = 'routing_sentinel';
    const titleSentinel = 'private-title-sentinel';
    const bodySentinel = 'private-body-sentinel';
    final event = NotificationOpenEvent(
      notificationId: NotificationId.parse(notificationSentinel),
      category: NotificationCategory.reply,
      accountSubscriptionId: AccountSubscriptionId.parse(routingSentinel),
      source: NotificationOpenSource.foregroundBanner,
    );
    final foregroundEvent = ForegroundNotificationEvent(
      title: titleSentinel,
      body: bodySentinel,
      openEvent: event,
    );

    final diagnostics = '$event $foregroundEvent';
    for (final sentinel in [
      notificationSentinel,
      routingSentinel,
      titleSentinel,
      bodySentinel,
    ]) {
      expect(diagnostics, isNot(contains(sentinel)));
    }
  });
}
