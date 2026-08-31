import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final zones = BusinessTimeZoneService.initialized();

  test('UT-008 converts timed local values to whole-second UTC', () {
    final draft = _draft(
      timeZone: 'Asia/Tokyo',
      startsAt: DateTime(2026, 9, 5, 10, 30, 15, 999),
      endsAt: DateTime(2026, 9, 5, 12, 0, 0, 500),
    );

    final range = draft.utcRange(zones);

    expect(range.startsAt, '2026-09-05T01:30:15Z');
    expect(range.endsAt, '2026-09-05T03:00:00Z');
  });

  test('UT-008 keeps all-day spring DST boundaries at local midnight', () {
    final draft = _draft(
      timeZone: 'Europe/London',
      startsAt: DateTime(2026, 3, 28),
      endsAt: DateTime(2026, 3, 30),
      isAllDay: true,
    );

    final range = draft.utcRange(zones);

    expect(range.startsAt, '2026-03-28T00:00:00Z');
    expect(range.endsAt, '2026-03-29T23:00:00Z');
  });

  test('UT-008 keeps all-day fall DST boundaries at local midnight', () {
    final draft = _draft(
      timeZone: 'America/New_York',
      startsAt: DateTime(2026, 10, 31),
      endsAt: DateTime(2026, 11, 2),
      isAllDay: true,
    );

    final range = draft.utcRange(zones);

    expect(range.startsAt, '2026-10-31T04:00:00Z');
    expect(range.endsAt, '2026-11-02T05:00:00Z');
  });

  test(
    'UT-008 rejects invalid zone, range, duration, and non-midnight day',
    () {
      expect(
        _draft(timeZone: 'BST').validate(zones),
        contains(EventDraftError.timeZoneInvalid),
      );
      expect(
        _draft(endsAt: DateTime(2026, 9, 5, 9)).validate(zones),
        contains(EventDraftError.endNotAfterStart),
      );
      expect(
        _draft(endsAt: DateTime(2026, 10, 7, 10, 1)).validate(zones),
        contains(EventDraftError.durationTooLong),
      );
      expect(
        _draft(isAllDay: true).validate(zones),
        contains(EventDraftError.allDayBoundaryInvalid),
      );
    },
  );

  test('UT-008 validates approved required catalogs, links, and bounds', () {
    final errors = _draft(
      name: '',
      roles: const [],
      mode: 'future-mode',
      status: 'future-status',
      summary: 's' * 1001,
      venueName: 'v' * 201,
      eventUri: 'http://event.example',
      registrationUri: 'http://event.example',
    ).validate(zones);

    expect(
      errors,
      containsAll(const {
        EventDraftError.nameRequired,
        EventDraftError.rolesInvalid,
        EventDraftError.modeInvalid,
        EventDraftError.statusInvalid,
        EventDraftError.summaryTooLong,
        EventDraftError.venueNameTooLong,
        EventDraftError.eventUriInvalid,
        EventDraftError.registrationUriInvalid,
      }),
    );
  });
}

BusinessEventDraft _draft({
  String name = 'Fibre fair',
  DateTime? startsAt,
  DateTime? endsAt,
  List<String> roles = const ['vendor'],
  String mode = 'in-person',
  String status = 'scheduled',
  String timeZone = 'UTC',
  bool isAllDay = false,
  String? summary,
  String? venueName,
  String? eventUri,
  String? registrationUri,
}) => BusinessEventDraft(
  name: name,
  startsAt: startsAt ?? DateTime(2026, 9, 5, 10),
  endsAt: endsAt ?? DateTime(2026, 9, 5, 12),
  roles: roles,
  mode: mode,
  status: status,
  timeZone: timeZone,
  isAllDay: isAllDay,
  summary: summary,
  venueName: venueName,
  eventUri: eventUri,
  registrationUri: registrationUri,
);
