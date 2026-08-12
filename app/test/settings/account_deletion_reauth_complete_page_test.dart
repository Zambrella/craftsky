import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/settings/pages/account_deletion_reauth_complete_page.dart';
import 'package:craftsky_app/settings/providers/account_deletion_controller.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _ReadyAccountDeletionController extends AccountDeletionController {
  @override
  FutureOr<void> build() => null;

  @override
  bool canComplete(String jobId) => true;

  @override
  String? requiredHandle(String jobId) => '@alice.test';
}

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

  testWidgets('uses CraftSky themed confirmation controls', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          accountDeletionControllerProvider.overrideWith(
            _ReadyAccountDeletionController.new,
          ),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const AccountDeletionReauthCompletePage(
            jobId: '10000000-0000-0000-0000-000000000001',
            proof: 'one-time-proof',
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byType(BrandTextField), findsOneWidget);
    final prompt = tester.widget<Text>(
      find.byWidgetPredicate(
        (widget) =>
            widget is Text &&
            widget.textSpan?.toPlainText() ==
                'Type @alice.test exactly to permanently delete this '
                    'CraftSky account.',
      ),
    );
    final handleSpan = (prompt.textSpan! as TextSpan).children!
        .whereType<TextSpan>()
        .singleWhere((span) => span.text == '@alice.test');
    expect(handleSpan.style?.fontWeight, FontWeight.bold);
    final button = tester.widget<ChunkyButton>(
      find.widgetWithText(ChunkyButton, 'Delete account'),
    );
    expect(button.backgroundColor, AppTheme.lightThemeData.colorScheme.error);
    expect(button.onPressed, isNull);

    await tester.enterText(find.byType(BrandTextField), '@alice.test');
    await tester.pump();

    final enabledButton = tester.widget<ChunkyButton>(
      find.widgetWithText(ChunkyButton, 'Delete account'),
    );
    expect(enabledButton.onPressed, isNotNull);
  });
}
