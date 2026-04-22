import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('ApiException', () {
    test('ApiUnauthorized carries a stable message', () {
      expect(const ApiUnauthorized().message, 'unauthorized');
    });

    test('ApiBadRequest exposes the server error code when present', () {
      expect(const ApiBadRequest('handle_required').code, 'handle_required');
      expect(const ApiBadRequest('handle_required').message, 'handle_required');
    });

    test('ApiBadRequest falls back to "bad_request" when code is null', () {
      expect(const ApiBadRequest(null).message, 'bad_request');
      expect(const ApiBadRequest(null).code, isNull);
    });

    test('ApiServerError preserves the provided message', () {
      expect(const ApiServerError('boom').message, 'boom');
    });

    test('ApiNetworkError preserves the provided message', () {
      expect(const ApiNetworkError('offline').message, 'offline');
    });

    test('ApiException is exhaustive via switch', () {
      const values = <ApiException>[
        ApiUnauthorized(),
        ApiBadRequest('x'),
        ApiServerError('y'),
        ApiNetworkError('z'),
      ];
      for (final e in values) {
        final kind = switch (e) {
          ApiUnauthorized() => 'unauth',
          ApiBadRequest() => 'bad',
          ApiServerError() => 'srv',
          ApiNetworkError() => 'net',
        };
        expect(kind, isNotEmpty);
      }
    });
  });
}
