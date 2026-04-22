import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

DioException _ex({
  int? status,
  DioExceptionType type = DioExceptionType.badResponse,
  dynamic data,
}) {
  final req = RequestOptions(path: '/v1/whoami');
  return DioException(
    requestOptions: req,
    type: type,
    response: status == null
        ? null
        : Response(requestOptions: req, statusCode: status, data: data),
  );
}

void main() {
  group('ErrorMappingInterceptor', () {
    late _CapturingHandler handler;

    setUp(() => handler = _CapturingHandler());

    test('401 → ApiUnauthorized', () {
      const ErrorMappingInterceptor().onError(_ex(status: 401), handler);
      expect(handler.error, isA<ApiUnauthorized>());
    });

    test('400 with {"error": "handle_required"} → ApiBadRequest(code)', () {
      const ErrorMappingInterceptor().onError(
        _ex(status: 400, data: <String, dynamic>{'error': 'handle_required'}),
        handler,
      );
      expect(handler.error, isA<ApiBadRequest>());
      expect((handler.error as ApiBadRequest?)?.code, 'handle_required');
    });

    test('400 with no error field → ApiBadRequest(null)', () {
      const ErrorMappingInterceptor()
          .onError(_ex(status: 400, data: <String, dynamic>{}), handler);
      expect(handler.error, isA<ApiBadRequest>());
      expect((handler.error as ApiBadRequest?)?.code, isNull);
    });

    test('500 → ApiServerError', () {
      const ErrorMappingInterceptor().onError(_ex(status: 500), handler);
      expect(handler.error, isA<ApiServerError>());
    });

    test('timeout → ApiNetworkError', () {
      const ErrorMappingInterceptor()
          .onError(_ex(type: DioExceptionType.connectionTimeout), handler);
      expect(handler.error, isA<ApiNetworkError>());
    });

    test('connection error → ApiNetworkError', () {
      const ErrorMappingInterceptor()
          .onError(_ex(type: DioExceptionType.connectionError), handler);
      expect(handler.error, isA<ApiNetworkError>());
    });
  });
}

class _CapturingHandler extends ErrorInterceptorHandler {
  Object? error;

  @override
  void next(DioException err) {
    error = err.error;
  }
}
