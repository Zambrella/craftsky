import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/providers/deletion_status_registry_provider.dart'
    show deletionStatusRegistryStorageProvider;
import 'package:craftsky_app/auth/providers/deletion_status_registry_storage.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/settings/pages/account_deletion_status_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

final class _Storage implements DeletionStatusRegistryStorage {
  _Storage(this.value);

  DeletionStatusRegistry value;

  @override
  Future<DeletionStatusRegistry> read() async => value;

  @override
  Future<void> write(DeletionStatusRegistry registry) async => value = registry;
}

void main() {
  testWidgets('attention status shows coarse copy and authorized actions', (
    tester,
  ) async {
    final storage = _Storage(
      DeletionStatusRegistry.empty().upsert(
        DeletionStatusEntry.pending(
          jobId: '10000000-0000-0000-0000-000000000001',
          did: 'did:plc:alice',
          handle: 'alice.test',
          statusToken: 'status-token',
        ).withStatus(
          status: AccountDeletionStatus.needsAttention,
          phase: AccountDeletionPhase.removingCraftskyRecords,
          canRetry: true,
          needsReauthentication: true,
        ),
      ),
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          deletionStatusRegistryStorageProvider.overrideWithValue(storage),
        ],
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: AccountDeletionStatusPage(
            jobId: '10000000-0000-0000-0000-000000000001',
            autoRefresh: false,
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Deletion needs attention'), findsOneWidget);
    expect(find.text('Removing CraftSky records'), findsOneWidget);
    expect(find.text('Retry'), findsOneWidget);
    expect(find.text('Reauthenticate'), findsOneWidget);
    expect(find.text('Get support'), findsOneWidget);
    expect(find.textContaining('did:plc:'), findsNothing);
    expect(find.textContaining('social.craftsky'), findsNothing);
  });
}
