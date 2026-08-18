import 'package:craftsky_app/notifications/services/flutter_local_notification_gateway.dart';
import 'package:craftsky_app/notifications/services/notification_local_presenter.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  AndroidFlutterLocalNotificationsPlugin.registerWith();

  const channel = MethodChannel('dexterous.com/flutter/local_notifications');
  final calls = <MethodCall>[];

  setUp(() {
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          calls.add(call);
          if (call.method == 'initialize') return true;
          if (call.method == 'getNotificationAppLaunchDetails') return null;
          return null;
        });
  });

  tearDown(() {
    calls.clear();
    debugDefaultTargetPlatformOverride = null;
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  test('IT-PUSH-017 sends the full tag and fixed ID to Android', () async {
    const notificationId = '00000000-0000-4000-8000-000000000040';
    final gateway = FlutterLocalNotificationGateway(
      plugin: FlutterLocalNotificationsPlugin(),
    );
    await gateway.initialize(onOpen: (_) {});
    await gateway.present(
      const NotificationPresentation(
        id: NotificationLocalPresenter.androidTypeId,
        tag: notificationId,
        title: 'Alice',
        body: 'liked your post',
        payload: '{"validated":true}',
      ),
    );

    final initialize = calls.singleWhere((call) => call.method == 'initialize');
    expect(
      Map<String, Object?>.from(initialize.arguments as Map)['defaultIcon'],
      'ic_stat_craftsky_notification',
    );
    final show = calls.singleWhere((call) => call.method == 'show');
    final arguments = Map<String, Object?>.from(show.arguments as Map);
    final platformSpecifics = Map<String, Object?>.from(
      arguments['platformSpecifics']! as Map,
    );
    expect(arguments['id'], NotificationLocalPresenter.androidTypeId);
    expect(arguments['payload'], '{"validated":true}');
    expect(platformSpecifics['tag'], notificationId);
    expect(
      platformSpecifics['channelId'],
      FlutterLocalNotificationGateway.channelId,
    );
  });
}
