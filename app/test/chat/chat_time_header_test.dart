import 'package:craftsky_app/chat/widgets/chat_time_header.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  test('UT-010 local day and elapsed time section boundaries', () {
    final first = DateTime(2026, 10, 31, 9);
    expect(startsChatTimeSection(first, null), true);
    expect(
      startsChatTimeSection(
        first.add(const Duration(minutes: 4, seconds: 59)),
        first,
      ),
      false,
    );
    expect(
      startsChatTimeSection(first.add(const Duration(minutes: 5)), first),
      true,
    );
    expect(
      startsChatTimeSection(
        DateTime(2026, 11, 1, 0, 1).toUtc(),
        DateTime(2026, 10, 31, 23, 59).toUtc(),
      ),
      true,
    );
    expect(
      startsChatTimeSection(
        DateTime(2026, 11, 1, 0, 3).toUtc(),
        DateTime(2026, 11, 1, 0, 1).toUtc(),
      ),
      false,
    );
  });
  for (final scenario in [
    (
      date: DateTime(2026, 10, 31, 9, 17),
      label: 'Today at 09:17',
      hours24: true,
    ),
    (
      date: DateTime(2026, 10, 30, 18, 39),
      label: 'Yesterday at 18:39',
      hours24: true,
    ),
    (
      date: DateTime(2026, 10, 21, 14, 15),
      label: 'Wed 21 October at 14:15',
      hours24: true,
    ),
    (
      date: DateTime(2025, 10, 21, 14, 15),
      label: 'Tue 21 October 2025 at 14:15',
      hours24: true,
    ),
    (
      date: DateTime(2026, 10, 31, 9, 17),
      label: 'Today at 9:17 AM',
      hours24: false,
    ),
  ]) {
    testWidgets('UT-010 localized time section ${scenario.label}', (
      tester,
    ) async {
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: MediaQuery(
            data: MediaQueryData(
              alwaysUse24HourFormat: scenario.hours24,
              textScaler: const TextScaler.linear(2),
            ),
            child: Scaffold(
              body: ChatTimeHeader(
                timestamp: scenario.date.toUtc(),
                now: DateTime(2026, 10, 31, 12),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text(scenario.label), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }
}
