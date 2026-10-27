import 'dart:io';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_edit_indicator.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets('UT-005 regional locale controls the date and 24-hour time', (
    tester,
  ) async {
    for (final (region, date) in [
      ('US', 'October 25, 2026'),
      ('GB', '25 October 2026'),
    ]) {
      await tester.pumpWidget(
        MaterialApp(
          locale: Locale('en', region),
          supportedLocales: [Locale('en', region)],
          localizationsDelegates: appLocalizationsDelegates,
          home: Builder(
            builder: (context) => MediaQuery(
              data: MediaQuery.of(context)
                  .copyWith(alwaysUse24HourFormat: true),
              child: Scaffold(
                body: PostEditIndicator(
                  editedAt: DateTime(2026, 10, 25, 12, 30),
                  source: PostEditTimeSource.confirmed,
                ),
              ),
            ),
          ),
        ),
      );
      final label = tester.widget<Tooltip>(find.byType(Tooltip)).message!;
      expect(label, contains(date));
      expect(label, contains('12:30'));
    }
  });

  testWidgets('UT-005 daylight-saving repeated wall times remain distinct', (
    tester,
  ) async {
    final labels = <String>[];
    for (final timestamp in [
      '2026-10-25T01:30:00+01:00',
      '2026-10-25T01:30:00+00:00',
    ]) {
      await tester.pumpWidget(
        MaterialApp(
          locale: const Locale('en', 'GB'),
          supportedLocales: const [Locale('en', 'GB')],
          localizationsDelegates: appLocalizationsDelegates,
          home: Scaffold(
            body: PostEditIndicator(
              editedAt: DateTime.parse(timestamp),
              source: PostEditTimeSource.confirmed,
            ),
          ),
        ),
      );
      labels.add(tester.widget<Tooltip>(find.byType(Tooltip)).message!);
    }
    expect(labels.toSet(), hasLength(2));
    // The directed TZ=Europe/London run exercises the actual repeated hour.
    if (Platform.environment['TZ'] == 'Europe/London') {
      expect(labels[0], contains('01:30 BST'));
      expect(labels[1], contains('01:30 GMT'));
    }
  });

  testWidgets('AT-003 edit time opens with touch and keyboard focus', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: PostEditIndicator(
            editedAt: DateTime.utc(2026, 10, 26, 12),
            source: PostEditTimeSource.confirmed,
          ),
        ),
      ),
    );
    final label = tester.widget<Tooltip>(find.byType(Tooltip)).message!;
    await tester.tap(find.byIcon(CraftskyIcons.edit));
    await tester.pump(const Duration(milliseconds: 250));
    expect(find.text(label), findsOneWidget);
    await tester.pump(const Duration(seconds: 5));
    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pump(const Duration(milliseconds: 250));
    expect(find.text(label), findsOneWidget);
  });

  testWidgets(
    'AT-003 edit disclosure qualifies its time in tooltip and semantics',
    (tester) async {
      final semantics = tester.ensureSemantics();
      for (final entry in {
        PostEditTimeSource.confirmed: 'Edited',
        PostEditTimeSource.reported: 'Edit reported for',
        PostEditTimeSource.observed: 'Edit first observed',
      }.entries) {
        await tester.pumpWidget(
          MaterialApp(
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: PostEditIndicator(
                editedAt: DateTime.utc(2026, 10, 26, 12),
                source: entry.key,
              ),
            ),
          ),
        );
        expect(find.byIcon(CraftskyIcons.edit), findsOneWidget);
        final tooltip = tester.widget<Tooltip>(find.byType(Tooltip));
        expect(tooltip.message, startsWith(entry.value));
        expect(tooltip.message, contains('2026'));
        expect(find.bySemanticsLabel(tooltip.message!), findsOneWidget);
      }
      await tester.pumpWidget(const MaterialApp(home: PostEditIndicator()));
      expect(find.byIcon(CraftskyIcons.edit), findsNothing);
      semantics.dispose();
    },
  );
}
