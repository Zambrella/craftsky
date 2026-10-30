import 'dart:async';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/models/status_app_version.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/service_status/providers/status_runtime_provider.dart';
import 'package:craftsky_app/service_status/widgets/service_status_host.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../fakes/auth_session_fakes.dart' show CompletedOnboardingStatus;
import 'announcement_dismissal_test.dart' show FakeDismissalStore;
import 'service_status_controller_test.dart' show FakeStatusRepository;
import 'service_status_recovery_test.dart' show StatusRegistryStorage;

class MessageAuthSession extends AuthSession {
  @override
  Future<AuthState> build() async => const SignedOut();
  AsyncValue<AuthState> get authState => state;
  set authState(AsyncValue<AuthState> value) => state = value;
}

class PendingMessageAuthSession extends AuthSession {
  @override
  Future<AuthState> build() => Completer<AuthState>().future;
}

void main() {
  for (final maintenance in [false, true]) {
    testWidgets(
      '${maintenance ? 'maintenance' : 'update'} works before auth resolves',
      (tester) async {
        final repository = FakeStatusRepository();
        final container = ProviderContainer(
          overrides: [
            authSessionProvider.overrideWith(PendingMessageAuthSession.new),
            statusRuntimeVersionProvider.overrideWith(
              (ref) async => StatusAppVersion.parse('1.8.0'),
            ),
            statusPlatformProvider.overrideWithValue(StatusPlatform.ios),
            serviceStatusRepositoryProvider.overrideWithValue(repository),
            announcementDismissalStoreProvider.overrideWithValue(
              FakeDismissalStore(),
            ),
          ],
        );
        try {
          final controller = container.read(
            serviceStatusControllerProvider.notifier,
          )..start();
          expect(repository.requests, hasLength(1));
          repository.requests.single.complete(
            ServiceStatusDocument(
              revision: 'combined',
              maintenance: maintenance
                  ? const StatusMaintenance(title: 'Break', description: 'Soon')
                  : null,
              minimumVersions: const StatusMinimumVersions(ios: '1.9.0'),
              message: const StatusMessage(
                revision: 'A',
                minVersion: '0.0.0',
                title: 'News',
                description: 'Body',
              ),
            ),
          );
          await tester.pumpAndSettle();
          expect(container.read(authSessionProvider).isLoading, isTrue);
          expect(
            controller.presentation,
            maintenance
                ? StatusPresentation.maintenance
                : StatusPresentation.updateRequired,
          );
          expect(controller.blockingActive, isTrue);
          expect(controller.announcement, isNull);
          expect(
            container.read(serviceStatusControllerProvider).fetching,
            isFalse,
          );
        } finally {
          container.dispose();
        }
      },
    );
  }

  // D-016 / FR-007, FR-008 / AC-010, AC-012: auth suppression is not dismissal.
  testWidgets('messages follow resolved sign-in without losing dismissal', (
    tester,
  ) async {
    final repository = FakeStatusRepository();
    final store = FakeDismissalStore();
    final container = ProviderContainer(
      overrides: [
        authSessionProvider.overrideWith(MessageAuthSession.new),
        secureSessionRegistryStorageProvider.overrideWithValue(
          StatusRegistryStorage(
            SessionRegistry.empty().upsertAndActivate(
              token: 'test-token',
              did: 'did:plc:alice',
              handle: 'alice.test',
            ),
          ),
        ),
        onboardingStatusProvider.overrideWith2(
          (_) => CompletedOnboardingStatus(),
        ),
        statusRuntimeVersionProvider.overrideWith(
          (ref) async => StatusAppVersion.parse('1.8.0'),
        ),
        statusPlatformProvider.overrideWithValue(StatusPlatform.ios),
        serviceStatusRepositoryProvider.overrideWithValue(repository),
        announcementDismissalStoreProvider.overrideWithValue(store),
      ],
    );
    addTearDown(container.dispose);
    final controller = container.read(serviceStatusControllerProvider.notifier);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (_, child) => ServiceStatusHost(child: child!),
          home: const ServiceStatusBackGuard(
            child: Scaffold(body: Text('Underlying app')),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    Future<void> accept(String revision) async {
      final pending = controller.refresh();
      repository.requests.last.complete(
        ServiceStatusDocument(
          revision: 'D$revision',
          message: StatusMessage(
            revision: revision,
            minVersion: '0.0.0',
            title: 'Signed-in news',
            description: 'Body',
          ),
        ),
      );
      await tester.pumpAndSettle();
      await pending;
    }

    var auth =
        container.read(authSessionProvider.notifier) as MessageAuthSession;
    final signedIn = SignedIn(did: 'did:plc:alice', handle: 'alice.test');
    await accept('A');
    expect(find.text('Signed-in news'), findsNothing);
    expect(await controller.dismissAnnouncement(), isFalse);
    expect(await controller.activateMessage('A', () async => true), isFalse);
    expect(await tester.binding.handlePopRoute(), isFalse);
    expect(store.revision, isNull);
    auth.authState = AsyncData(signedIn);
    await tester.pumpAndSettle();
    expect(find.text('Signed-in news'), findsOneWidget);
    for (final hidden in <AsyncValue<AuthState>>[
      const AsyncData(SignedOut()),
      const AsyncLoading(),
      AsyncError(StateError('auth unavailable'), StackTrace.current),
    ]) {
      auth.authState = hidden;
      await tester.pumpAndSettle();
      expect(find.text('Signed-in news'), findsNothing);
      expect(controller.announcement, isNull);
      expect(store.revision, isNull);
      auth.authState = AsyncData(signedIn);
      await tester.pumpAndSettle();
      expect(find.text('Signed-in news'), findsOneWidget);
    }
    container.invalidate(authSessionProvider);
    final refreshing = container.read(authSessionProvider);
    expect(refreshing.isLoading, isTrue);
    expect(refreshing.value, isA<SignedIn>());
    expect(container.read(statusMessageAudienceReadyProvider), isFalse);
    await tester.pumpAndSettle();
    auth = (container.read(authSessionProvider.notifier) as MessageAuthSession)
      ..authState = AsyncData(signedIn);
    await tester.pumpAndSettle();
    expect(find.text('Signed-in news'), findsOneWidget);
    expect(
      repository.requests,
      hasLength(1),
      reason: 'Auth changes do not restart status requests',
    );
    await tester.tap(find.text('Dismiss'));
    await tester.pumpAndSettle();
    expect(store.revision, 'A');
    auth.authState = const AsyncData(SignedOut());
    await tester.pumpAndSettle();
    auth.authState = AsyncData(signedIn);
    await tester.pumpAndSettle();
    expect(find.text('Signed-in news'), findsNothing);
    auth.authState = const AsyncData(SignedOut());
    await tester.pumpAndSettle();
    await accept('B');
    expect(find.text('Signed-in news'), findsNothing);
    auth.authState = AsyncData(signedIn);
    await tester.pumpAndSettle();
    expect(find.text('Signed-in news'), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
  });
}
