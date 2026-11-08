import 'dart:typed_data';

import 'package:dio/dio.dart';

/// Requests can reach the adapter after an asynchronous interceptor even when
/// their account client was closed during that wait. Retiring is cancellation
/// of that client lifetime; it must never dispatch through a closed adapter.
final class RetiringHttpClientAdapter implements HttpClientAdapter {
  new(this._delegate);

  final HttpClientAdapter _delegate;
  bool _retired = false;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    if (_retired) {
      throw DioException.requestCancelled(
        requestOptions: options,
        reason: null,
      );
    }
    try {
      return await _delegate.fetch(options, requestStream, cancelFuture);
    } on Object catch (error, stack) {
      if (!_retired) rethrow;
      Error.throwWithStackTrace(
        DioException(
          requestOptions: options,
          type: DioExceptionType.cancel,
          error: error,
          stackTrace: stack,
        ),
        stack,
      );
    }
  }

  @override
  void close({bool force = false}) {
    _retired = true;
    _delegate.close(force: force);
  }
}
