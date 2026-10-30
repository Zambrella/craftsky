import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/onboarding/models/onboarding_completion.dart';
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

import 'announcement_dismissal_test.dart' show FakeDismissalStore;
import 'message_auth_visibility_test.dart' show MessageAuthSession;
import 'service_status_controller_test.dart' show FakeStatusRepository;
import 'service_status_recovery_test.dart' show StatusRegistryStorage;

class MessageOnboarding extends OnboardingStatus {
  @override
  Future<OnboardingCompletion> build(AccountSessionLease lease) async =>
      const OnboardingCompletion(completed: false);
  AsyncValue<OnboardingCompletion> get completion => state;
  set completion(AsyncValue<OnboardingCompletion> value) => state = value;
}

void main() {
  // D-017 / FR-007, FR-008 / AC-010, AC-012.
  testWidgets(
    'message waits for current account onboarding without dismissal',
    (tester) async {
      final repository = FakeStatusRepository();
      final store = FakeDismissalStore();
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'test-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final container = ProviderContainer(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            StatusRegistryStorage(registry),
          ),
          authSessionProvider.overrideWith(MessageAuthSession.new),
          onboardingStatusProvider.overrideWith2((_) => MessageOnboarding()),
          statusRuntimeVersionProvider.overrideWith(
            (ref) async => StatusAppVersion.parse('1.8.0'),
          ),
          statusPlatformProvider.overrideWithValue(StatusPlatform.ios),
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          announcementDismissalStoreProvider.overrideWithValue(store),
        ],
      );
      addTearDown(container.dispose);
      await container.read(sessionRegistryProvider.future);
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            builder: (_, child) => ServiceStatusHost(child: child!),
            home: const Scaffold(body: Text('Onboarding content')),
          ),
        ),
      );
      await tester.pump();
      final auth =
          (container.read(authSessionProvider.notifier) as MessageAuthSession)
            ..authState = AsyncData(
              SignedIn(did: 'did:plc:alice', handle: 'alice.test'),
            );
      final pending = controller.refresh();
      repository.requests.last.complete(
        const ServiceStatusDocument(
          revision: 'D',
          message: StatusMessage(
            revision: 'A',
            minVersion: '0.0.0',
            title: 'News',
            description: 'Body',
          ),
        ),
      );
      await tester.pump();
      await pending;
      expect(find.text('News'), findsNothing);
      expect(store.revision, isNull);
      final onboarding =
          (container.read(
              onboardingStatusProvider(registry.activeLease!.session).notifier,
            ) as MessageOnboarding)
            ..completion = const AsyncData(
              OnboardingCompletion(completed: true),
            );
      await tester.pump();
      expect(find.text('News'), findsOneWidget);
      for (final completion in <AsyncValue<OnboardingCompletion>>[
        const AsyncLoading(),
        AsyncError(StateError('onboarding unavailable'), StackTrace.current),
        const AsyncData(OnboardingCompletion(completed: false)),
      ]) {
        onboarding.completion = completion;
        await tester.pump();
        expect(find.text('News'), findsNothing);
        expect(store.revision, isNull);
      }
      onboarding.completion = const AsyncData(
        OnboardingCompletion(completed: true),
      );
      await tester.pump();
      expect(find.text('News'), findsOneWidget);

      final aliceLease = registry.activeLease!.session;
      container.invalidate(onboardingStatusProvider(aliceLease));
      expect(
        container.read(onboardingStatusProvider(aliceLease)).isLoading,
        isTrue,
      );
      expect(container.read(statusMessageAudienceReadyProvider), isFalse);
      await tester.pumpAndSettle();
      expect(find.text('News'), findsNothing);
      final aliceOnboarding = container.read(
        onboardingStatusProvider(aliceLease).notifier,
      ) as MessageOnboarding;
      final keepAlice = container.listen(
        onboardingStatusProvider(aliceLease),
        (_, _) {},
      );
      aliceOnboarding.completion = const AsyncData(
        OnboardingCompletion(completed: true),
      );
      await tester.pump();
      expect(find.text('News'), findsOneWidget);

      await container
          .read(sessionRegistryProvider.notifier)
          .upsertAndActivate(
            token: 'bob-token',
            did: 'did:plc:bob',
            handle: 'bob.test',
          );
      await tester.pumpAndSettle();
      expect(find.text('News'), findsNothing);
      auth.authState = AsyncData(
        SignedIn(did: 'did:plc:bob', handle: 'bob.test'),
      );
      await tester.pumpAndSettle();
      expect(find.text('News'), findsNothing);
      // A late result for Alice cannot qualify the newly active Bob.
      aliceOnboarding.completion = const AsyncData(
        OnboardingCompletion(completed: true),
      );
      await tester.pump();
      expect(find.text('News'), findsNothing);
      final bobLease = container
          .read(sessionRegistryProvider)
          .requireValue
          .activeLease!
          .session;
      (container.read(
        onboardingStatusProvider(bobLease).notifier,
      ) as MessageOnboarding).completion = const AsyncData(
        OnboardingCompletion(completed: true),
      );
      await tester.pump();
      expect(find.text('News'), findsOneWidget);
      expect(store.revision, isNull);
      expect(repository.requests, hasLength(1));
      keepAlice.close();
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
}
