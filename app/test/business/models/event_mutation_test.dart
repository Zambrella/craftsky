import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final zones = BusinessTimeZoneService.initialized();

  test('UT-009 serializes every writable create field only', () {
    final body = _completeDraft().toCreateJson(zones);

    expect(body, {
      'name': 'Fibre fair',
      'startsAt': '2026-09-05T09:00:00Z',
      'endsAt': '2026-09-05T11:00:00Z',
      'roles': ['organizer', 'vendor'],
      'mode': 'hybrid',
      'status': 'scheduled',
      'timeZone': 'Europe/London',
      'isAllDay': false,
      'summary': 'Meet local makers',
      'venueName': 'Guild Hall',
      'eventUri': 'https://event.example/details',
      'registrationUri': 'https://event.example/register',
      'image': {
        'image': {
          r'$type': 'blob',
          'ref': {r'$link': 'bafy-image'},
          'mimeType': 'image/webp',
          'size': 1234,
        },
        'alt': 'Stalls at the fair',
        'aspectRatio': {'width': 4, 'height': 3},
      },
    });
    expect(
      body.keys,
      isNot(containsAll(['did', 'rkey', 'uri', 'cid', 'createdAt', 'past'])),
    );
  });

  test(
    'UT-009 update preserves exact stored blob and omits cleared fields',
    () {
      final event = _event();
      final draft = BusinessEventDraft.fromEvent(event).copyWith(
        status: 'postponed',
        summary: null,
        venueName: null,
        eventUri: null,
        registrationUri: null,
      );

      final body = draft.toUpdateJson(zones);

      expect(body['status'], 'postponed');
      expect(body['image'], ExistingBusinessImageDraft(event.image!).toJson());
      expect(
        body.keys,
        isNot(
          containsAll([
            'summary',
            'venueName',
            'eventUri',
            'registrationUri',
            'createdAt',
            'publicSuppressionReasons',
            'upcomingExclusionReasons',
          ]),
        ),
      );
    },
  );
}

BusinessEventDraft _completeDraft() => BusinessEventDraft(
  name: 'Fibre fair',
  startsAt: DateTime(2026, 9, 5, 10),
  endsAt: DateTime(2026, 9, 5, 12),
  roles: const ['organizer', 'vendor'],
  mode: 'hybrid',
  status: 'scheduled',
  timeZone: 'Europe/London',
  isAllDay: false,
  summary: 'Meet local makers',
  venueName: 'Guild Hall',
  eventUri: 'https://event.example/details',
  registrationUri: 'https://event.example/register',
  image: UploadedBusinessImageDraft(
    cid: 'bafy-image',
    mime: 'image/webp',
    size: 1234,
    alt: 'Stalls at the fair',
    aspectRatio: BusinessImageAspectRatio(width: 4, height: 3),
  ),
);

BusinessEvent _event() => BusinessEvent(
  did: 'did:plc:owner',
  rkey: '3m4event',
  uri: 'at://did:plc:owner/social.craftsky.business.event/3m4event',
  cid: 'bafy-current',
  name: 'Fibre fair',
  startsAt: DateTime.parse('2026-09-05T09:00:00Z'),
  endsAt: DateTime.parse('2026-09-05T11:00:00Z'),
  roles: const [BusinessOpenValue(value: 'vendor', known: true)],
  mode: const BusinessOpenValue(value: 'in-person', known: true),
  status: const BusinessOpenValue(value: 'scheduled', known: true),
  timeZone: 'Europe/London',
  isAllDay: false,
  image: BusinessImageView(
    cid: 'bafy-saved',
    mime: 'image/jpeg',
    size: 999,
    alt: 'Saved image',
    aspectRatio: BusinessImageAspectRatio(width: 16, height: 9),
    thumb: 'https://cdn.example/thumb',
    fullsize: 'https://cdn.example/full',
  ),
  createdAt: DateTime.parse('2026-08-30T12:00:00Z'),
  past: false,
  publicSuppressionReasons: const ['diagnostic'],
  upcomingExclusionReasons: const ['diagnostic'],
);
