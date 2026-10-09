import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:craftsky_app/service_status/data/service_status_repository.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  // IT-001 / FR-001, RULE-002, RULE-003 / AC-001, AC-019.
  test(
    'status requests are anonymous and reject redirects or invalid bodies',
    () async {
      final received = <HttpRequest>[];
      var redirectCalls = 0;
      final target = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
      target.listen((request) {
        redirectCalls++;
        request.response.statusCode = 200;
        unawaited(request.response.close());
      });
      final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
      var responseKind = 'valid';
      server.listen((request) {
        received.add(request);
        final response = request.response;
        if (responseKind == 'redirect') {
          response
            ..statusCode = 302
            ..headers.set(
              'location',
              'http://127.0.0.1:${target.port}/capture',
            );
        } else if (responseKind == 'html') {
          response.headers.contentType = ContentType.html;
          response.write('<html>proxy</html>');
        } else if (responseKind == 'oversized') {
          response.headers.contentType = ContentType.json;
          response.write(' ' * 16385);
        } else {
          response.headers.contentType = ContentType.json;
          response.write(
            jsonEncode({'schemaVersion': 1, 'mode': 'normal', 'revision': 'A'}),
          );
        }
        unawaited(response.close());
      });
      final repository = createServiceStatusRepository(
        Uri.parse('http://127.0.0.1:${server.port}/app.json'),
      );
      try {
        expect((await repository.start().result).revision, 'A');
        for (final kind in ['redirect', 'html', 'oversized']) {
          responseKind = kind;
          await expectLater(
            repository.start().result,
            throwsA(isA<Exception>()),
          );
        }
        expect(redirectCalls, 0);
        expect(received, hasLength(4));
        for (final request in received) {
          expect(request.headers.value('authorization'), isNull);
          expect(request.headers.value('cookie'), isNull);
          expect(request.headers.value('x-device-id'), isNull);
          expect(request.headers.value('x-dev-did'), isNull);
          expect(request.uri.query, isEmpty);
          expect(
            await request.fold<int>(0, (count, bytes) => count + bytes.length),
            0,
          );
        }
      } finally {
        repository.close();
        await server.close(force: true);
        await target.close(force: true);
      }
    },
  );
  test(
    'endpoint configuration rejects credentials and production overrides',
    () {
      expect(
        validateStatusEndpoint(
          'https://status.craftsky.social/app.json',
          production: true,
        ).host,
        'status.craftsky.social',
      );
      expect(
        validateStatusEndpoint(
          'http://127.0.0.1:1234/app.json',
          production: false,
        ).port,
        1234,
      );
      for (final url in [
        'https://user:secret@status.example/app.json',
        'https://status.example/app.json?account=A',
        'https://status.example/app.json#secret',
        'http://status.example/app.json',
        'file:///tmp/status',
      ]) {
        expect(
          () => validateStatusEndpoint(url, production: false),
          throwsArgumentError,
        );
      }
      expect(
        () => validateStatusEndpoint(
          'https://preview.example/app.json',
          production: true,
        ),
        throwsArgumentError,
      );
    },
  );

  test('cancellation aborts a response body that never completes', () async {
    final headersSent = Completer<void>();
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    server.listen((request) async {
      request.response.headers.contentType = ContentType.json;
      request.response.write('{');
      await request.response.flush();
      headersSent.complete();
    });
    final repository = createServiceStatusRepository(
      Uri.parse('http://127.0.0.1:${server.port}/app.json'),
    );
    try {
      final request = repository.start();
      final rejected = expectLater(request.result, throwsA(isA<Exception>()));
      await headersSent.future;
      request.cancel();
      await rejected;
    } finally {
      repository.close();
      await server.close(force: true);
    }
  });
}
