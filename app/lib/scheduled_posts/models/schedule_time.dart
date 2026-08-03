final class ScheduledInstant {
  ScheduledInstant(DateTime instant) : utc = instant.toUtc();

  final DateTime utc;

  String get apiValue => utc.toIso8601String();

  ScheduleTimeDisplay displayIn({
    required String zoneName,
    required Duration offset,
  }) {
    return ScheduleTimeDisplay(
      instantUtc: utc,
      wallTime: utc.add(offset),
      zoneLabel: '${zoneName.trim()} (${_formatUtcOffset(offset)})',
    );
  }
}

final class ScheduleTimeDisplay {
  const ScheduleTimeDisplay({
    required this.instantUtc,
    required this.wallTime,
    required this.zoneLabel,
  });

  final DateTime instantUtc;
  final DateTime wallTime;
  final String zoneLabel;
}

String _formatUtcOffset(Duration offset) {
  final negative = offset.isNegative;
  final totalMinutes = offset.inMinutes.abs();
  final hours = totalMinutes ~/ Duration.minutesPerHour;
  final minutes = totalMinutes.remainder(Duration.minutesPerHour);
  final sign = negative ? '-' : '+';
  final paddedHours = hours.toString().padLeft(2, '0');
  final paddedMinutes = minutes.toString().padLeft(2, '0');
  return 'UTC$sign$paddedHours:$paddedMinutes';
}
