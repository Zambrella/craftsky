import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:craftsky_app/auth/providers/account_transition_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/auth/widgets/account_transition_overlay.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.registry);
  final SessionRegistry registry;

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry registry) async {}
}

void main() {
  testWidgets('UT-018 transition overlay blocks old account interaction', (
    tester,
  ) async {
    final registry = SessionRegistry.empty()
        .upsertAndActivate(
          token: 'alice-token',
          did: 'did:plc:alice',
          handle: 'alice.test',
          cachedDisplayName: 'Alice',
        )
        .upsertAndActivate(
          token: 'bob-token',
          did: 'did:plc:bob',
          handle: 'bob.test',
        );
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _RegistryStorage(registry),
        ),
      ],
    );
    var taps = 0;
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: AccountTransitionOverlay(
            child: Scaffold(
              body: TextButton(
                onPressed: () => taps++,
                child: const Text('Old'),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final target = registry.leaseFor(AccountKey('did:plc:alice'))!;
    container.read(accountTransitionStateProvider.notifier).transition =
        AccountTransition(target);
    await tester.pump();
    await container.read(sessionRegistryProvider.future);
    await tester.pump();

    expect(find.text('Alice'), findsOneWidget);
    expect(find.byIcon(Icons.person), findsOneWidget);
    expect(find.bySemanticsLabel('Switching account'), findsOneWidget);
    await tester.tap(find.text('Old'), warnIfMissed: false);
    expect(taps, 0);

    container.read(accountTransitionStateProvider.notifier).transition = null;
    await tester.pump();
    await tester.tap(find.text('Old'));
    expect(taps, 1);
  });
}
