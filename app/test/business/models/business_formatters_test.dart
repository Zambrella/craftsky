import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_formatters.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/l10n/generated/app_localizations_en.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:intl/date_symbol_data_local.dart';

void main() {
  setUpAll(() => initializeDateFormatting('en_GB'));

  group('UT-013 locale-aware business formatters', () {
    test('formats seller-authored money without changing canonical values', () {
      const usd = BusinessPrice(amount: '1234.5', currency: 'USD');
      const jpy = BusinessPrice(amount: '1200', currency: 'JPY');

      expect(BusinessFormatters.money(usd, 'en_US'), r'$1,234.50');
      expect(BusinessFormatters.money(usd, 'en_GB'), r'$1,234.50');
      expect(BusinessFormatters.money(jpy, 'en_US'), '¥1,200');
      expect(usd.amount, '1234.5');
      expect(usd.currency, 'USD');
    });

    test('formats timed and exclusive-end all-day ranges for locale', () {
      final us = AppLocalizationsEn('en_US');
      final gb = AppLocalizationsEn('en_GB');
      final timed = _event(
        startsAt: DateTime(2026, 8, 30, 9),
        endsAt: DateTime(2026, 8, 30, 17, 30),
      );
      final allDay = _event(
        startsAt: DateTime(2026, 8, 30),
        endsAt: DateTime(2026, 9, 2),
        isAllDay: true,
      );

      expect(BusinessFormatters.event(timed, us).date, 'Aug 30, 2026');
      expect(BusinessFormatters.event(timed, us).time, '9:00 AM–5:30 PM');
      expect(BusinessFormatters.event(timed, gb).date, '30 Aug 2026');
      expect(BusinessFormatters.event(timed, gb).time, '09:00–17:30');
      expect(
        BusinessFormatters.event(allDay, us).date,
        'Aug 30–Sep 1, 2026',
      );
      expect(BusinessFormatters.event(allDay, us).time, 'All day');
      expect(allDay.endsAt, DateTime(2026, 9, 2));
    });

    test('localizes hydrated country and preserves locality', () {
      final l10n = AppLocalizationsEn('en_GB');

      expect(
        BusinessFormatters.location(
          const BusinessLocation(country: 'US', locality: 'Portland'),
          l10n,
        ),
        'Portland, United States',
      );
    });
  });
}

BusinessEvent _event({
  required DateTime startsAt,
  required DateTime endsAt,
  bool isAllDay = false,
}) => BusinessEvent(
  did: 'did:plc:formatter',
  rkey: 'event',
  cid: 'cid',
  uri: 'at://did:plc:formatter/social.craftsky.business.event/event',
  name: 'Event',
  startsAt: startsAt,
  endsAt: endsAt,
  roles: const [BusinessOpenValue(value: 'vendor', known: true)],
  status: const BusinessOpenValue(value: 'scheduled', known: true),
  isAllDay: isAllDay,
  past: false,
  createdAt: DateTime(2026),
  publicSuppressionReasons: const [],
  upcomingExclusionReasons: const [],
);
