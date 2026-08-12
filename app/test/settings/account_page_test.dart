import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/settings/pages/account_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/auth_session_fakes.dart';

void main() {
  testWidgets('Delete account requires both warning and exact typed handle', (
    tester,
  ) async {
    String? confirmedHandle;
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
        ],
        child: MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: AccountPage(
            onDeleteConfirmed: (handle) async => confirmedHandle = handle,
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Delete account'));
    await tester.pumpAndSettle();
    expect(find.text('Delete CraftSky account?'), findsOneWidget);
    expect(
      find.textContaining('all your CraftSky data from your PDS'),
      findsOneWidget,
    );
    expect(
      find.textContaining('won’t delete your PDS, DID'),
      findsOneWidget,
    );

    await tester.tap(find.text('Continue'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), '@Test.bsky.social');
    await tester.pump();
    FilledButton deleteButton() => tester.widget<FilledButton>(
      find.widgetWithText(FilledButton, 'Delete account'),
    );
    expect(deleteButton().onPressed, isNull);

    await tester.enterText(find.byType(TextField), '@test.bsky.social');
    await tester.pump();
    expect(deleteButton().onPressed, isNotNull);
    await tester.tap(find.widgetWithText(FilledButton, 'Delete account'));
    await tester.pumpAndSettle();
    expect(confirmedHandle, '@test.bsky.social');
  });
}
