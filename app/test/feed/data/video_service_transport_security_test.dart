import 'dart:typed_data';

import 'package:craftsky_app/feed/data/video_service_client.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('IT-010 service JWT is confined to the exact upload request', () async {
    final recorder = _RecordingAdapter();
    final client = VideoServiceClient.forTesting(
      uploadEndpoint: Uri.parse(
        'https://video.bsky.app/xrpc/app.bsky.video.uploadVideo',
      ),
      dio: Dio()..httpClientAdapter = recorder,
    );
    final source = VideoUploadSource(
      length: 8,
      openRead: () => Stream.value(
        Uint8List.fromList(const [0, 0, 0, 4, 102, 116, 121, 112]),
      ),
    );

    await client.upload(
      source: source,
      ownerDid: 'did:plc:alice',
      authorizationHeader: 'Bearer service-secret',
    );
    final status = await client.getJobStatus('job-one');

    expect(status.outcome.name, 'completed');
    expect(status.blob?.cid, 'bafy');
    expect(recorder.requests, hasLength(2));
    expect(recorder.requests.first.uri.scheme, 'https');
    expect(
      recorder.requests.first.uri.path,
      '/xrpc/app.bsky.video.uploadVideo',
    );
    expect(recorder.requests.first.uri.queryParameters, {
      'did': 'did:plc:alice',
      'name': 'video.mp4',
    });
    expect(
      recorder.requests.first.headers['authorization'],
      'Bearer service-secret',
    );
    expect(
      recorder.requests.last.headers.containsKey('authorization'),
      isFalse,
    );
    expect(
      recorder.requests.every((request) => !request.followRedirects),
      isTrue,
    );
    expect(client.toString(), isNot(contains('service-secret')));
    expect(status.retryAfter, const Duration(seconds: 12));
  });

  test('IT-010 rejects any non-approved upload destination', () {
    for (final uri in [
      'http://video.bsky.app/xrpc/app.bsky.video.uploadVideo',
      'https://evil.example/xrpc/app.bsky.video.uploadVideo',
      'https://video.bsky.app/xrpc/other',
      'https://video.bsky.app:444/xrpc/app.bsky.video.uploadVideo',
    ]) {
      expect(
        () => VideoServiceClient(uploadEndpoint: Uri.parse(uri)),
        throwsArgumentError,
      );
    }
  });

  test('IT-010 accepts documented already_exists upload response', () async {
    final client = VideoServiceClient.forTesting(
      uploadEndpoint: Uri.parse(
        'https://video.bsky.app/xrpc/app.bsky.video.uploadVideo',
      ),
      dio: Dio()..httpClientAdapter = _AlreadyExistsAdapter(),
    );

    final result = await client.upload(
      source: VideoUploadSource(
        length: 8,
        openRead: () => Stream.value(
          Uint8List.fromList(const [0, 0, 0, 4, 102, 116, 121, 112]),
        ),
      ),
      ownerDid: 'did:plc:alice',
      authorizationHeader: 'Bearer service-secret',
    );

    expect(result.outcome.name, 'completed');
    expect(result.blob?.cid, 'bafy');
  });
}

final class _RecordingAdapter implements HttpClientAdapter {
  final requests = <RequestOptions>[];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    await requestStream?.drain<void>();
    final upload = options.method == 'POST';
    return ResponseBody.fromString(
      upload
          ? '{"jobId":"job-one","state":"JOB_STATE_PROCESSING"}'
          : '{"jobStatus":{"jobId":"job-one",'
                '"state":"JOB_STATE_COMPLETED",'
                r'"blob":{"$type":"blob","ref":{"$link":"bafy"},'
                '"mimeType":"video/mp4","size":8}}}',
      200,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
        if (!upload) 'retry-after': ['12'],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

final class _AlreadyExistsAdapter implements HttpClientAdapter {
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    await requestStream?.drain<void>();
    return ResponseBody.fromString(
      '{"jobId":"job-one","state":"JOB_STATE_FAILED",'
      '"error":"already_exists",'
      r'"blob":{"$type":"blob","ref":{"$link":"bafy"},'
      '"mimeType":"video/mp4","size":8}}',
      409,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}
