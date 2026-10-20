import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('IT-018 unmapped private transport errors retain '
      'typed cause and stack, excluding prose', () async {
    final cause = StateError('private-cause-canary');
    final original = StackTrace.fromString(
      'originalTransport (transport.dart:12)',
    );
    final failure = DioException(
      requestOptions: RequestOptions(
        path: '/v1/chat/conversations/private-id/messages',
        method: 'POST',
        data: {'text': 'private-body-canary'},
      ),
      error: cause,
      message: 'private-provider-canary',
      stackTrace: original,
    );
    Object? caught;
    StackTrace? stack;
    try {
      await unwrapApi<void>(() => Future<void>.error(failure, original));
    } on Object catch (error, trace) {
      caught = error;
      stack = trace;
    }
    expect(caught, isA<ApiServerError>());
    final error = caught! as ApiServerError;
    expect(error.message, 'server_error');
    expect(error.details.cause, same(cause));
    expect(error.details.stackTrace.toString(), original.toString());
    expect(stack.toString(), original.toString());
    expect(error.toString(), isNot(contains('private-')));
  });
}
