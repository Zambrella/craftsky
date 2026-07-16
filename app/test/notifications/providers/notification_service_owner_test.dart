import 'dart:async';

import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:craftsky_app/notifications/services/notification_service_owner.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-012 / IT-010 / AT-009 owns service streams exactly once', () async {
    final service = _RecordingService(initialOpen: _open('initial'));
    final tokens = <String>[];
    final events = <ForegroundNotificationEvent>[];
    final opens = <NotificationOpenEvent>[];
    final owner = NotificationServiceOwner(
      service: service,
      onTokenRefresh: (value) async => tokens.add(value),
      onForegroundEvent: (value) async => events.add(value),
      onOpen: (value) async => opens.add(value),
    );

    await owner.start();
    await owner.start();
    expect(service.initializeCalls, 1);
    expect(service.initialOpenCalls, 1);
    expect(opens, hasLength(1));

    service.tokenController.add('token');
    service.eventController.add(_event('foreground'));
    service.openController.add(_open('background'));
    await Future<void>.delayed(Duration.zero);

    expect(tokens, ['token']);
    expect(events, hasLength(1));
    expect(opens, hasLength(2));

    await owner.dispose();
    service.tokenController.add('ignored');
    service.openController.add(_open('ignored'));
    await Future<void>.delayed(Duration.zero);

    expect(service.disposeCalls, 1);
    expect(tokens, ['token']);
    expect(opens, hasLength(2));
    await service.close();
  });
}

NotificationOpenEvent _open(String suffix) => NotificationOpenEvent(
  notificationId: NotificationId.parse(
    suffix == 'initial'
        ? '00000000-0000-0000-0000-000000000001'
        : suffix == 'background'
        ? '00000000-0000-0000-0000-000000000002'
        : '00000000-0000-0000-0000-000000000003',
  ),
  category: NotificationCategory.like,
  accountSubscriptionId: AccountSubscriptionId.parse('binding'),
  source: NotificationOpenSource.backgroundOpen,
);

ForegroundNotificationEvent _event(String suffix) =>
    ForegroundNotificationEvent(
      title: 'Title',
      body: 'Body',
      openEvent: _open(suffix),
    );

final class _RecordingService implements NotificationService {
  _RecordingService({this.initialOpen});

  final NotificationOpenEvent? initialOpen;
  final tokenController = StreamController<String>.broadcast();
  final eventController =
      StreamController<ForegroundNotificationEvent>.broadcast();
  final openController = StreamController<NotificationOpenEvent>.broadcast();
  int initializeCalls = 0;
  int initialOpenCalls = 0;
  int disposeCalls = 0;

  Future<void> close() async {
    await tokenController.close();
    await eventController.close();
    await openController.close();
  }

  @override
  Future<void> deleteToken() async {}

  @override
  Future<void> dispose() async => disposeCalls++;

  @override
  Stream<ForegroundNotificationEvent> get foregroundEvents =>
      eventController.stream;

  @override
  Future<NotificationPermission> getPermission() async =>
      NotificationPermission.authorized;

  @override
  Future<String?> getToken() async => null;

  @override
  Future<void> initialize() async => initializeCalls++;

  @override
  Stream<NotificationOpenEvent> get openedNotifications =>
      openController.stream;

  @override
  Future<void> openSystemNotificationSettings() async {}

  @override
  Future<NotificationPermission> requestPermission() async =>
      NotificationPermission.authorized;

  @override
  Future<NotificationOpenEvent?> takeInitialOpen() async {
    initialOpenCalls++;
    return initialOpen;
  }

  @override
  Stream<String> get tokenRefreshes => tokenController.stream;
}
