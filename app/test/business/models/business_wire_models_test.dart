import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('decodes full-profile business data and blocked omission safely', () {
    final profile = ProfileMapper.fromMap({
      'did': 'did:plc:business',
      'handle': 'business.example',
      'crafts': <String>[],
      'accountType': 'business',
      'business': {
        'cid': 'bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq',
        'businessTypes': [
          {'value': 'dyer', 'known': true},
          {'value': 'future-catalog-value', 'known': false},
        ],
        'tagline': 'Small-batch colour.',
        'products': [
          {
            'title': 'First skein',
            'uri': 'https://shop.example/first',
            'image': {
              'cid':
                  'bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq',
              'mime': 'image/jpeg',
              'size': 101,
              'alt': 'Rust yarn',
              'aspectRatio': {'width': 4, 'height': 3},
              'thumb': 'https://cdn.example/thumb.jpeg',
              'fullsize': 'https://cdn.example/full.jpeg',
            },
          },
        ],
      },
    });

    expect(profile.accountType, AccountType.business);
    expect(profile.business?.cid.toString(), startsWith('bafy'));
    expect(profile.business?.businessTypes, hasLength(2));
    expect(profile.business?.businessTypes.last.known, isFalse);
    final image = profile.business?.products.single.image;
    expect(image?.mime, 'image/jpeg');
    expect(image?.size, 101);
    expect(image?.aspectRatio?.width, 4);
    expect(image?.thumb, 'https://cdn.example/thumb.jpeg');

    final blocked = ProfileMapper.fromMap({
      'did': 'did:plc:blocked',
      'handle': 'blocked.example',
      'crafts': <String>[],
      'blocking': true,
    });
    expect(blocked.accountType, isNull);
    expect(blocked.business, isNull);

    final compact = ProfileAccountSummaryMapper.fromMap({
      'did': 'did:plc:compact',
      'handle': 'compact.example',
      'isCraftskyProfile': true,
      'accountType': 'business',
    });
    expect(compact.did.toString(), 'did:plc:compact');
  });

  test('rejects a malformed required business image field', () {
    expect(
      () => BusinessImageViewMapper.fromMap({
        'cid': 'bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq',
        'mime': 'image/jpeg',
        'size': -1,
        'alt': '',
        'thumb': 'https://cdn.example/thumb.jpeg',
        'fullsize': 'https://cdn.example/full.jpeg',
      }),
      throwsA(isA<MapperException>()),
    );
  });

  test('decodes typed event identities and reuses the business image view', () {
    final page = BusinessEventPageMapper.fromMap({
      'items': [
        {
          'did': 'did:plc:business',
          'rkey': '3m4event',
          'uri':
              'at://did:plc:business/social.craftsky.business.event/3m4event',
          'cid': 'bafyreievent',
          'name': 'Summer fibre fair',
          'startsAt': '2026-09-05T09:00:00Z',
          'endsAt': '2026-09-05T17:00:00Z',
          'roles': [
            {'value': 'vendor', 'known': true},
            {'value': 'future-role', 'known': false},
          ],
          'mode': {'value': 'in-person', 'known': true},
          'status': {'value': 'scheduled', 'known': true},
          'timeZone': 'Europe/London',
          'isAllDay': false,
          'summary': 'Independent makers and growers.',
          'venueName': 'Town Hall',
          'eventUri': 'https://events.example/fair',
          'registrationUri': 'https://events.example/fair/register',
          'image': {
            'cid': 'bafyreieventimage',
            'mime': 'image/webp',
            'size': 0,
            'alt': '',
            'thumb': 'https://cdn.example/event-thumb.webp',
            'fullsize': 'https://cdn.example/event.webp',
          },
          'createdAt': '2026-08-30T12:00:00Z',
          'past': false,
          'publicSuppressionReasons': <String>[],
          'upcomingExclusionReasons': ['record-moderated'],
        },
      ],
      'cursor': 'opaque:event:cursor',
    });

    final event = page.items.single;
    expect(event.did.toString(), 'did:plc:business');
    expect(event.rkey.toString(), '3m4event');
    expect(
      event.uri.toString(),
      'at://did:plc:business/social.craftsky.business.event/3m4event',
    );
    expect(event.cid.toString(), 'bafyreievent');
    expect(event.startsAt, DateTime.utc(2026, 9, 5, 9));
    expect(event.roles.last.known, isFalse);
    expect(event.image, isA<BusinessImageView>());
    expect(event.image?.size, 0);
    expect(event.publicSuppressionReasons, isEmpty);
    expect(event.upcomingExclusionReasons, ['record-moderated']);
    expect(page.cursor, 'opaque:event:cursor');
  });
}
