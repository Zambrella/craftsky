import 'dart:js_interop';
import 'dart:js_interop_unsafe';

import 'package:craftsky_app/service_status/data/service_status_repository.dart';
import 'package:craftsky_app/service_status/data/service_status_repository_web.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:web/web.dart' as web;

void main() {
  // IT-006: connection loss during the body is expected polling failure.
  test(
    'body network failure is expected and has a bounded summary',
    () async {
      final source = JSObject()
        ..setProperty(
          'start'.toJS,
          ((web.ReadableStreamDefaultController controller) {
            controller.error('body-network-canary'.toJS);
          }).toJS,
        );
      final repository = WebServiceStatusRepository(
        Uri.parse('https://status.example.test/app.json'),
        fetch: (_, _) async => web.Response(
          web.ReadableStream(source),
          web.ResponseInit(
            status: 200,
            headers: web.Headers()..set('content-type', 'application/json'),
          ),
        ),
      );
      await expectLater(
        repository.start().result,
        throwsA(isA<StatusNetworkFailure>()),
      );
      repository.close();
    },
  );

  // IT-001 / RULE-002, RULE-003 / AC-019. Hosted CORS/cookies are IT-007.
  test(
    'Fetch opts out of credentials, redirects, cache and referrer',
    () async {
      var calls = 0;
      final repository = WebServiceStatusRepository(
        Uri.parse('https://status.example.test/app.json'),
        fetch: (uri, init) async {
          calls++;
          expect(uri, 'https://status.example.test/app.json');
          expect(init.credentials, 'omit');
          expect(init.redirect, 'error');
          expect(init.cache, 'no-store');
          expect(init.referrerPolicy, 'no-referrer');
          expect(web.Headers(init.headers).get('authorization'), isNull);
          expect(web.Headers(init.headers).get('cookie'), isNull);
          return web.Response(
            '{"schemaVersion":1,"mode":"normal","revision":"A"}'.toJS,
            web.ResponseInit(
              status: 200,
              headers: web.Headers()..set('content-type', 'application/json'),
            ),
          );
        },
      );
      final document = await repository.start().result;
      expect(document.revision, 'A');
      expect(calls, 1);
      repository.close();
    },
  );
}
