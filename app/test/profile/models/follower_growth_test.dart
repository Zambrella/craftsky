import 'package:craftsky_app/profile/models/follower_growth.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('FollowerGrowth.fromMap', () {
    test('decodes populated camelCase payload with UTC date identity', () {
      final growth = FollowerGrowth.fromMap(
        growthPayload(populated: true),
      );

      expect(growth.period, FollowerGrowthPeriod.thirtyDays);
      expect(growth.rangeStart, DateTime.utc(2026, 7, 27));
      expect(growth.rangeStart.isUtc, isTrue);
      expect(growth.rangeEnd, DateTime.utc(2026, 8, 25));
      expect(growth.availableFrom, DateTime.utc(2026, 6));
      expect(growth.latestSnapshotDate, DateTime.utc(2026, 8, 25));
      expect(growth.latestCapturedAt, DateTime.utc(2026, 8, 25, 0, 0, 2));
      expect(growth.latestFollowerCount, 42);
      expect(growth.netChange, 5);
      expect(growth.points.first.date, DateTime.utc(2026, 7, 27));
      expect(growth.points.first.count, 37);
      expect(growth.points.last.date.isUtc, isTrue);
      expect(growth.points.last.count, 42);
    });

    test('decodes successful no-history payload without fabricated values', () {
      final growth = FollowerGrowth.fromMap(
        growthPayload(period: '7d', populated: false),
      );

      expect(growth.period, FollowerGrowthPeriod.sevenDays);
      expect(growth.availableFrom, isNull);
      expect(growth.latestSnapshotDate, isNull);
      expect(growth.latestCapturedAt, isNull);
      expect(growth.latestFollowerCount, isNull);
      expect(growth.netChange, isNull);
      expect(growth.points.every((point) => point.count == null), isTrue);
    });

    test('rejects unsupported periods and non-chronological points', () {
      final unsupported = growthPayload(period: '7d', populated: false)
        ..['period'] = '90d';
      final reversed = growthPayload(period: '7d', populated: false);
      (reversed['points'] as List<dynamic>)[1] = {
        'date': '2026-08-19',
        'count': null,
      };

      expect(
        () => FollowerGrowth.fromMap(unsupported),
        throwsFormatException,
      );
      expect(
        () => FollowerGrowth.fromMap(reversed),
        throwsFormatException,
      );
    });

    test('rejects incomplete, non-contiguous, and out-of-range points', () {
      final incomplete = growthPayload(period: '7d', populated: false);
      (incomplete['points'] as List<dynamic>).removeAt(3);
      final outOfRange = growthPayload(period: '7d', populated: false);
      (outOfRange['points'] as List<dynamic>)[0] = {
        'date': '2026-08-18',
        'count': null,
      };

      expect(
        () => FollowerGrowth.fromMap(incomplete),
        throwsFormatException,
      );
      expect(
        () => FollowerGrowth.fromMap(outOfRange),
        throwsFormatException,
      );
    });

    test('rejects contradictory no-history metadata', () {
      final payload = growthPayload(period: '7d', populated: false)
        ..['latestFollowerCount'] = 4;

      expect(() => FollowerGrowth.fromMap(payload), throwsFormatException);
    });

    test('rejects contradictory populated metadata and points', () {
      final incorrectChange = growthPayload(populated: true)..['netChange'] = 6;
      final observationBeforeAvailability = growthPayload(populated: true)
        ..['availableFrom'] = '2026-08-01';
      final mismatchedLatestCount = growthPayload(populated: true)
        ..['latestFollowerCount'] = 43;

      for (final payload in [
        incorrectChange,
        observationBeforeAvailability,
        mismatchedLatestCount,
      ]) {
        expect(() => FollowerGrowth.fromMap(payload), throwsFormatException);
      }
    });

    test('rejects global boundaries that do not identify observations', () {
      final nullAvailabilityPoint = growthPayload(populated: true)
        ..['availableFrom'] = '2026-07-27';
      (nullAvailabilityPoint['points'] as List<dynamic>)[0] = {
        'date': '2026-07-27',
        'count': null,
      };
      (nullAvailabilityPoint['points'] as List<dynamic>)[1] = {
        'date': '2026-07-28',
        'count': 37,
      };

      final observationAfterLatest = growthPayload(populated: true)
        ..['latestSnapshotDate'] = '2026-08-24'
        ..['latestFollowerCount'] = 41;
      (observationAfterLatest['points'] as List<dynamic>)[28] = {
        'date': '2026-08-24',
        'count': 41,
      };

      expect(
        () => FollowerGrowth.fromMap(nullAvailabilityPoint),
        throwsFormatException,
      );
      expect(
        () => FollowerGrowth.fromMap(observationAfterLatest),
        throwsFormatException,
      );
    });
  });
}

Map<String, dynamic> growthPayload({
  required bool populated,
  String period = '30d',
}) {
  final start = switch (period) {
    '7d' => DateTime.utc(2026, 8, 19),
    _ => DateTime.utc(2026, 7, 27),
  };
  final end = DateTime.utc(2026, 8, 25);
  final points = <Map<String, dynamic>>[];
  for (
    var date = start;
    !date.isAfter(end);
    date = date.add(const Duration(days: 1))
  ) {
    points.add({
      'date': date.toIso8601String().substring(0, 10),
      'count': populated
          ? switch (date) {
              final value when value == start => 37,
              final value when value == end => 42,
              _ => null,
            }
          : null,
    });
  }
  return {
    'period': period,
    'rangeStart': start.toIso8601String().substring(0, 10),
    'rangeEnd': '2026-08-25',
    'availableFrom': populated ? '2026-06-01' : null,
    'latestSnapshotDate': populated ? '2026-08-25' : null,
    'latestCapturedAt': populated ? '2026-08-25T00:00:02Z' : null,
    'latestFollowerCount': populated ? 42 : null,
    'netChange': populated ? 5 : null,
    'points': points,
  };
}
