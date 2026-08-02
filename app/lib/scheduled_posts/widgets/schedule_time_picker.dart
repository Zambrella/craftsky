import 'package:flutter/material.dart';

typedef ScheduleDateSelector =
    Future<DateTime?> Function({
      required DateTime initialDate,
      required DateTime firstDate,
      required DateTime lastDate,
    });

typedef ScheduleTimeSelector =
    Future<TimeOfDay?> Function({required TimeOfDay initialTime});

Future<DateTime?> pickScheduledLocalTime({
  required BuildContext context,
  required DateTime now,
  ScheduleDateSelector? selectDate,
  ScheduleTimeSelector? selectTime,
  ValueChanged<DateTime>? onOutOfRange,
}) async {
  final window = ScheduleTimeWindow.fromNow(now);
  final datePicker =
      selectDate ??
      ({required initialDate, required firstDate, required lastDate}) =>
          showDatePicker(
            context: context,
            initialDate: initialDate,
            firstDate: firstDate,
            lastDate: lastDate,
          );
  final timePicker =
      selectTime ??
      ({required initialTime}) => showTimePicker(
        context: context,
        initialTime: initialTime,
      );

  final date = await datePicker(
    initialDate: window.earliest,
    firstDate: DateTime(
      window.earliest.year,
      window.earliest.month,
      window.earliest.day,
    ),
    lastDate: DateTime(
      window.latest.year,
      window.latest.month,
      window.latest.day,
    ),
  );
  if (date == null || !context.mounted) return null;

  final time = await timePicker(
    initialTime: TimeOfDay.fromDateTime(window.earliest),
  );
  if (time == null || !context.mounted) return null;

  final selected = DateTime(
    date.year,
    date.month,
    date.day,
    time.hour,
    time.minute,
  );
  if (!window.contains(selected)) {
    onOutOfRange?.call(selected);
    return null;
  }
  return selected;
}

final class ScheduleTimeWindow {
  const ScheduleTimeWindow({required this.earliest, required this.latest});

  factory ScheduleTimeWindow.fromNow(DateTime now) => ScheduleTimeWindow(
    earliest: _ceilToMinute(now.add(const Duration(minutes: 5))),
    latest: _floorToMinute(now.add(const Duration(days: 28))),
  );

  final DateTime earliest;
  final DateTime latest;

  bool contains(DateTime value) =>
      value.second == 0 &&
      value.millisecond == 0 &&
      value.microsecond == 0 &&
      !value.isBefore(earliest) &&
      !value.isAfter(latest);
}

DateTime _ceilToMinute(DateTime value) {
  final minute = DateTime(
    value.year,
    value.month,
    value.day,
    value.hour,
    value.minute,
  );
  return value == minute ? minute : minute.add(const Duration(minutes: 1));
}

DateTime _floorToMinute(DateTime value) => DateTime(
  value.year,
  value.month,
  value.day,
  value.hour,
  value.minute,
);
