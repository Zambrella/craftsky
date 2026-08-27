import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/l10n/generated/app_localizations_en.dart';
import 'package:craftsky_app/profile/models/follower_growth.dart';
import 'package:craftsky_app/settings/widgets/follower_growth_chart.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('chart preserves null gaps and exposes one textual summary', (
    tester,
  ) async {
    final growth = FollowerGrowth(
      period: FollowerGrowthPeriod.thirtyDays,
      rangeStart: DateTime.utc(2026, 8),
      rangeEnd: DateTime.utc(2026, 8, 3),
      availableFrom: DateTime.utc(2026, 8),
      latestSnapshotDate: DateTime.utc(2026, 8, 3),
      latestCapturedAt: DateTime.utc(2026, 8, 3, 0, 0, 2),
      latestFollowerCount: 42,
      netChange: 5,
      points: [
        FollowerGrowthPoint(date: DateTime.utc(2026, 8), count: 37),
        FollowerGrowthPoint(date: DateTime.utc(2026, 8, 2), count: null),
        FollowerGrowthPoint(date: DateTime.utc(2026, 8, 3), count: 42),
      ],
    );
    final semantics = tester.ensureSemantics();
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: FollowerGrowthChart(
            growth: growth,
            period: FollowerGrowthPeriod.thirtyDays,
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final chart = tester.widget<LineChart>(find.byType(LineChart));
    expect(chart.data.lineBarsData.single.spots, contains(FlSpot.nullSpot));
    expect(
      tester.getSemantics(find.byType(FollowerGrowthChart)).label,
      contains('30 days'),
    );
    expect(
      tester.getSemantics(find.byType(FollowerGrowthChart)).label,
      allOf(contains('42 followers'), contains('Up 5'), contains('1 day')),
    );
    semantics.dispose();
  });

  for (final fixture in <(String, List<int>, int?, String)>[
    ('negative', [42, 37], -5, 'Down 5'),
    ('flat', [42, 42], 0, 'No change'),
    ('one point', [42], null, 'Not enough history'),
  ]) {
    testWidgets('chart semantics describe ${fixture.$1} history', (
      tester,
    ) async {
      final points = [
        for (final (index, count) in fixture.$2.indexed)
          FollowerGrowthPoint(
            date: DateTime.utc(2026, 8, index + 1),
            count: count,
          ),
      ];
      final growth = FollowerGrowth(
        period: FollowerGrowthPeriod.sevenDays,
        rangeStart: points.first.date,
        rangeEnd: points.last.date,
        availableFrom: points.first.date,
        latestSnapshotDate: points.last.date,
        latestCapturedAt: points.last.date,
        latestFollowerCount: points.last.count,
        netChange: fixture.$3,
        points: points,
      );
      final semantics = tester.ensureSemantics();
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.darkThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: FollowerGrowthChart(
              growth: growth,
              period: FollowerGrowthPeriod.sevenDays,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      final label = tester.getSemantics(find.byType(FollowerGrowthChart)).label;
      expect(
        label,
        allOf(
          contains('7 days'),
          contains('${fixture.$2.last} followers'),
          contains(fixture.$4),
          contains('Updated daily'),
        ),
      );
      semantics.dispose();
    });
  }

  testWidgets(
    'localized large Y-axis labels stay inside compact chart bounds',
    (tester) async {
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      tester.view
        ..devicePixelRatio = 1
        ..physicalSize = const Size(320, 640);
      final growth = FollowerGrowth(
        period: FollowerGrowthPeriod.thirtyDays,
        rangeStart: DateTime.utc(2026, 8),
        rangeEnd: DateTime.utc(2026, 8, 2),
        availableFrom: DateTime.utc(2026, 8),
        latestSnapshotDate: DateTime.utc(2026, 8, 2),
        latestCapturedAt: DateTime.utc(2026, 8, 2),
        latestFollowerCount: 123456789,
        netChange: 9,
        points: [
          FollowerGrowthPoint(
            date: DateTime.utc(2026, 8),
            count: 123456780,
          ),
          FollowerGrowthPoint(
            date: DateTime.utc(2026, 8, 2),
            count: 123456789,
          ),
        ],
      );

      await tester.pumpWidget(
        MaterialApp(
          locale: const Locale('de'),
          supportedLocales: const [Locale('de')],
          localizationsDelegates: const [
            _TestAppLocalizationsDelegate(),
            ...AppLocalizations.localizationsDelegates,
          ],
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(
              context,
            ).copyWith(textScaler: const TextScaler.linear(2)),
            child: child!,
          ),
          home: Scaffold(
            body: FollowerGrowthChart(
              growth: growth,
              period: FollowerGrowthPeriod.thirtyDays,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      final labels = find.byKey(
        const ValueKey('follower-growth-y-axis-label'),
      );
      expect(labels, findsWidgets);
      final chartBounds = tester.getRect(find.byType(FollowerGrowthChart));
      for (final element in labels.evaluate()) {
        final box = element.renderObject! as RenderBox;
        final bounds = box.localToGlobal(Offset.zero) & box.size;
        expect(
          chartBounds.contains(bounds.topLeft),
          isTrue,
          reason: '$bounds starts outside $chartBounds',
        );
        expect(
          chartBounds.contains(bounds.bottomRight),
          isTrue,
          reason: '$bounds ends outside $chartBounds',
        );
      }
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('one-year chart groups daily history by calendar month', (
    tester,
  ) async {
    final growth = FollowerGrowth(
      period: FollowerGrowthPeriod.oneYear,
      rangeStart: DateTime.utc(2025, 12, 30),
      rangeEnd: DateTime.utc(2026, 2, 2),
      availableFrom: DateTime.utc(2025, 12, 30),
      latestSnapshotDate: DateTime.utc(2026, 2, 2),
      latestCapturedAt: DateTime.utc(2026, 2, 2),
      latestFollowerCount: 14,
      netChange: 4,
      points: [
        FollowerGrowthPoint(date: DateTime.utc(2025, 12, 30), count: 10),
        FollowerGrowthPoint(date: DateTime.utc(2025, 12, 31), count: 11),
        FollowerGrowthPoint(date: DateTime.utc(2026), count: null),
        FollowerGrowthPoint(date: DateTime.utc(2026, 1, 31), count: null),
        FollowerGrowthPoint(date: DateTime.utc(2026, 2), count: 13),
        FollowerGrowthPoint(date: DateTime.utc(2026, 2, 2), count: 14),
      ],
    );
    await _pumpChart(tester, growth);

    final spots = tester
        .widget<LineChart>(find.byType(LineChart))
        .data
        .lineBarsData
        .single
        .spots;
    expect(spots, [const FlSpot(0, 11), FlSpot.nullSpot, const FlSpot(2, 14)]);
  });

  testWidgets('small follower ranges use distinct integer Y-axis ticks', (
    tester,
  ) async {
    final growth = FollowerGrowth(
      period: FollowerGrowthPeriod.sevenDays,
      rangeStart: DateTime.utc(2026, 8),
      rangeEnd: DateTime.utc(2026, 8, 4),
      availableFrom: DateTime.utc(2026, 8),
      latestSnapshotDate: DateTime.utc(2026, 8, 4),
      latestCapturedAt: DateTime.utc(2026, 8, 4),
      latestFollowerCount: 155,
      netChange: 2,
      points: [
        FollowerGrowthPoint(date: DateTime.utc(2026, 8), count: 153),
        FollowerGrowthPoint(date: DateTime.utc(2026, 8, 2), count: 154),
        FollowerGrowthPoint(date: DateTime.utc(2026, 8, 3), count: 154),
        FollowerGrowthPoint(date: DateTime.utc(2026, 8, 4), count: 155),
      ],
    );
    await _pumpChart(tester, growth);

    final data = tester.widget<LineChart>(find.byType(LineChart)).data;
    expect(data.titlesData.leftTitles.sideTitles.interval, 1);
    expect(data.gridData.horizontalInterval, 1);
    expect(data.minY, 152);
    expect(data.maxY, 156);
  });
}

Future<void> _pumpChart(WidgetTester tester, FollowerGrowth growth) async {
  await tester.pumpWidget(
    MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: Scaffold(
        body: FollowerGrowthChart(growth: growth, period: growth.period),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

class _TestAppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _TestAppLocalizationsDelegate();

  @override
  bool isSupported(Locale locale) => true;

  @override
  Future<AppLocalizations> load(Locale locale) async => AppLocalizationsEn();

  @override
  bool shouldReload(_TestAppLocalizationsDelegate old) => false;
}
