import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/feed/data/video_service_client.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('IT-012 cancellation closes a slow upload stream and future', () async {
    final adapter = _BlockingAdapter();
    final cancelToken = CancelToken();
    final client = _client(adapter);

    final upload = client.upload(
      source: VideoUploadSource(
        length: 8,
        openRead: () => Stream<Uint8List>.periodic(
          const Duration(days: 1),
          (_) => Uint8List(1),
        ),
      ),
      ownerDid: 'did:plc:alice',
      authorizationHeader: 'Bearer service-secret',
      cancelToken: cancelToken,
    );
    await adapter.started.future;
    cancelToken.cancel('interrupted');

    await expectLater(
      upload.timeout(const Duration(seconds: 1)),
      throwsA(isA<DioException>()),
    );
    await adapter.cancelObserved.future.timeout(const Duration(seconds: 1));
    expect(adapter.requestStreamCanceled, isTrue);
  });

  test(
    'IT-012 streams bounded chunks without buffering the full source',
    () async {
      final adapter = _DrainingAdapter();
      final client = _client(adapter);
      const chunkLength = 64 * 1024;
      const chunkCount = 100;

      await client.upload(
        source: VideoUploadSource(
          length: chunkLength * chunkCount,
          openRead: () async* {
            for (var i = 0; i < chunkCount; i++) {
              yield Uint8List(chunkLength);
            }
          },
        ),
        ownerDid: 'did:plc:alice',
        authorizationHeader: 'Bearer service-secret',
      );

      expect(adapter.receivedBytes, chunkLength * chunkCount);
      expect(adapter.largestChunk, chunkLength);
    },
  );

  for (final type in [
    DioExceptionType.connectionError,
    DioExceptionType.sendTimeout,
    DioExceptionType.receiveTimeout,
  ]) {
    test(
      'IT-012 ${type.name} finishes with stable unavailable failure',
      () async {
        final client = _client(_FailingAdapter(type));

        await expectLater(
          client.getJobStatus('job-one').timeout(const Duration(seconds: 1)),
          throwsA(
            isA<VideoTransportException>().having(
              (error) => error.kind,
              'kind',
              VideoTransportFailure.unavailable,
            ),
          ),
        );
      },
    );
  }
}

VideoServiceClient _client(HttpClientAdapter adapter) =>
    VideoServiceClient.forTesting(
      uploadEndpoint: Uri.parse(
        'https://video.bsky.app/xrpc/app.bsky.video.uploadVideo',
      ),
      dio: Dio()..httpClientAdapter = adapter,
    );

final class _BlockingAdapter implements HttpClientAdapter {
  final started = Completer<void>();
  final cancelObserved = Completer<void>();
  bool requestStreamCanceled = false;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) {
    final response = Completer<ResponseBody>();
    final subscription = requestStream?.listen((_) {});
    if (!started.isCompleted) started.complete();
    if (cancelFuture case final cancellation?) {
      unawaited(
        cancellation.then((_) {
          if (subscription != null) unawaited(subscription.cancel());
          requestStreamCanceled = true;
          if (!cancelObserved.isCompleted) cancelObserved.complete();
          if (!response.isCompleted) {
            response.completeError(
              DioException.requestCancelled(
                requestOptions: options,
                reason: 'interrupted',
              ),
            );
          }
        }),
      );
    }
    return response.future;
  }

  @override
  void close({bool force = false}) {}
}

final class _DrainingAdapter implements HttpClientAdapter {
  int receivedBytes = 0;
  int largestChunk = 0;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    await for (final chunk in requestStream!) {
      receivedBytes += chunk.length;
      if (chunk.length > largestChunk) largestChunk = chunk.length;
    }
    return ResponseBody.fromString(
      '{"jobId":"job-one","state":"JOB_STATE_PROCESSING"}',
      200,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

final class _FailingAdapter implements HttpClientAdapter {
  _FailingAdapter(this.type);

  final DioExceptionType type;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) => Future.error(DioException(requestOptions: options, type: type));

  @override
  void close({bool force = false}) {}
}
