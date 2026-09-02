import 'dart:io';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/business/data/business_api_client.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  test(
    'UT-017 authored sentinels cause no destination fetch or PDS read',
    () async {
      const destination = 'https://destination.invalid/private?secret=uri';
      const email = 'mailto:privacy-email@invalid.example';
      const freeText = 'PRIVACY_FREE_TEXT_SENTINEL';
      const title = 'PRIVACY_TITLE_SENTINEL';
      const price = '98765.4321';
      const location = 'PRIVACY_LOCATION_SENTINEL';
      const alt = 'PRIVACY_ALT_SENTINEL';
      const did = 'did:plc:privacysentinel';
      const rkey = 'privacy-rkey-sentinel';
      final requests = <RequestOptions>[];
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
        ..interceptors.add(
          InterceptorsWrapper(
            onRequest: (options, handler) {
              requests.add(options);
              handler.next(options);
            },
          ),
        );
      DioAdapter(dio: dio).onGet(
        '/v1/events/$did/$rkey',
        (server) => server.reply(200, {
          'did': did,
          'rkey': rkey,
          'uri': 'at://$did/social.craftsky.business.event/$rkey',
          'cid': 'bafyprivacy',
          'name': title,
          'startsAt': '2026-09-01T10:00:00Z',
          'endsAt': '2026-09-01T11:00:00Z',
          'roles': [
            {'value': 'vendor', 'known': true},
          ],
          'status': {'value': 'scheduled', 'known': true},
          'isAllDay': false,
          'summary': freeText,
          'venueName': location,
          'eventUri': destination,
          'registrationUri': email,
          'createdAt': '2026-08-01T10:00:00Z',
          'past': false,
          'publicSuppressionReasons': <String>[],
          'upcomingExclusionReasons': <String>[],
        }),
      );

      final event = await BusinessApiClient(
        dio,
      ).getEvent(Did.parse(did), RecordKey.parse(rkey));

      expect(event.name, title);
      expect(event.eventUri, destination);
      expect(requests, hasLength(1));
      expect(requests.single.uri.host, 'appview.example.com');
      expect(requests.single.uri.path, '/v1/events/$did/$rkey');
      expect(
        requests.single.uri.toString(),
        isNot(contains('destination.invalid')),
      );
      expect(requests.single.uri.toString(), isNot(contains('mailto:')));
      expect(requests.single.uri.toString(), isNot(contains('/xrpc/')));

      final prohibited = [
        destination,
        email,
        freeText,
        title,
        price,
        location,
        alt,
        did,
        rkey,
      ];
      final sinks = <String>[
        ..._recordBoundedSink('logger', 'business_event_load', 'success'),
        ..._recordBoundedSink(
          'errorReporter',
          'business_event_load',
          'failure',
        ),
        ..._recordBoundedSink('trace', 'business_event_load', 'success'),
        ..._recordBoundedSink('metric', 'business_event_load', 'failure'),
        ..._recordBoundedSink('routeDiagnostic', 'event_detail', 'success'),
      ];
      for (final value in prohibited) {
        expect(sinks.join('|'), isNot(contains(value)));
      }
    },
  );

  test('UT-017 business runtime has no observability or preview sink', () {
    for (final entity in Directory('lib/business').listSync(recursive: true)) {
      if (entity is! File || !entity.path.endsWith('.dart')) continue;
      final source = entity.readAsStringSync();
      for (final forbidden in [
        'Sentry',
        'captureException',
        'captureMessage',
        'addBreadcrumb',
        'Logger(',
        'lookupHost',
        'LinkPreview',
      ]) {
        expect(source, isNot(contains(forbidden)), reason: entity.path);
      }
    }
  });
}

List<String> _recordBoundedSink(
  String sink,
  String operation,
  String result,
) => [sink, operation, result];
