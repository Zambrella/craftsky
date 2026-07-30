import 'dart:ui' show Tristate;

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/post_language_selection.dart';
import 'package:craftsky_app/languages/widgets/post_language_selector.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets(
    'AT-014 exposes selected state, labels, limit, and enlarged layout',
    (tester) async {
      final semantics = tester.ensureSemantics();
      await tester.binding.setSurfaceSize(const Size(320, 640));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      await tester.pumpWidget(
        MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: MediaQuery(
            data: const MediaQueryData(
              size: Size(320, 640),
              textScaler: TextScaler.linear(2),
            ),
            child: Scaffold(
              body: SingleChildScrollView(
                child: PostLanguageSelector(
                  selection: PostLanguageSelection.fromPrimary(
                    'en',
                  ).add('fr').add('cy'),
                  onChanged: (_) {},
                ),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.bySemanticsLabel('Post languages'), findsOneWidget);
      expect(find.text('Up to three languages'), findsOneWidget);
      expect(tester.takeException(), isNull);

      for (final label in ['English', 'French', 'Welsh']) {
        final data = tester
            .getSemantics(find.widgetWithText(InputChip, label))
            .getSemanticsData();
        expect(data.flagsCollection.isSelected, Tristate.isTrue);
      }
      final addLanguage = tester.widget<ActionChip>(
        find.widgetWithText(ActionChip, 'Add language'),
      );
      expect(addLanguage.onPressed, isNull);

      semantics.dispose();
    },
  );
}
