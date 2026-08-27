import 'package:intl/intl.dart';

enum FollowerGrowthPeriod {
  sevenDays('7d'),
  thirtyDays('30d'),
  oneYear('1y');

  const FollowerGrowthPeriod(this.wireValue);

  final String wireValue;

  static FollowerGrowthPeriod fromWireValue(Object? value) => switch (value) {
    '7d' => sevenDays,
    '30d' => thirtyDays,
    '1y' => oneYear,
    _ => throw FormatException('Unsupported follower growth period: $value'),
  };
}

class FollowerGrowthPoint {
  const FollowerGrowthPoint({required this.date, required this.count});

  factory FollowerGrowthPoint.fromMap(Map<String, dynamic> map) {
    return FollowerGrowthPoint(
      date: _parseDateOnly(map['date'], 'date'),
      count: _nullableNonNegativeInt(map['count'], 'count'),
    );
  }

  final DateTime date;
  final int? count;
}

class FollowerGrowth {
  const FollowerGrowth({
    required this.period,
    required this.rangeStart,
    required this.rangeEnd,
    required this.availableFrom,
    required this.latestSnapshotDate,
    required this.latestCapturedAt,
    required this.latestFollowerCount,
    required this.netChange,
    required this.points,
  });

  factory FollowerGrowth.fromMap(Map<String, dynamic> map) {
    const requiredKeys = {
      'period',
      'rangeStart',
      'rangeEnd',
      'availableFrom',
      'latestSnapshotDate',
      'latestCapturedAt',
      'latestFollowerCount',
      'netChange',
      'points',
    };
    for (final key in requiredKeys) {
      if (!map.containsKey(key)) {
        throw FormatException('Missing follower growth field: $key');
      }
    }

    final period = FollowerGrowthPeriod.fromWireValue(map['period']);
    final rangeStart = _parseDateOnly(map['rangeStart'], 'rangeStart');
    final rangeEnd = _parseDateOnly(map['rangeEnd'], 'rangeEnd');
    if (rangeStart.isAfter(rangeEnd)) {
      throw const FormatException('Follower growth range is reversed');
    }

    final rawPoints = map['points'];
    final expectedPointCount = rangeEnd.difference(rangeStart).inDays + 1;
    if (rawPoints is! List ||
        expectedPointCount > 367 ||
        rawPoints.length != expectedPointCount) {
      throw const FormatException('Invalid follower growth points');
    }
    final points = rawPoints
        .map((value) {
          if (value is! Map<String, dynamic>) {
            throw const FormatException('Invalid follower growth point');
          }
          return FollowerGrowthPoint.fromMap(value);
        })
        .toList(growable: false);
    for (var i = 0; i < points.length; i++) {
      if (points[i].date != rangeStart.add(Duration(days: i))) {
        throw const FormatException('Follower growth points must be complete');
      }
    }
    final validPeriodRange = switch (period) {
      FollowerGrowthPeriod.sevenDays => expectedPointCount == 7,
      FollowerGrowthPeriod.thirtyDays => expectedPointCount == 30,
      FollowerGrowthPeriod.oneYear =>
        rangeStart == _previousYearAnniversary(rangeEnd),
    };
    if (!validPeriodRange) {
      throw const FormatException(
        'Follower growth range does not match period',
      );
    }

    final availableFrom = _nullableDateOnly(
      map['availableFrom'],
      'availableFrom',
    );
    final latestSnapshotDate = _nullableDateOnly(
      map['latestSnapshotDate'],
      'latestSnapshotDate',
    );
    final latestCapturedAt = _nullableTimestamp(
      map['latestCapturedAt'],
      'latestCapturedAt',
    );
    final latestFollowerCount = _nullableNonNegativeInt(
      map['latestFollowerCount'],
      'latestFollowerCount',
    );
    final netChange = _nullableInt(map['netChange'], 'netChange');
    final latestFieldsPresent = [
      latestSnapshotDate,
      latestCapturedAt,
      latestFollowerCount,
    ].where((value) => value != null).length;
    if ((availableFrom == null && latestFieldsPresent != 0) ||
        (availableFrom != null && latestFieldsPresent != 3) ||
        (availableFrom != null &&
            latestSnapshotDate!.isBefore(availableFrom))) {
      throw const FormatException('Inconsistent follower growth metadata');
    }
    final observedPoints = points
        .where((point) => point.count != null)
        .toList(growable: false);
    final observedPointCount = observedPoints.length;
    if (availableFrom == null &&
        (observedPointCount != 0 || netChange != null)) {
      throw const FormatException('Inconsistent no-history response');
    }
    if ((observedPointCount < 2 && netChange != null) ||
        (observedPointCount >= 2 && netChange == null)) {
      throw const FormatException('Inconsistent follower growth change');
    }
    if (availableFrom != null &&
        observedPoints.any((point) => point.date.isBefore(availableFrom))) {
      throw const FormatException('Follower growth predates availability');
    }
    if (availableFrom != null &&
        !availableFrom.isBefore(rangeStart) &&
        (observedPoints.isEmpty ||
            observedPoints.first.date != availableFrom)) {
      throw const FormatException(
        'Follower growth availability does not match observations',
      );
    }
    if (observedPointCount >= 2 &&
        netChange != observedPoints.last.count! - observedPoints.first.count!) {
      throw const FormatException('Incorrect follower growth change');
    }
    if (latestSnapshotDate != null) {
      if (latestSnapshotDate.isAfter(rangeEnd)) {
        throw const FormatException('Latest follower growth is out of range');
      }
      if (observedPoints.isNotEmpty &&
          latestSnapshotDate != observedPoints.last.date) {
        throw const FormatException(
          'Latest follower growth does not identify latest observation',
        );
      }
      if (!latestSnapshotDate.isBefore(rangeStart)) {
        final latestPoint =
            points[latestSnapshotDate.difference(rangeStart).inDays];
        if (latestPoint.count != latestFollowerCount) {
          throw const FormatException('Latest follower growth does not match');
        }
      }
    }

    return FollowerGrowth(
      period: period,
      rangeStart: rangeStart,
      rangeEnd: rangeEnd,
      availableFrom: availableFrom,
      latestSnapshotDate: latestSnapshotDate,
      latestCapturedAt: latestCapturedAt,
      latestFollowerCount: latestFollowerCount,
      netChange: netChange,
      points: List.unmodifiable(points),
    );
  }

