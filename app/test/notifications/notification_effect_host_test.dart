import 'dart:async';

import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_effect.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/models/notification_resolution.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_permission_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_runtime_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_service_provider.dart';
import 'package:craftsky_app/notifications/services/foreground_notification_handler.dart';
import 'package:craftsky_app/notifications/services/notification_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:craftsky_app/notifications/services/notification_runtime.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:craftsky_app/notifications/services/notification_service_owner.dart';
import 'package:craftsky_app/notifications/widgets/notification_effect_host.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/auth_session_fakes.dart';
import '../fakes/recording_messenger.dart';

void main() {
  testWidgets(
    'IT-008 signed-out resume skips count and refreshes permission cache',
    (tester) async {
      final service = _FakeNotificationService();
      final newness = _RecordingNewnessRepository();
      final effects = StreamController<NotificationEffect>.broadcast();
      final runtime = _runtime(service, effects);
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedOutAuthSession.new),
          notificationServiceProvider.overrideWithValue(service),
          notificationNewnessRepositoryProvider.overrideWithValue(newness),
          notificationRuntimeProvider.overrideWithValue(runtime),
        ],
      );
      addTearDown(effects.close);
      addTearDown(runtime.dispose);

      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: const MaterialApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: NotificationEffectHost(child: Text('Welcome')),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(
        await container.read(notificationPermissionProvider.future),
        NotificationPermission.denied,
      );
      service.permission = NotificationPermission.authorized;

      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await tester.pumpAndSettle();

      expect(newness.countCalls, 0);
      expect(
        await container.read(notificationPermissionProvider.future),
        NotificationPermission.authorized,
      );
    },
  );

  testWidgets('IT-008 ready resume refreshes count exactly once', (
    tester,
  ) async {
    final service = _FakeNotificationService();
    final newness = _RecordingNewnessRepository();
    final effects = StreamController<NotificationEffect>.broadcast();
    final runtime = _runtime(service, effects);
    final container = ProviderContainer.test(
      overrides: [
        authSessionProvider.overrideWith(SignedInAuthSession.new),
        onboardingStatusProvider.overrideWith2(
          (_) => CompletedOnboardingStatus(),
        ),
        notificationServiceProvider.overrideWithValue(service),
        notificationNewnessRepositoryProvider.overrideWithValue(newness),
        notificationRuntimeProvider.overrideWithValue(runtime),
      ],
    );
    addTearDown(effects.close);
    addTearDown(runtime.dispose);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: NotificationEffectHost(child: Text('Feed')),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await container.read(notificationNewCountProvider.future);
    final countBeforeResume = newness.countCalls;

    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await tester.pumpAndSettle();

    expect(newness.countCalls, countBeforeResume + 1);
  });

  testWidgets(
    'IT-007 root host presents and refreshes every foreground event',
    (tester) async {
      final service = _FakeNotificationService();
      final messenger = RecordingMessenger();
      final effects = StreamController<NotificationEffect>.broadcast();
      var listInvalidations = 0;
      var countRefreshes = 0;
      final runtime = _runtime(
        service,
        effects,
        invalidateList: () => listInvalidations++,
        refreshCount: () => countRefreshes++,
      );
      await runtime.start();
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedOutAuthSession.new),
          notificationServiceProvider.overrideWithValue(service),
          notificationRuntimeProvider.overrideWithValue(runtime),
          notificationEffectStreamProvider.overrideWithValue(effects.stream),
        ],
      );
      addTearDown(effects.close);
      addTearDown(runtime.dispose);

      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MessengerScope(
            messenger: messenger,
            child: const MaterialApp(
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: NotificationEffectHost(child: Text('Welcome')),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      final event = ForegroundNotificationEvent(
        title: 'New activity',
        body: 'Someone replied',
        openEvent: NotificationOpenEvent(
          notificationId: NotificationId.parse(
            '00000000-0000-4000-8000-000000000001',
          ),
          category: NotificationCategory.reply,
          accountSubscriptionId: AccountSubscriptionId.parse('binding'),
          source: NotificationOpenSource.foregroundBanner,
        ),
      );

      service
        ..emitForeground(event)
        ..emitForeground(event);
      await tester.pumpAndSettle();

      expect(
        messenger.calls.where((call) => call.$1 == 'info'),
        hasLength(2),
      );
      expect(messenger.calls[0].$2, 'New activity\nSomeone replied');
      expect(messenger.calls[0].$3?.label, 'Open');
      expect(listInvalidations, 2);
      expect(countRefreshes, 2);
    },
  );
}

NotificationRuntime _runtime(
  NotificationService service,
  StreamController<NotificationEffect> effects, {
  void Function()? invalidateList,
  void Function()? refreshCount,
}) {
  final registration = NotificationRegistrationCoordinator(
    platform: NotificationPlatform.ios,
    getToken: service.getToken,
    register: ({required platform, required token}) async =>
        AccountSubscriptionId.parse('binding'),
    saveBinding: ({required did, required binding}) async {},
  );
  late final NotificationRuntime runtime;
  return runtime = NotificationRuntime(
    coordinator: NotificationCoordinator(
      service: service,
      registration: registration,
    ),
    owner: NotificationServiceOwner(
      service: service,
      onTokenRefresh: registration.onTokenRefresh,
      onForegroundEvent: (event) => runtime.receiveForegroundEvent(event),
      onOpen: (event) => runtime.receiveOpen(event),
    ),
    routingStorage: NotificationRoutingStorage(_MemoryRoutingBackend()),
    resolutionRepository: _UnavailableResolutionRepository(),
    foregroundHandler: ForegroundNotificationHandler(
      showBanner: (event) => effects.add(NotificationBannerEffect(event)),
      invalidateList: invalidateList ?? () {},
      refreshCount: refreshCount ?? () {},
    ),
    effects: effects,
  );
}

final class _FakeNotificationService implements NotificationService {
  NotificationPermission permission = NotificationPermission.denied;
  final _foregroundEvents =
      StreamController<ForegroundNotificationEvent>.broadcast();

  void emitForeground(ForegroundNotificationEvent event) =>
      _foregroundEvents.add(event);

  @override
  Future<void> deleteToken() async {}

  @override
  Future<void> dispose() => _foregroundEvents.close();

  @override
  Stream<ForegroundNotificationEvent> get foregroundEvents =>
      _foregroundEvents.stream;

  @override
  Future<NotificationPermission> getPermission() async => permission;

  @override
  Future<String?> getToken() async => 'token';

  @override
  Future<void> initialize() async {}

  @override
  Stream<NotificationOpenEvent> get openedNotifications => const Stream.empty();

  @override
  Future<void> openSystemNotificationSettings() async {}

  @override
  Future<NotificationPermission> requestPermission() async => permission;

  @override
  Future<NotificationOpenEvent?> takeInitialOpen() async => null;

  @override
  Stream<String> get tokenRefreshes => const Stream.empty();
}

final class _RecordingNewnessRepository
    implements NotificationNewnessRepository {
  int countCalls = 0;

  @override
  Future<int> count() async {
    countCalls++;
    return 0;
  }

  @override
  Future<void> markSeen() async {}
}

final class _UnavailableResolutionRepository
    implements NotificationResolutionRepository {
  @override
  Future<NotificationResolution> resolve(NotificationId id) =>
      throw StateError('not used');
}

final class _MemoryRoutingBackend implements NotificationRoutingStorageBackend {
  String? value;

  @override
  Future<void> delete() async => value = null;

  @override
  Future<String?> read() async => value;

  @override
  Future<void> write(String value) async => this.value = value;
}
