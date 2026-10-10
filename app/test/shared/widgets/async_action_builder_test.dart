import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/shared/widgets/async_action_builder.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

enum _TestAction { save, other, delete }

void main() {
  testWidgets(
    'awaited action spins, prevents repeats and recovers after failure',
    (tester) async {
      final request = Completer<void>();
      var attempts = 0;
      await tester.pumpWidget(
        MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: AsyncActionBuilder(
              builder: (context, actions) => Row(
                children: [
                  TextButton(
                    key: const Key('save'),
                    onPressed: actions.isBusy
                        ? null
                        : () async {
                            try {
                              await actions.run(_TestAction.save, () {
                                attempts++;
                                return request.future;
                              });
                            } on Exception {
                              /* The caller presents the failure. */
                            }
                          },
                    child: ActionProgress(
                      isLoading: actions.isRunning(_TestAction.save),
                      child: const Text('Save'),
                    ),
                  ),
                  TextButton(
                    key: const Key('other'),
                    onPressed: actions.isBusy ? null : () {},
                    child: ActionProgress(
                      isLoading: actions.isRunning(_TestAction.other),
                      child: const Text('Other'),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.byKey(const Key('save')));
      await tester.pump();
      expect(find.byType(StitchProgressIndicator), findsOneWidget);
      expect(
        tester.widget<TextButton>(find.byKey(const Key('other'))).onPressed,
        isNull,
      );
      await tester.tap(find.byKey(const Key('save')));
      expect(attempts, 1);
      request.completeError(Exception('failed'));
      await tester.pump();
      expect(find.byType(StitchProgressIndicator), findsNothing);
      expect(
        tester.widget<TextButton>(find.byKey(const Key('save'))).onPressed,
        isNotNull,
      );
      expect(
        tester.widget<TextButton>(find.byKey(const Key('other'))).onPressed,
        isNotNull,
      );
    },
  );
  testWidgets('confirmation waits without spinning behind the dialog', (
    tester,
  ) async {
    final request = Completer<void>();
    await tester.pumpWidget(
      MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: AsyncActionBuilder(
            builder: (context, actions) => TextButton(
              onPressed: actions.isBusy
                  ? null
                  : () => actions.run(_TestAction.delete, () async {
                      final confirmed = await showDialog<bool>(
                        context: context,
                        builder: (dialogContext) => AlertDialog(
                          actions: [
                            TextButton(
                              onPressed: () =>
                                  Navigator.pop(dialogContext, true),
                              child: const Text('Confirm'),
                            ),
                          ],
                        ),
                      );
                      if (confirmed == true) await request.future;
                    }),
              child: ActionProgress(
                isLoading: actions.isBusy,
                child: const Text('Delete'),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Delete'));
    await tester.pumpAndSettle();
    expect(find.byType(StitchProgressIndicator), findsNothing);
    await tester.tap(find.text('Confirm'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));
    expect(find.byType(StitchProgressIndicator), findsOneWidget);
    request.complete();
    await tester.pumpAndSettle();
    expect(find.byType(StitchProgressIndicator), findsNothing);
  });
}
