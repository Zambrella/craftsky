import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:timezone/data/latest.dart' as timezone_data;
import 'package:timezone/timezone.dart' as timezone;

final businessTimeZoneServiceProvider = Provider<BusinessTimeZoneService>(
  (_) => BusinessTimeZoneService.initialized(),
);

class BusinessTimeZoneService {
  BusinessTimeZoneService._();

  factory BusinessTimeZoneService.initialized() {
    timezone_data.initializeTimeZones();
    return BusinessTimeZoneService._();
  }

  List<String> get names {
    final values = {'UTC', ...timezone.timeZoneDatabase.locations.keys}.toList()
      ..sort();
    return List.unmodifiable(values);
  }

  bool contains(String name) {
    if (name == 'UTC') return true;
    try {
      timezone.getLocation(name);
      return true;
    } on timezone.LocationNotFoundException {
      return false;
    }
  }

  DateTime toUtc(String zone, DateTime local) {
    final location = _location(zone);
    return timezone.TZDateTime(
      location,
      local.year,
      local.month,
      local.day,
      local.hour,
      local.minute,
      local.second,
    ).toUtc();
  }

  DateTime toLocal(String zone, DateTime utc) {
    final value = timezone.TZDateTime.from(utc.toUtc(), _location(zone));
    return DateTime(
      value.year,
      value.month,
      value.day,
      value.hour,
      value.minute,
      value.second,
    );
  }

  timezone.Location _location(String zone) =>
      zone == 'UTC' ? timezone.UTC : timezone.getLocation(zone);
}
