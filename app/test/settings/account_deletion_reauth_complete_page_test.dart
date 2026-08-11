import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/settings/pages/account_deletion_reauth_complete_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('back cancels an unsubmitted deletion intent', (tester) async {
    var cancelCalls = 0;
    await tester.pumpWidget(
      ProviderScope(
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
