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
    expect(recorder.requests.first.uri.queryParameters['did'], 'did:plc:alice');
    expect(
      recorder.requests.first.uri.queryParameters['name'],
      matches(RegExp(r'^craftsky-[0-9a-f]{32}\.mp4$')),
    );
    expect(
      recorder.requests.first.headers['authorization'],
      'Bearer service-secret',
    );
    expect(recorder.uploadBytes, hasLength(8));
    expect(recorder.requests.first.headers[Headers.contentLengthHeader], 8);
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

  test('polls when already_exists upload has no immediate blob', () async {
    final client = VideoServiceClient.forTesting(
      uploadEndpoint: Uri.parse(
        'https://video.bsky.app/xrpc/app.bsky.video.uploadVideo',
      ),
      dio: Dio()..httpClientAdapter = _AlreadyExistsWithoutBlobAdapter(),
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

    expect(result.outcome.name, 'processing');
    expect(result.jobId, 'job-one');
  });

  test('accepts already_exists polling envelope with a blob', () async {
    final client = VideoServiceClient.forTesting(
      uploadEndpoint: Uri.parse(
        'https://video.bsky.app/xrpc/app.bsky.video.uploadVideo',
      ),
      dio: Dio()..httpClientAdapter = _AlreadyExistsStatusAdapter(),
    );

    final result = await client.getJobStatus('job-one');

    expect(result.outcome.name, 'completed');
    expect(result.blob?.cid, 'bafy');
  });

  test('deduplication bypass appends a fresh valid MP4 free box', () async {
    final recorder = _UploadNameAdapter();
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
    await client.upload(
      source: source,
      ownerDid: 'did:plc:alice',
      authorizationHeader: 'Bearer service-secret',
      bypassDeduplication: true,
    );

    expect(recorder.names, hasLength(2));
    expect(recorder.names.first, isNot(recorder.names.last));
    expect(
      recorder.names,
      everyElement(matches(RegExp(r'^craftsky-[0-9a-f]{32}\.mp4$'))),
    );
    expect(recorder.bodies.first, hasLength(8));
    expect(recorder.bodies.last, hasLength(24));
    expect(
      recorder.bodies.last.sublist(8, 16),
      const [0, 0, 0, 16, 102, 114, 101, 101],
    );
  });
}

final class _RecordingAdapter implements HttpClientAdapter {
  final requests = <RequestOptions>[];
  final uploadBytes = <int>[];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    final upload = options.method == 'POST';
    if (upload) {
      await requestStream!.forEach(uploadBytes.addAll);
    } else {
      await requestStream?.drain<void>();
    }
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
      r'{"error":"already_exists","message":"Video already processed","jobStatus":{"jobId":"job-one","state":"JOB_STATE_FAILED","blob":{"$type":"blob","ref":{"$link":"bafy"},"mimeType":"video/mp4","size":8}}}',
      409,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

final class _AlreadyExistsWithoutBlobAdapter implements HttpClientAdapter {
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    await requestStream?.drain<void>();
    return ResponseBody.fromString(
      '{"error":"already_exists","jobStatus":'
      '{"jobId":"job-one","state":"JOB_STATE_FAILED"}}',
      409,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

final class _AlreadyExistsStatusAdapter implements HttpClientAdapter {
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    return ResponseBody.fromString(
      r'{"error":"already_exists","message":"Video already processed","jobStatus":{"jobId":"job-one","state":"JOB_STATE_FAILED","blob":{"$type":"blob","ref":{"$link":"bafy"},"mimeType":"video/mp4","size":8}}}',
      409,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

final class _UploadNameAdapter implements HttpClientAdapter {
  final names = <String>[];
  final bodies = <List<int>>[];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    final body = <int>[];
    await requestStream!.forEach(body.addAll);
    names.add(options.uri.queryParameters['name']!);
    bodies.add(body);
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
