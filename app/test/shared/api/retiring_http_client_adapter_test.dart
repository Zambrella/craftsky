import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:craftsky_app/shared/api/retiring_http_client_adapter.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

final class _Adapter implements HttpClientAdapter {
  final result = Completer<ResponseBody>();
  final started = Completer<void>();
  int calls = 0;
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? stream,
    Future<void>? cancel,
  ) {
    calls++;
    started.complete();
    return result.future;
  }

  @override
  void close({bool force = false}) {}
}

void main() {
  test(
    'request paused in an interceptor cannot dispatch after client retirement',
    () async {
      final delegate = _Adapter();
      final client = Dio()
        ..httpClientAdapter = RetiringHttpClientAdapter(delegate);
      final entered = Completer<void>();
      final resume = Completer<void>();
      client.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) async {
            entered.complete();
            await resume.future;
            handler.next(options);
          },
        ),
      );
      client.interceptors.add(const ErrorMappingInterceptor());
      final result = unwrapApi(
        () => client.get<void>('https://example.invalid'),
      );
      final assertion = expectLater(result, throwsA(isA<ApiCanceled>()));
      await entered.future;
      client.close(force: true);
      resume.complete();
      await assertion;
      expect(delegate.calls, 0);
    },
  );

  for (final retire in [false, true]) {
    test(
      'in-flight adapter error '
      '${retire ? 'cancels a retired lifetime' : 'retains an active failure'}',
      () async {
        final delegate = _Adapter();
        final client = Dio()
          ..httpClientAdapter = RetiringHttpClientAdapter(delegate);
        client.interceptors.add(const ErrorMappingInterceptor());
        final result = unwrapApi(
          () => client.get<void>('https://example.invalid'),
        );
        final assertion = expectLater(
          result,
          throwsA(retire ? isA<ApiCanceled>() : isA<ApiServerError>()),
        );
        await delegate.started.future;
        if (retire) client.close(force: true);
        delegate.result.completeError(
          StateError(
            "Can't establish connection after the adapter was closed.",
          ),
        );
        await assertion;
        client.close(force: true);
      },
    );
  }
}
