import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/post_language_selection.dart';
import 'package:craftsky_app/languages/widgets/post_language_selector.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_text_inputs.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

Widget _app(Widget child) => MaterialApp(
  theme: AppTheme.lightThemeData,
  localizationsDelegates: const [
    AppLocalizations.delegate,
    GlobalMaterialLocalizations.delegate,
    GlobalWidgetsLocalizations.delegate,
    GlobalCupertinoLocalizations.delegate,
  ],
  supportedLocales: AppLocalizations.supportedLocales,
  home: Scaffold(body: child),
);

void main() {
  testWidgets('searches the full catalogue and adds a distinct language', (
    tester,
  ) async {
    var selection = PostLanguageSelection.fromPrimary('en');
    await tester.pumpWidget(
      _app(
        StatefulBuilder(
          builder: (context, setState) => PostLanguageSelector(
            selection: selection,
            onChanged: (value) => setState(() => selection = value),
          ),
        ),
      ),
    );

    await tester.tap(find.text('Add language'));
    await tester.pumpAndSettle();
    expect(find.byType(CraftskyDialog), findsOneWidget);
    expect(find.byType(AlertDialog), findsNothing);
    expect(find.byType(CraftskyTextInput), findsOneWidget);
    expect(find.text('Search languages'), findsOneWidget);
    final resultList = tester.widget<ListView>(find.byType(ListView));
    expect(resultList.padding, EdgeInsets.zero);

    await tester.enterText(find.byType(TextField), 'Welsh');
    await tester.pump();
    await tester.tap(find.text('Welsh').last);
    await tester.pumpAndSettle();

    expect(selection.values, ['en', 'cy']);
    expect(find.text('Welsh'), findsOneWidget);
  });

  testWidgets('exposes the three-language limit and prevents a fourth', (
    tester,
  ) async {
    final selection = PostLanguageSelection.fromPrimary(
      'en',
    ).add('fr').add('cy');
    await tester.pumpWidget(
      _app(
        PostLanguageSelector(
          selection: selection,
          onChanged: (_) => fail('A fourth language must not be added'),
        ),
      ),
    );

    final add = tester.widget<ActionChip>(find.byType(ActionChip));
    expect(add.onPressed, isNull);
    expect(find.text('Up to three languages'), findsOneWidget);
  });
}
