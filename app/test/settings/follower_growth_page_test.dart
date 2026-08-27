import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/follower_growth.dart';
import 'package:craftsky_app/profile/providers/follower_growth_provider.dart';
import 'package:craftsky_app/settings/pages/follower_growth_page.dart';
import 'package:craftsky_app/settings/widgets/follower_growth_chart.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets(
    'active account switch and logout never render late private data',
    (
      tester,
    ) async {
      final alice = AccountKey('did:plc:alice');
      final bob = AccountKey('did:plc:bob');
      final registry = SessionRegistry(
        nextSessionGeneration: 3,
        nextUseOrdinal: 3,
        activationGeneration: 2,
        activeDid: alice.did.value,
        sessions: {
          alice.did.value: StoredSession(
            token: 'alice-token',
            did: alice.did.value,
            handle: 'alice.test',
            sessionGeneration: 1,
            lastUsedOrdinal: 2,
          ),
          bob.did.value: StoredSession(
            token: 'bob-token',
            did: bob.did.value,
            handle: 'bob.test',
            sessionGeneration: 2,
            lastUsedOrdinal: 1,
          ),
        },
      );
      final aliceResult = Completer<FollowerGrowth>();
      final bobResult = Completer<FollowerGrowth>();
      final container = ProviderContainer(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(registry),
          ),
          followerGrowthProvider.overrideWith((ref, key) {
            return key.$1 == alice ? aliceResult.future : bobResult.future;
          }),
        ],
      );
      addTearDown(container.dispose);
      await container.read(sessionRegistryProvider.future);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const FollowerGrowthPage(),
          ),
        ),
      );
      await tester.pump();
      expect(find.byType(StitchProgressIndicator), findsOneWidget);

      final bobLease = registry.leaseFor(bob)!;
      await container.read(sessionRegistryProvider.notifier).activate(bobLease);
      await tester.pump();
      bobResult.complete(
        _growth(FollowerGrowthPeriod.thirtyDays, count: 7, change: null),
      );
      await tester.pumpAndSettle();
      expect(_followerTotal(tester), '7');

      aliceResult.complete(
        _growth(FollowerGrowthPeriod.thirtyDays, count: 41, change: null),
      );
      await tester.pumpAndSettle();
      expect(_followerTotal(tester), '7');

      await container
          .read(sessionRegistryProvider.notifier)
          .removeConfirmed(registry.leaseFor(alice)!);
      await container
          .read(sessionRegistryProvider.notifier)
          .removeConfirmed(bobLease);
      await tester.pump();
      expect(_followerTotal(tester), isNull);
      expect(find.byType(StitchProgressIndicator), findsOneWidget);
    },
  );

  testWidgets('period changes keep the loaded total without a spinner', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    final results = {
      for (final period in FollowerGrowthPeriod.values)
        period: Completer<FollowerGrowth>(),
    };
    await tester.pumpWidget(
      _app(
        overrides: [
          followerGrowthProvider.overrideWith(
            (ref, key) => results[key.$2]!.future,
          ),
        ],
      ),
    );
    await tester.pump();

    expect(find.text('30 days'), findsOneWidget);
    await tester.tap(find.text('7 days'));
    await tester.pump();
    expect(find.text('1 year'), findsOneWidget);
    await tester.tap(find.text('1 year'));
    await tester.pump();

    results[FollowerGrowthPeriod.thirtyDays]!.complete(
      _growth(FollowerGrowthPeriod.thirtyDays, count: 30, change: 3),
    );
    await tester.pump();
    expect(find.byType(StitchProgressIndicator), findsNothing);
    expect(_followerTotal(tester), '30');
    expect(
      tester
          .widget<SegmentedButton<FollowerGrowthPeriod>>(
            find.byType(SegmentedButton<FollowerGrowthPeriod>),
          )
          .selected,
      {FollowerGrowthPeriod.oneYear},
    );

    results[FollowerGrowthPeriod.sevenDays]!.complete(
      _growth(FollowerGrowthPeriod.sevenDays, count: 7, change: 1),
    );
    await tester.pump();
    expect(find.byType(StitchProgressIndicator), findsNothing);
    expect(_followerTotal(tester), '30');

    results[FollowerGrowthPeriod.oneYear]!.complete(
      _growth(FollowerGrowthPeriod.oneYear, count: 100, change: 10),
    );
    await tester.pumpAndSettle();
    expect(_followerTotal(tester), '30');
    expect(find.byType(FollowerGrowthChart), findsOneWidget);
    expect(
      tester.getSemantics(find.byType(FollowerGrowthChart)).label,
      allOf(contains('1 year'), contains('100 followers')),
    );
    semantics.dispose();
  });

  testWidgets('defaults to 30 days and switches period-specific summaries', (
    tester,
  ) async {
    final responses = {
      FollowerGrowthPeriod.sevenDays: _growth(
        FollowerGrowthPeriod.sevenDays,
        count: 10,
        change: -2,
      ),
      FollowerGrowthPeriod.thirtyDays: _growth(
        FollowerGrowthPeriod.thirtyDays,
        count: 42,
        change: 5,
      ),
      FollowerGrowthPeriod.oneYear: _growth(
        FollowerGrowthPeriod.oneYear,
        count: 100,
        change: 0,
      ),
    };

    await tester.pumpWidget(
      _app(
        overrides: [
          followerGrowthProvider.overrideWith(
            (ref, key) async => responses[key.$2]!,
          ),
        ],
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Trend'), findsOneWidget);
    expect(find.textContaining('Craftsky followers'), findsOneWidget);
    expect(find.textContaining('Updated daily'), findsOneWidget);
    expect(find.textContaining('Dates are UTC'), findsOneWidget);
    expect(_followerTotal(tester), '42');
    expect(find.text('followers'), findsOneWidget);
    final totalFinder = find.byKey(const ValueKey('follower-total-count'));
    final totalLabelFinder = find.byKey(
      const ValueKey('follower-total-label'),
    );
    final total = tester.widget<Text>(totalFinder);
    final totalLabel = tester.widget<Text>(totalLabelFinder);
    expect(
      total.style?.color,
      Theme.of(tester.element(totalFinder)).colorScheme.primary,
    );
    expect(
      totalLabel.style?.color,
      Theme.of(tester.element(totalLabelFinder)).colorScheme.onSurfaceVariant,
    );
    expect(
      totalLabel.style?.fontSize,
      Theme.of(tester.element(totalLabelFinder)).textTheme.bodyLarge?.fontSize,
    );
    expect(totalLabel.style?.fontSize, lessThan(total.style!.fontSize!));
    expect(
      tester.getTopLeft(totalFinder).dy,
      lessThan(tester.getTopLeft(totalLabelFinder).dy),
    );
    final totalTop = tester.getTopLeft(totalFinder).dy;
    final trendTop = tester.getTopLeft(find.text('Trend')).dy;
    final selector = find.byType(SegmentedButton<FollowerGrowthPeriod>);
    final selectorTop = tester.getTopLeft(selector).dy;
    final chart = find.byType(FollowerGrowthChart);
    final chartTop = tester.getTopLeft(chart).dy;
    expect(totalTop, lessThan(trendTop));
    expect(trendTop, lessThan(selectorTop));
    expect(selectorTop, lessThan(chartTop));
    expect(
      tester.getSize(selector).width,
      closeTo(tester.getSize(chart).width, 0.01),
    );
    expect(
      tester
          .widget<SegmentedButton<FollowerGrowthPeriod>>(
            find.byType(SegmentedButton<FollowerGrowthPeriod>),
          )
          .selected,
      {FollowerGrowthPeriod.thirtyDays},
    );

    await tester.tap(find.text('7 days'));
    await tester.pumpAndSettle();
    expect(_followerTotal(tester), '42');

    await tester.tap(find.text('1 year'));
    await tester.pumpAndSettle();
    expect(_followerTotal(tester), '42');
  });

  testWidgets('shows loading and retries the current account and period', (
    tester,
  ) async {
    final first = Completer<FollowerGrowth>();
    var calls = 0;
    await tester.pumpWidget(
      _app(
        overrides: [
          followerGrowthProvider(
            _account,
            FollowerGrowthPeriod.thirtyDays,
          ).overrideWith((ref) {
            calls++;
            if (calls == 1) return first.future;
            return Future.value(
              _growth(
                FollowerGrowthPeriod.thirtyDays,
                count: 8,
                change: null,
              ),
            );
          }),
        ],
      ),
    );
    await tester.pump();
    expect(find.byType(StitchProgressIndicator), findsOneWidget);

    first.completeError(Exception('private server detail'));
    await tester.pumpAndSettle();
    expect(find.text('Could not load follower growth.'), findsOneWidget);
    expect(find.text('private server detail'), findsNothing);

    await tester.tap(find.text('Retry'));
    await tester.pumpAndSettle();
    expect(_followerTotal(tester), '8');
    expect(calls, 2);
  });

  testWidgets('distinguishes no history from no observations in the period', (
    tester,
  ) async {
    await _pumpGrowth(tester, _history());
    expect(find.text('No follower history yet'), findsOneWidget);
    expect(find.textContaining('Latest snapshot:'), findsNothing);

    await _pumpGrowth(
      tester,
      _history(
        availableFrom: DateTime.utc(2026, 7),
        latestSnapshotDate: DateTime.utc(2026, 7, 31),
        latestFollowerCount: 3,
      ),
    );
    expect(find.text('No observations in this period'), findsOneWidget);
    expect(_followerTotal(tester), '3');
    expect(find.text('No follower history yet'), findsNothing);
  });

  testWidgets('explains one-point, partial, and missing-date histories', (
    tester,
  ) async {
    await _pumpGrowth(
      tester,
      _history(
        availableFrom: DateTime.utc(2026, 8, 2),
        latestSnapshotDate: DateTime.utc(2026, 8, 2),
        latestFollowerCount: 4,
        points: [
          FollowerGrowthPoint(date: DateTime.utc(2026, 8), count: null),
          FollowerGrowthPoint(date: DateTime.utc(2026, 8, 2), count: 4),
          FollowerGrowthPoint(date: DateTime.utc(2026, 8, 3), count: null),
        ],
      ),
    );
    expect(find.text('History available since 8/2/2026'), findsOneWidget);
    expect(find.text('2 days have no observation'), findsOneWidget);

    await _pumpGrowth(
      tester,
      _history(
        availableFrom: DateTime.utc(2026, 8),
        latestSnapshotDate: DateTime.utc(2026, 8, 3),
        latestFollowerCount: 5,
        netChange: 1,
        points: [
          FollowerGrowthPoint(date: DateTime.utc(2026, 8), count: 4),
          FollowerGrowthPoint(date: DateTime.utc(2026, 8, 2), count: null),
          FollowerGrowthPoint(date: DateTime.utc(2026, 8, 3), count: 5),
        ],
      ),
    );
    expect(find.text('1 day has no observation'), findsOneWidget);
  });

  for (final fixture in <(Size, bool)>[
    (const Size(320, 640), false),
    (const Size(320, 640), true),
    (const Size(1200, 800), false),
    (const Size(1200, 800), true),
  ]) {
    testWidgets('renders chart without overflow at ${fixture.$1}, '
        '${fixture.$2 ? 'dark' : 'light'}, and 200% text', (
      tester,
    ) async {
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      tester.view
        ..devicePixelRatio = 1
        ..physicalSize = fixture.$1;

      await tester.pumpWidget(
        _app(
          textScaler: const TextScaler.linear(2),
          dark: fixture.$2,
          overrides: [
            followerGrowthProvider.overrideWith(
              (ref, key) async => _growth(
                FollowerGrowthPeriod.thirtyDays,
                count: 123456789,
                change: 5,
              ),
            ),
          ],
        ),
      );
      await tester.pumpAndSettle();

      expect(find.byType(FollowerGrowthChart), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }
}

const _accountDid = 'did:plc:test';
final _account = AccountKey(_accountDid);

Widget _app({
  required List<dynamic> overrides,
  Key? key,
  TextScaler? textScaler,
  bool dark = false,
}) => ProviderScope(
  key: key,
  retry: (_, _) => null,
  overrides: List.from([
    secureSessionRegistryStorageProvider.overrideWithValue(
      _RegistryStorage(_registry),
    ),
    ...overrides,
  ]),
  child: MaterialApp(
    theme: dark ? AppTheme.darkThemeData : AppTheme.lightThemeData,
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    builder: textScaler == null
        ? null
        : (context, child) => MediaQuery(
            data: MediaQuery.of(context).copyWith(textScaler: textScaler),
            child: child!,
          ),
    home: const FollowerGrowthPage(),
  ),
);

final _registry = SessionRegistry(
  nextSessionGeneration: 2,
  nextUseOrdinal: 2,
  activationGeneration: 1,
  activeDid: _accountDid,
  sessions: {
    _accountDid: StoredSession(
      token: 'session-token',
      did: _accountDid,
      handle: 'test.craftsky.social',
      sessionGeneration: 1,
      lastUsedOrdinal: 1,
    ),
  },
);

final class _RegistryStorage implements SessionRegistryStorage {
  const _RegistryStorage(this.value);

  final SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async {}
}

Future<void> _pumpGrowth(
  WidgetTester tester,
  FollowerGrowth growth,
) async {
  await tester.pumpWidget(
    _app(
      key: ValueKey(growth),
      overrides: [
        followerGrowthProvider.overrideWith((ref, key) async => growth),
      ],
    ),
  );
  await tester.pumpAndSettle();
}

String? _followerTotal(WidgetTester tester) {
  final finder = find.byKey(const ValueKey('follower-total-count'));
  if (finder.evaluate().isEmpty) return null;
  return tester.widget<Text>(finder).data;
}

FollowerGrowth _history({
  DateTime? availableFrom,
  DateTime? latestSnapshotDate,
  int? latestFollowerCount,
  int? netChange,
  List<FollowerGrowthPoint>? points,
}) => FollowerGrowth(
  period: FollowerGrowthPeriod.thirtyDays,
  rangeStart: DateTime.utc(2026, 8),
  rangeEnd: DateTime.utc(2026, 8, 3),
  availableFrom: availableFrom,
  latestSnapshotDate: latestSnapshotDate,
  latestCapturedAt: latestSnapshotDate,
  latestFollowerCount: latestFollowerCount,
  netChange: netChange,
  points:
      points ??
      [
        FollowerGrowthPoint(date: DateTime.utc(2026, 8), count: null),
        FollowerGrowthPoint(date: DateTime.utc(2026, 8, 2), count: null),
        FollowerGrowthPoint(date: DateTime.utc(2026, 8, 3), count: null),
      ],
);

FollowerGrowth _growth(
  FollowerGrowthPeriod period, {
  required int count,
  required int? change,
}) => FollowerGrowth(
  period: period,
  rangeStart: DateTime.utc(2026, 8),
  rangeEnd: DateTime.utc(2026, 8, 2),
  availableFrom: DateTime.utc(2026, 8),
  latestSnapshotDate: DateTime.utc(2026, 8, 2),
  latestCapturedAt: DateTime.utc(2026, 8, 2, 0, 0, 2),
  latestFollowerCount: count,
  netChange: change,
  points: [
    FollowerGrowthPoint(
      date: DateTime.utc(2026, 8),
      count: change == null ? null : count - change,
    ),
    FollowerGrowthPoint(date: DateTime.utc(2026, 8, 2), count: count),
  ],
);
