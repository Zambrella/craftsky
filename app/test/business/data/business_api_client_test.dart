import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/business/data/business_api_client.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  Dio buildDio() =>
      Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
        ..interceptors.add(const ErrorMappingInterceptor());

  Map<String, dynamic> eventJson() => {
    'did': 'did:plc:business',
    'rkey': '3m4event',
    'uri': 'at://did:plc:business/social.craftsky.business.event/3m4event',
    'cid': 'bafy-event',
    'name': 'Summer fibre fair',
    'startsAt': '2026-09-05T09:00:00Z',
    'endsAt': '2026-09-05T17:00:00Z',
    'roles': [
      {'value': 'vendor', 'known': true},
    ],
    'mode': {'value': 'in-person', 'known': true},
    'status': {'value': 'scheduled', 'known': true},
    'timeZone': 'Europe/London',
    'isAllDay': false,
    'createdAt': '2026-08-30T12:00:00Z',
    'past': false,
    'publicSuppressionReasons': <String>[],
    'upcomingExclusionReasons': <String>[],
  };

  test('updates account type with the exact camelCase body', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onPut(
      '/v1/profiles/me/account-type',
      (server) => server.reply(200, {'accountType': 'business'}),
      data: {'accountType': 'business'},
    );

    final result = await BusinessApiClient(
      dio,
    ).updateAccountType(AccountType.business);

    expect(result, AccountType.business);
  });

  test('creates and replaces a complete declaration with If-Match', () async {
    const body = <String, dynamic>{
      'businessTypes': ['dyer'],
      'offerings': ['yarn'],
      'products': <Map<String, dynamic>>[],
    };
    final createDio = buildDio();
    DioAdapter(dio: createDio).onPut(
      '/v1/profiles/me/business',
      (server) => server.reply(200, {'cid': 'bafy-created'}),
      data: body,
      headers: {'If-Match': '*'},
    );

    final created = await BusinessApiClient(
      createDio,
    ).putBusinessProfile(body, expectedCid: null);
    expect(created.cid.toString(), 'bafy-created');

    final replaceDio = buildDio();
    DioAdapter(dio: replaceDio).onPut(
      '/v1/profiles/me/business',
      (server) => server.reply(200, {'cid': 'bafy-replaced'}),
      data: body,
      headers: {'If-Match': 'bafy-current'},
    );

    final replaced = await BusinessApiClient(replaceDio).putBusinessProfile(
      body,
      expectedCid: Cid.parse('bafy-current'),
    );
    expect(replaced.cid.toString(), 'bafy-replaced');
  });

  test('maps declaration conflicts through the standard API error path', () {
    final dio = buildDio();
    DioAdapter(dio: dio).onPut(
      '/v1/profiles/me/business',
      (server) => server.reply(409, {
        'error': 'pds_record_conflict',
        'message': 'record changed',
        'requestId': 'request-1',
      }),
      data: <String, dynamic>{},
      headers: {'If-Match': 'bafy-stale'},
    );

    expect(
      BusinessApiClient(dio).putBusinessProfile(
        const <String, dynamic>{},
        expectedCid: Cid.parse('bafy-stale'),
      ),
      throwsA(
        isA<ApiBadRequest>().having(
          (error) => error.code,
          'code',
          'pds_record_conflict',
        ),
      ),
    );
  });

  test('lists profile events with an opaque cursor unchanged', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onGet(
      '/v1/profiles/business.example/events',
      (server) => server.reply(200, {
        'items': [eventJson()],
        'cursor': 'opaque:next',
      }),
      queryParameters: {'limit': '12', 'cursor': 'opaque:current'},
    );

    final page = await BusinessApiClient(dio).listProfileEvents(
      AtIdentifier.parse('business.example'),
      limit: 12,
      cursor: 'opaque:current',
    );

    expect(page.items.single.did.toString(), 'did:plc:business');
    expect(page.cursor, 'opaque:next');
  });

  test('lists each owner event partition with its exact filter', () async {
    for (final filter in OwnerEventFilter.values) {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/events',
        (server) => server.reply(200, {'items': <Map<String, dynamic>>[]}),
        queryParameters: {
          'filter': filter.name,
          'limit': '20',
          'cursor': 'opaque:${filter.name}',
        },
      );

      final page = await BusinessApiClient(dio).listOwnerEvents(
        filter,
        cursor: 'opaque:${filter.name}',
      );
      expect(page.items, isEmpty);
      expect(page.cursor, isNull);
    }
  });

  test('reads event detail by typed owner DID and record key', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onGet(
      '/v1/events/did:plc:business/3m4event',
      (server) => server.reply(200, eventJson()),
    );

    final event = await BusinessApiClient(dio).getEvent(
      Did.parse('did:plc:business'),
      RecordKey.parse('3m4event'),
    );

    expect(event.cid.toString(), 'bafy-event');
  });

  test('maps exact event detail not-found envelope', () {
    final dio = buildDio();
    DioAdapter(dio: dio).onGet(
      '/v1/events/did:plc:business/missing',
      (server) => server.reply(404, {
        'error': 'event_not_found',
        'message': 'event unavailable',
        'requestId': 'request-event-missing',
      }),
    );

    expect(
      BusinessApiClient(dio).getEvent(
        Did.parse('did:plc:business'),
        RecordKey.parse('missing'),
      ),
      throwsA(
        isA<ApiBadRequest>()
            .having((error) => error.code, 'code', 'event_not_found')
            .having(
              (error) => error.details.requestId,
              'requestId',
              'request-event-missing',
            ),
      ),
    );
  });

  test(
    'creates, updates, and deletes events with complete caller bodies',
    () async {
      const body = <String, dynamic>{
        'name': 'Summer fibre fair',
        'startsAt': '2026-09-05T09:00:00Z',
        'endsAt': '2026-09-05T17:00:00Z',
        'roles': ['vendor'],
        'mode': 'in-person',
        'status': 'scheduled',
        'timeZone': 'Europe/London',
      };
      final owner = Did.parse('did:plc:business');
      final rkey = RecordKey.parse('3m4event');

      final createDio = buildDio();
      DioAdapter(dio: createDio).onPost(
        '/v1/events',
        (server) => server.reply(201, {
          'did': owner.toString(),
          'rkey': rkey.toString(),
          'uri':
              'at://did:plc:business/social.craftsky.business.event/3m4event',
          'cid': 'bafy-created-event',
        }),
        data: body,
      );
      final created = await BusinessApiClient(createDio).createEvent(body);
      expect(created.rkey, rkey);

      final updateDio = buildDio();
      DioAdapter(dio: updateDio).onPut(
        '/v1/events/did:plc:business/3m4event',
        (server) => server.reply(200, {
          'did': owner.toString(),
          'rkey': rkey.toString(),
          'uri':
              'at://did:plc:business/social.craftsky.business.event/3m4event',
          'cid': 'bafy-updated-event',
        }),
        data: body,
        headers: {'If-Match': 'bafy-current-event'},
      );
      final updated = await BusinessApiClient(updateDio).updateEvent(
        owner,
        rkey,
        Cid.parse('bafy-current-event'),
        body,
      );
      expect(updated.cid.toString(), 'bafy-updated-event');

      final deleteDio = buildDio();
      DioAdapter(dio: deleteDio).onDelete(
        '/v1/events/did:plc:business/3m4event',
        (server) => server.reply(200, {
          'did': owner.toString(),
          'rkey': rkey.toString(),
          'uri':
              'at://did:plc:business/social.craftsky.business.event/3m4event',
          'cid': 'bafy-updated-event',
        }),
        headers: {'If-Match': 'bafy-updated-event'},
      );
      final deleted = await BusinessApiClient(deleteDio).deleteEvent(
        owner,
        rkey,
        Cid.parse('bafy-updated-event'),
      );
      expect(deleted.cid.toString(), 'bafy-updated-event');
    },
  );

  test('reports an event through its exact record route', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onPost(
      '/v1/events/did:plc:business/3m4event/reports',
      (server) => server.reply(201, {
        'reportId': 'report-event-1',
        'status': 'accepted',
      }),
      data: {'reasonType': 'spam', 'details': 'Repeated promotions'},
    );

    final result = await BusinessApiClient(dio).reportEvent(
      Did.parse('did:plc:business'),
      RecordKey.parse('3m4event'),
      const ReportSubmission(
        reasonType: 'spam',
        details: 'Repeated promotions',
      ),
    );

    expect(result.reportId, 'report-event-1');
    expect(result.status, 'accepted');
  });
}
