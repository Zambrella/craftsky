import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/business/data/api_business_repository.dart';
import 'package:craftsky_app/business/data/business_api_client.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  test('IT-008 typed repository sends exact create and update JSON', () async {
    final draft = BusinessEventDraft(
      name: 'Fibre fair',
      startsAt: DateTime(2026, 9, 5, 10),
      endsAt: DateTime(2026, 9, 5, 12),
      roles: const ['vendor'],
      mode: 'in-person',
      status: 'scheduled',
      timeZone: 'Europe/London',
      isAllDay: false,
    );
    const body = <String, dynamic>{
      'name': 'Fibre fair',
      'startsAt': '2026-09-05T09:00:00Z',
      'endsAt': '2026-09-05T11:00:00Z',
      'roles': ['vendor'],
      'mode': 'in-person',
      'status': 'scheduled',
      'timeZone': 'Europe/London',
      'isAllDay': false,
    };
    final owner = Did.parse('did:plc:owner');
    final rkey = RecordKey.parse('3m4event');

    final createDio = _dio();
    DioAdapter(dio: createDio).onPost(
      '/v1/events',
      (server) => server.reply(201, {
        'did': owner.toString(),
        'rkey': rkey.toString(),
        'uri': 'at://did:plc:owner/social.craftsky.business.event/3m4event',
        'cid': 'bafy-created',
      }),
      data: body,
    );
    await ApiBusinessRepository(
      BusinessApiClient(createDio),
      BusinessTimeZoneService.initialized(),
    ).createEvent(draft);

    final updateDio = _dio();
    DioAdapter(dio: updateDio).onPut(
      '/v1/events/did:plc:owner/3m4event',
      (server) => server.reply(200, {'cid': 'bafy-updated'}),
      data: body,
      headers: {'If-Match': 'bafy-current'},
    );
    await ApiBusinessRepository(
      BusinessApiClient(updateDio),
      BusinessTimeZoneService.initialized(),
    ).updateEvent(owner, rkey, Cid.parse('bafy-current'), draft);
  });
}

Dio _dio() => Dio(BaseOptions(baseUrl: 'https://appview.example.com'));
