import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:intl/intl.dart';
import 'package:l10n_countries/l10n_countries.dart';
import 'package:sealed_countries/sealed_countries.dart';

abstract final class BusinessFormatters {
  static final _timeZones = BusinessTimeZoneService.initialized();

  static String? money(BusinessPrice? price, String locale) {
    if (price == null) return null;
    final amount = num.tryParse(price.amount);
    if (amount == null) return null;
    return NumberFormat.simpleCurrency(
      locale: locale,
      name: price.currency,
    ).format(amount);
  }

  static BusinessEventDisplay event(
    BusinessEvent event,
    AppLocalizations l10n,
  ) {
    final requestedZone = event.timeZone ?? 'UTC';
    final zone = _timeZones.contains(requestedZone) ? requestedZone : 'UTC';
    final start = _timeZones.toLocal(zone, event.startsAt);
    final end = _timeZones.toLocal(zone, event.endsAt);
    if (event.isAllDay) {
      // All-day ends are exclusive calendar dates. Subtracting 24 hours can
      // land on the wrong date across daylight-saving changes.
      final inclusiveEnd = DateTime.utc(end.year, end.month, end.day - 1);
      return BusinessEventDisplay(
        date: _dateRange(start, inclusiveEnd, l10n),
        time: l10n.businessEventAllDayDisplay,
      );
    }
    return BusinessEventDisplay(
      date: _dateRange(start, end, l10n),
      time: l10n.businessEventTimeRange(
        _time(start, l10n.localeName),
        _time(end, l10n.localeName),
      ),
    );
  }

  static String location(
    BusinessLocation location,
    AppLocalizations l10n,
  ) {
    final country = WorldCountry.maybeFromCodeShort(location.country);
    final localized = country == null
        ? null
        : CountriesLocaleMapper()
              .localize(
                {country.code},
                mainLocale: l10n.localeName,
                fallbackLocale: 'en',
              )
              .values
              .firstOrNull;
    final countryLabel = localized ?? location.country;
    final locality = location.locality?.trim();
    return locality == null || locality.isEmpty
        ? countryLabel
        : l10n.businessLocationValue(locality, countryLabel);
  }

  static String _dateRange(
    DateTime start,
    DateTime end,
    AppLocalizations l10n,
  ) {
    if (start.year == end.year &&
        start.month == end.month &&
        start.day == end.day) {
      return DateFormat.yMMMd(l10n.localeName).format(start);
    }
    final startText =
        (start.year == end.year
                ? DateFormat.MMMd(l10n.localeName)
                : DateFormat.yMMMd(l10n.localeName))
            .format(start);
    final endText = DateFormat.MMMd(l10n.localeName).format(end);
    return l10n.businessEventDateRange(startText, endText, end.year);
  }

  static String _time(DateTime value, String locale) => DateFormat.jm(
    locale,
  ).format(value).replaceAll(RegExp('[\u00a0\u202f]'), ' ');
}

final class BusinessEventDisplay {
  const new({required this.date, required this.time});

  final String date;
  final String time;
}