  final FollowerGrowthPeriod period;
  final DateTime rangeStart;
  final DateTime rangeEnd;
  final DateTime? availableFrom;
  final DateTime? latestSnapshotDate;
  final DateTime? latestCapturedAt;
  final int? latestFollowerCount;
  final int? netChange;
  final List<FollowerGrowthPoint> points;
}

DateTime _previousYearAnniversary(DateTime end) {
  final previousYear = end.year - 1;
  final day = end.month == DateTime.february && end.day == 29 ? 28 : end.day;
  return DateTime.utc(previousYear, end.month, day);
}

final _dateOnlyPattern = RegExp(r'^(\d{4})-(\d{2})-(\d{2})$');

DateTime _parseDateOnly(Object? value, String field) {
  if (value is! String) {
    throw FormatException('Invalid $field');
  }
  final match = _dateOnlyPattern.firstMatch(value);
  if (match == null) {
    throw FormatException('Invalid $field');
  }
  final year = int.parse(match.group(1)!);
  final month = int.parse(match.group(2)!);
  final day = int.parse(match.group(3)!);
  final date = DateTime.utc(year, month, day);
  if (date.year != year || date.month != month || date.day != day) {
    throw FormatException('Invalid $field');
  }
  return date;
}

DateTime? _nullableDateOnly(Object? value, String field) =>
    value == null ? null : _parseDateOnly(value, field);

DateTime? _nullableTimestamp(Object? value, String field) {
  if (value == null) return null;
  if (value is! String || !RegExp(r'(Z|[+-]\d{2}:\d{2})$').hasMatch(value)) {
    throw FormatException('Invalid $field');
  }
  return DateTime.tryParse(value)?.toUtc() ??
      (throw FormatException('Invalid $field'));
}

int? _nullableInt(Object? value, String field) {
  if (value == null) return null;
  if (value is! int) throw FormatException('Invalid $field');
  return value;
}

int? _nullableNonNegativeInt(Object? value, String field) {
  final parsed = _nullableInt(value, field);
  if (parsed != null && parsed < 0) {
    throw FormatException('Invalid $field');
  }
  return parsed;
}

String formatFollowerCount(int count, String locale) =>
    NumberFormat.decimalPattern(locale).format(count);

String formatCompactFollowerCount(int count, String locale) =>
    NumberFormat.compact(locale: locale).format(count);

String formatFollowerGrowthDate(DateTime date, String locale) =>
    DateFormat.yMd(locale).format(date);
