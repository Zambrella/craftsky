import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/active_account_initialization.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/auth/widgets/active_account_initialization_gate.dart';
import 'package:craftsky_app/initialization_loading_screen.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/service_status/widgets/service_status_host.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'announcement_dismissal_test.dart' show FakeDismissalStore, announcement;
import 'service_status_controller_test.dart'
    show FakeStatusClock, FakeStatusRepository, maintenance;

class StatusRegistryStorage implements SessionRegistryStorage {
  StatusRegistryStorage(this.value);
  SessionRegistry value;
  @override
  Future<SessionRegistry> read() async => value;
  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

void main() {
  // AT-006 / FR-004, FR-008, FR-009, RULE-004 / AC-007, AC-011, AC-012.
  for (final transition in ['normal', 'announcement', 'expiry']) {
    testWidgets('$transition clears cover through the current account gate', (
      tester,
    ) async {
      final repository = FakeStatusRepository();
      final clock = FakeStatusClock();
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'bob-token',
        did: 'did:plc:bob',
        handle: 'bob.test',
      );
      final storage = StatusRegistryStorage(registry);
      final initialization = Completer<ActiveAccountInitialization?>();
      final container = ProviderContainer(
        overrides: [
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          serviceStatusClockProvider.overrideWithValue(clock),
          announcementDismissalStoreProvider.overrideWithValue(
            FakeDismissalStore(),
          ),
          secureSessionRegistryStorageProvider.overrideWithValue(storage),
          activeAccountInitializationProvider.overrideWith(
            (ref) => initialization.future,
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            builder: (context, child) => ServiceStatusHost(child: child!),
            home: const ActiveAccountInitializationGate(
              child: Text('Allowed Bob'),
            ),
          ),
        ),
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      final covered = controller.refresh();
      repository.requests.last.complete(maintenance);
      await tester.pump();
      await covered;
      expect(find.text('Maintenance'), findsOneWidget);
      if (transition == 'expiry') {
        clock.advance(const Duration(minutes: 5));
        await tester.pump(const Duration(minutes: 5));
      } else {
        final clear = controller.refresh();
        repository.requests.last.complete(
          transition == 'normal'
              ? const ServiceStatusDocument(
                  mode: ServiceStatusMode.normal,
                  revision: 'B',
                )
              : announcement('B'),
        );
        await tester.pump();
        await clear;
      }
      expect(find.text('Maintenance'), findsNothing);
      expect(find.byType(InitializationLoadingScreen), findsOneWidget);
      expect(find.text('Allowed Bob'), findsNothing);
      expect(
        container.read(sessionRegistryProvider).requireValue,
        same(registry),
      );
      initialization.complete(
        ActiveAccountInitialization(
          lease: registry.activeLease!,
          languagePreferences: const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
          onboardingComplete: true,
        ),
      );
      await tester.pump();
      await tester.pump();
      expect(find.text('Allowed Bob'), findsOneWidget);
      expect(find.text('Service recovered'), findsNothing);
      await tester.pumpWidget(const SizedBox.shrink());
      container.dispose();
    });
  }
  // REG-002 / FR-009, RULE-004 / AC-012.
  testWidgets(
    'old account completion under cover cannot reveal current-account content',
    (tester) async {
      final repository = FakeStatusRepository();
      final registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'bob-token',
            did: 'did:plc:bob',
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'alice-token',
            did: 'did:plc:alice',
            handle: 'alice.test',
          );
      final alice = Completer<ActiveAccountInitialization?>();
      final bob = Completer<ActiveAccountInitialization?>();
      final container = ProviderContainer(
        overrides: [
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          announcementDismissalStoreProvider.overrideWithValue(
            FakeDismissalStore(),
          ),
          secureSessionRegistryStorageProvider.overrideWithValue(
            StatusRegistryStorage(registry),
          ),
          activeAccountInitializationProvider.overrideWith((ref) {
            final current = ref.watch(sessionRegistryProvider).requireValue;
            return current.activeDid?.value == 'did:plc:alice'
                ? alice.future
                : bob.future;
          }),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            builder: (context, child) => ServiceStatusHost(child: child!),
            home: const ActiveAccountInitializationGate(
              child: Text('Current account allowed'),
            ),
          ),
        ),
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      final covered = controller.refresh();
      repository.requests.last.complete(maintenance);
      await tester.pump();
      await covered;
      await container
          .read(sessionRegistryProvider.notifier)
          .activate(registry.leaseFor(AccountKey('did:plc:bob'))!);
      await tester.pump();
      alice.complete(
        ActiveAccountInitialization(
          lease: registry.activeLease!,
          languagePreferences: const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
          onboardingComplete: true,
        ),
      );
      await tester.pump();
      final clear = controller.refresh();
      repository.requests.last.complete(
        const ServiceStatusDocument(
          mode: ServiceStatusMode.normal,
          revision: 'B',
        ),
      );
      await tester.pump();
      await clear;
      expect(find.byType(InitializationLoadingScreen), findsOneWidget);
      expect(find.text('Current account allowed'), findsNothing);
      final current = container.read(sessionRegistryProvider).requireValue;
      expect(current.activeDid?.value, 'did:plc:bob');
      bob.complete(
        ActiveAccountInitialization(
          lease: current.activeLease!,
          languagePreferences: const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
          onboardingComplete: true,
        ),
      );
      await tester.pump();
      await tester.pump();
      expect(find.text('Current account allowed'), findsOneWidget);
      await tester.pumpWidget(const SizedBox.shrink());
      container.dispose();
    },
  );
}
