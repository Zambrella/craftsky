import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/time/relative_time_text.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  final timestamp = DateTime(2026, 7, 12, 14, 30);

  Future<void> pumpTimestamp(
    WidgetTester tester, {
    required DateTime now,
    DateTime? when,
    DateTime? editedAt,
  }) => tester.pumpWidget(
    MaterialApp(
      locale: const Locale('en', 'GB'),
      supportedLocales: const [Locale('en', 'GB')],
      localizationsDelegates: appLocalizationsDelegates,
      home: Scaffold(
        body: RelativeTimeText(
          timestamp: when ?? timestamp,
          now: now,
          editedAt: editedAt,
        ),
      ),
    ),
  );

  for (final (age, label) in [
    (const Duration(seconds: -30), 'now'),
    (const Duration(seconds: 30), 'now'),
    (const Duration(minutes: 3), '3m'),
    (const Duration(hours: 4), '4h'),
    (const Duration(days: 2), '2d'),
    (const Duration(days: 7) - const Duration(microseconds: 1), '6d'),
    (const Duration(days: 7), '7d'),
    (const Duration(days: 7, microseconds: 1), '12 Jul 26'),
    (const Duration(days: 365), '12 Jul 26'),
  ]) {
    testWidgets('shows $label for timestamp age $age', (tester) async {
      await pumpTimestamp(tester, now: timestamp.add(age));
      expect(find.text(label), findsOneWidget);
    });
  }

  testWidgets('compact date uses the local day and an unpadded day', (
    tester,
  ) async {
    final when = DateTime(2025, 1, 2, 0, 30).toUtc();
    await pumpTimestamp(tester, when: when, now: DateTime(2026));
    expect(find.text('2 Jan 25'), findsOneWidget);
  });

  testWidgets('compact date retains the full timestamp tooltip', (
    tester,
  ) async {
    await pumpTimestamp(tester, now: timestamp.add(const Duration(days: 8)));
    final message = tester.widget<Tooltip>(find.byType(Tooltip)).message!;
    expect(message, contains('12 July 2026'));
    expect(message, contains('14:30'));
    await tester.longPress(find.text('12 Jul 26'));
    await tester.pump();
    expect(find.text(message), findsOneWidget);
  });

  testWidgets(
    'old edited posts show the publication date and edit disclosure',
    (tester) async {
      final now = timestamp.add(const Duration(days: 8));
      await pumpTimestamp(tester, now: now, editedAt: now);
      expect(
        find.textContaining('12 Jul 26', findRichText: true),
        findsOneWidget,
      );
      expect(find.byIcon(CraftskyIcons.edit), findsOneWidget);
      final message = tester.widget<Tooltip>(find.byType(Tooltip)).message!;
      expect(message, contains('12 July 2026'));
      expect(message, contains('Edited'));
      expect(message, contains('20 July 2026'));
    },
  );
}
