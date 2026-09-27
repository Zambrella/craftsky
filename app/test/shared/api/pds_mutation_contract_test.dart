import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('generates canonical UUID operation keys', () {
    final key = newPdsMutationOperationKey();
    expect(isCanonicalPdsMutationOperationKey(key), isTrue);
  });

  test('parses only the exact ambiguous response contract', () {
    final response = Response<Object?>(
      requestOptions: RequestOptions(path: '/mutation'),
      statusCode: 202,
      data: {'status': 'ambiguous'},
      headers: Headers.fromMap({
        'retry-after': ['3'],
        'location': ['/operations/ignored'],
      }),
    );

    expect(
      () => parsePdsMutationResponse<void>(response, accepted: (_) {}),
      throwsA(
        isA<PdsMutationAmbiguousException>().having(
          (error) => error.retryAfterSeconds,
          'retryAfterSeconds',
          3,
        ),
      ),
    );
  });

  test(
    'clamps valid integer retry guidance and rejects malformed ambiguity',
    () {
      for (final value in ['0', '9']) {
        final response = Response<Object?>(
          requestOptions: RequestOptions(path: '/mutation'),
          statusCode: 202,
          data: {'status': 'ambiguous'},
          headers: Headers.fromMap({
            'retry-after': [value],
          }),
        );
        try {
          parsePdsMutationResponse<void>(response, accepted: (_) {});
          fail('expected ambiguity');
        } on PdsMutationAmbiguousException catch (error) {
          expect(error.retryAfterSeconds, value == '0' ? 1 : 5);
        }
      }

      for (final response in <Response<Object?>>[
        Response<Object?>(
          requestOptions: RequestOptions(path: '/mutation'),
          statusCode: 202,
          data: {'status': 'ambiguous', 'extra': true},
          headers: Headers.fromMap({
            'retry-after': ['2'],
          }),
        ),
        Response<Object?>(
          requestOptions: RequestOptions(path: '/mutation'),
          statusCode: 202,
          data: {'status': 'ambiguous'},
          headers: Headers.fromMap({
            'retry-after': ['1.5'],
          }),
        ),
      ]) {
        expect(
          () => parsePdsMutationResponse<void>(response, accepted: (_) {}),
          throwsFormatException,
        );
      }
    },
  );

  test('requires accepted delete to be an empty 204', () {
    final accepted = Response<Object?>(
      requestOptions: RequestOptions(path: '/mutation'),
      statusCode: 204,
    );
    parsePdsMutationResponse<void>(
      accepted,
      accepted: (data) {
        expect(data, isNull);
      },
      requireEmptyNoContent: true,
    );

    final invalid = Response<Object?>(
      requestOptions: RequestOptions(path: '/mutation'),
      statusCode: 204,
      data: {'unexpected': true},
    );
    expect(
      () => parsePdsMutationResponse<void>(
        invalid,
        accepted: (_) {},
        requireEmptyNoContent: true,
      ),
      throwsFormatException,
    );
  });
}
