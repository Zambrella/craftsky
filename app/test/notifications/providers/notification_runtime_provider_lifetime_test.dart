import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_runtime_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_service_provider.dart';
import 'package:craftsky_app/notifications/services/notification_registration_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:craftsky_app/onboarding/data/onboarding_repository.dart';
import 'package:craftsky_app/onboarding/models/onboarding_completion.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_repository_provider.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'device registration owns its account repository until completion',
    () async {
      final initial = SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final storage = _Storage(initial);
      final lease = initial.activeLease!.session;
      final started = Completer<void>();
      final finish = Completer<AccountSubscriptionId>();
      final repository = _DeviceRepository(started, finish.future);
      var disposed = false;
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(storage),
          notificationServiceProvider.overrideWithValue(_AuthorizedService()),
          onboardingRepositoryProvider.overrideWith(
            (ref, lease) async => _CompletedOnboarding(),
          ),
          accountNotificationDeviceRepositoryProvider.overrideWith((
            ref,
            account,
          ) async {
            ref.onDispose(() => disposed = true);
            return repository;
          }),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final onboarding = onboardingStatusProvider(lease);
      final subscription = container.listen(onboarding, (_, _) {});
      addTearDown(subscription.close);
      await container.read(onboarding.future);
      final runtime = container.read(notificationRuntimeProvider);
      addTearDown(runtime.dispose);
      await runtime.start();
      final operation = runtime.updateReadiness(
        did: lease.account.did,
        onboarded: true,
      );
      try {
        await started.future;
        await container.pump();
        expect(
          disposed,
          isFalse,
          reason: 'registration still uses this repository',
        );
      } finally {
        finish.complete(AccountSubscriptionId.parse('fixture-binding'));
        await operation;
      }
      await container.pump();
      expect(disposed, isTrue);
      expect(
        storage.value.routingBindings[lease.account.did],
        'fixture-binding',
      );
    },
  );
}

final class _Storage implements SessionRegistryStorage {
  new(this.value);
  SessionRegistry value;
  @override
  Future<SessionRegistry> read() async => value;
  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

final class _DeviceRepository implements NotificationDeviceRepository {
  new(this.started, this.result);
  final Completer<void> started;
  final Future<AccountSubscriptionId> result;
  @override
  Future<AccountSubscriptionId> register({
    required NotificationPlatform platform,
    required String token,
  }) {
    started.complete();
    return result;
  }
}

final class _CompletedOnboarding implements OnboardingRepository {
  @override
  Future<OnboardingCompletion> readStatus() async =>
      const OnboardingCompletion(completed: true);
  @override
  Future<OnboardingCompletion> complete({
    required bool meetsMinimumAge,
    required String policyVersion,
  }) async => const OnboardingCompletion(completed: true);
}

final class _AuthorizedService implements NotificationService {
  @override
  Future<void> initialize() async {}
  @override
  Future<void> dispose() async {}
  @override
  Future<NotificationPermission> getPermission() async =>
      NotificationPermission.authorized;
  @override
  Future<NotificationPermission> requestPermission() async =>
      NotificationPermission.authorized;
  @override
  Future<String?> getToken() async => 'fixture-token';
  @override
  Stream<String> get tokenRefreshes => const Stream.empty();
  @override
  Stream<ForegroundNotificationEvent> get foregroundEvents =>
      const Stream.empty();
  @override
  Stream<NotificationOpenAttempt> get openedNotifications =>
      const Stream.empty();
  @override
  Future<NotificationOpenAttempt?> takeInitialOpen() async => null;
  @override
  Future<void> deleteToken() async {}
  @override
  Future<void> openSystemNotificationSettings() async {}
}
