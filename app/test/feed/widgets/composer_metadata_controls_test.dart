import 'dart:ui' show Tristate;

import 'package:craftsky_app/feed/widgets/composer_metadata_controls.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/post_language_selection.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('collapses inactive composer metadata into three icon controls', (
    tester,
  ) async {
    await tester.pumpWidget(
      _app(
        ComposerMetadataControls(
          languages: PostLanguageSelection.fromPrimary('en').add('fr'),
          onLanguagesChanged: (_) {},
          sponsored: false,
          onSponsoredChanged: (_) {},
          scheduledAtLocal: null,
          onSchedulePressed: (_) {},
        ),
      ),
    );

    expect(find.byKey(const Key('composer-language-control')), findsOneWidget);
    expect(find.byKey(const Key('composer-sponsored-control')), findsOneWidget);
    expect(find.byKey(const Key('composer-schedule-control')), findsOneWidget);
    expect(find.text('2'), findsOneWidget);
    expect(find.text('Sponsored'), findsNothing);
    expect(find.text('Now'), findsNothing);
  });

  testWidgets(
    'language context shows selections and removes a checked language',
    (
      tester,
    ) async {
      var languages = PostLanguageSelection.fromPrimary('en').add('fr');
      await tester.pumpWidget(
        _app(
          StatefulBuilder(
            builder: (context, setState) => ComposerMetadataControls(
              languages: languages,
              onLanguagesChanged: (value) => setState(() => languages = value),
              sponsored: false,
              onSponsoredChanged: (_) {},
              scheduledAtLocal: null,
              onSchedulePressed: (_) {},
            ),
          ),
        ),
      );

      await tester.tap(find.byKey(const Key('composer-language-control')));
      await tester.pumpAndSettle();

      expect(find.text('English'), findsOneWidget);
      expect(find.text('French'), findsOneWidget);
      expect(find.text('Add language'), findsOneWidget);

      await tester.tap(find.text('French'));
      await tester.pumpAndSettle();

      expect(languages.values, ['en']);
      expect(find.text('1'), findsOneWidget);
    },
  );

  testWidgets('language context adds another language from search', (
    tester,
  ) async {
    var languages = PostLanguageSelection.fromPrimary('en');
    await tester.pumpWidget(
      _app(
        StatefulBuilder(
          builder: (context, setState) => ComposerMetadataControls(
            languages: languages,
            onLanguagesChanged: (value) => setState(() => languages = value),
            sponsored: false,
            onSponsoredChanged: (_) {},
            scheduledAtLocal: null,
            onSchedulePressed: (_) {},
          ),
        ),
      ),
    );

    await tester.tap(find.byKey(const Key('composer-language-control')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Add language'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'Welsh');
    await tester.pump();
    await tester.tap(find.text('Welsh').last);
    await tester.pumpAndSettle();

    expect(languages.values, ['en', 'cy']);
    expect(find.text('2'), findsOneWidget);
  });

  testWidgets('sponsored context checks the option and activates its color', (
    tester,
  ) async {
    var sponsored = false;
    await tester.pumpWidget(
      _app(
        StatefulBuilder(
          builder: (context, setState) => ComposerMetadataControls(
            languages: PostLanguageSelection.fromPrimary('en'),
            onLanguagesChanged: (_) {},
            sponsored: sponsored,
            onSponsoredChanged: (value) => setState(() => sponsored = value),
            scheduledAtLocal: null,
            onSchedulePressed: (_) {},
          ),
        ),
      ),
    );

    await tester.tap(find.byKey(const Key('composer-sponsored-control')));
    await tester.pumpAndSettle();
    expect(find.text('Sponsored'), findsOneWidget);
    expect(tester.widget<Switch>(find.byType(Switch)).value, isFalse);
    expect(
      find.text(
        'This post includes sponsorship or other commercial consideration',
      ),
      findsOneWidget,
    );

    await tester.tap(find.text('Sponsored'));
    await tester.pumpAndSettle();

    expect(sponsored, isTrue);
    final button = tester.widget<IconButton>(
      find.byKey(const Key('composer-sponsored-control')),
    );
    expect(button.color, AppTheme.lightThemeData.colorScheme.primary);
  });

  testWidgets('scheduled control shows a compact primary date and time', (
    tester,
  ) async {
    var presses = 0;
    await tester.pumpWidget(
      _app(
        ComposerMetadataControls(
          languages: PostLanguageSelection.fromPrimary('en'),
          onLanguagesChanged: (_) {},
          sponsored: false,
          onSponsoredChanged: (_) {},
          scheduledAtLocal: DateTime(2026, 9, 15, 14, 30),
          onSchedulePressed: (_) => presses++,
        ),
      ),
    );

    expect(find.text('Sep 15, 2:30 PM'), findsOneWidget);
    final schedule = find.byKey(const Key('composer-schedule-control'));
    expect(tester.widget<TextButton>(schedule), isNotNull);
    expect(
      tester
          .widget<Icon>(
            find.descendant(of: schedule, matching: find.byType(Icon)),
          )
          .color,
      AppTheme.lightThemeData.colorScheme.primary,
    );

    await tester.tap(schedule);
    expect(presses, 1);
  });

  testWidgets('announces complete metadata values and selected states', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    await tester.pumpWidget(
      _app(
        ComposerMetadataControls(
          languages: PostLanguageSelection.fromPrimary('en').add('fr'),
          onLanguagesChanged: (_) {},
          sponsored: true,
          onSponsoredChanged: (_) {},
          scheduledAtLocal: null,
          onSchedulePressed: (_) {},
        ),
      ),
    );

    final languages = tester.getSemantics(
      find.bySemanticsLabel('Post languages'),
    );
    expect(languages.value, 'English, French');
    expect(
      tester
          .getSemantics(find.bySemanticsLabel('Sponsored'))
          .flagsCollection
          .isSelected,
      Tristate.isTrue,
    );
    expect(
      tester.getSemantics(find.bySemanticsLabel('When')).value,
      'Now',
    );
    semantics.dispose();
  });

  testWidgets('hides unavailable actions and wraps without overflow', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(320, 640);
    addTearDown(tester.view.reset);
    await tester.pumpWidget(
      _app(
        MediaQuery(
          data: const MediaQueryData(textScaler: TextScaler.linear(2)),
          child: ComposerMetadataControls(
            languages: PostLanguageSelection.fromPrimary('en'),
            onLanguagesChanged: (_) {},
            sponsored: false,
            onSponsoredChanged: null,
            showSponsored: false,
            scheduledAtLocal: null,
            onSchedulePressed: null,
            showSchedule: false,
          ),
        ),
      ),
    );

    expect(find.byKey(const Key('composer-language-control')), findsOneWidget);
    expect(find.byKey(const Key('composer-sponsored-control')), findsNothing);
    expect(find.byKey(const Key('composer-schedule-control')), findsNothing);
    expect(
      tester.getSize(find.byKey(const Key('composer-language-control'))).height,
      48,
    );
    expect(tester.takeException(), isNull);
  });
}

Widget _app(Widget child) => MaterialApp(
  theme: AppTheme.lightThemeData,
  localizationsDelegates: AppLocalizations.localizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  home: Scaffold(body: child),
);
