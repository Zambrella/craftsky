import 'package:craftsky_app/account_eligibility/pages/account_eligibility_page.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AT-011 exposes retained safety and account controls', (
    tester,
  ) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: AccountEligibilityPage(appealGuidance: null),
        ),
      ),
    );

    expect(find.text('View decision and appeal options'), findsOneWidget);
    expect(find.text('Muted accounts'), findsOneWidget);
    expect(find.text('Blocked accounts'), findsOneWidget);
    expect(find.text('Privacy policy'), findsOneWidget);
    expect(find.text('Manage account'), findsOneWidget);
  });
}
