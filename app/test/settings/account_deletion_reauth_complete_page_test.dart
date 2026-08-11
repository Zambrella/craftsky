import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/providers/deletion_status_registry_provider.dart'
    show deletionStatusRegistryStorageProvider;
import 'package:craftsky_app/auth/providers/deletion_status_registry_storage.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/settings/pages/account_deletion_reauth_complete_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

final class _EmptyStatusStorage implements DeletionStatusRegistryStorage {
  @override
  Future<DeletionStatusRegistry> read() async => DeletionStatusRegistry.empty();

  @override
  Future<void> write(DeletionStatusRegistry registry) async {}
}

void main() {
  testWidgets('back cancels an unsubmitted deletion intent', (tester) async {
    var cancelCalls = 0;
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          deletionStatusRegistryStorageProvider.overrideWithValue(
            _EmptyStatusStorage(),
          ),
        ],
        child: MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => TextButton(
              onPressed: () => Navigator.push<void>(
                context,
                MaterialPageRoute(
                  builder: (_) => AccountDeletionReauthCompletePage(
                    jobId: '10000000-0000-0000-0000-000000000001',
                    proof: 'one-time-proof',
                    onCancel: () async => cancelCalls++,
                  ),
                ),
              ),
              child: const Text('Open confirmation'),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Open confirmation'));
    await tester.pumpAndSettle();

    await tester.pageBack();
    await tester.pumpAndSettle();

    expect(cancelCalls, 1);
  });
}
