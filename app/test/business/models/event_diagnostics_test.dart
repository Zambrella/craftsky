import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/event_diagnostics.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final l10n = lookupAppLocalizations(const Locale('en'));

  test('UT-016 maps the closed diagnostic catalog to localized copy', () {
    expect(
      EventDiagnostics.localized(
        const [
          'owner-not-business',
          'invalid-time-range',
          'duration-exceeds-limit',
          'record-moderated',
          'ended',
          'cancelled',
          'postponed',
        ],
        l10n,
      ),
      [
        'Your account is not currently presented as a business.',
        'The event’s time range is invalid.',
        'The event is longer than the supported limit.',
        'This event is hidden by moderation.',
        'This event has ended.',
        'This event is cancelled.',
        'This event is postponed.',
      ],
    );
  });

  test('UT-016 omits empty, duplicate, and unknown diagnostics', () {
    expect(
      EventDiagnostics.localized(
        const ['', 'ended', 'ended', 'independent-code'],
        l10n,
      ),
      ['This event has ended.'],
    );
  });

  test('UT-016 diagnostics never become editable event fields', () {
    final draft = BusinessEventDraft(
      name: 'Fibre fair',
      startsAt: DateTime(2026, 9, 5, 10),
      endsAt: DateTime(2026, 9, 5, 12),
      roles: const ['vendor'],
      mode: 'in-person',
      status: 'scheduled',
      timeZone: 'UTC',
      isAllDay: false,
    );

    final json = draft.toUpdateJson(BusinessTimeZoneService.initialized());
    expect(json, isNot(contains('publicSuppressionReasons')));
    expect(json, isNot(contains('upcomingExclusionReasons')));
    expect(json, isNot(contains('diagnostics')));
  });
}
