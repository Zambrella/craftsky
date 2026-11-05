import 'dart:convert';
import 'dart:typed_data';

import 'package:craftsky_app/feed/data/video_service_client.dart';
import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:craftsky_app/shared/observability/diagnostic_details.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  const job = 'aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa';
  const video = 'bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb';
  final signature = 'c' * 64;
  BunnyVideoUpload capability() => BunnyVideoUpload(
    jobId: job,
    videoId: video,
    libraryId: '123',
    title: '2026-11-05 12:34 UTC · aaaaaaaa',
    collectionId: 'dddddddd-dddd-4ddd-dddd-dddddddddddd',
    signature: signature,
    expiresAt: DateTime.utc(2030),
  );
  test('Bunny authorization retains and validates server upload metadata', () {
    final payload = <String, dynamic>{
      'provider': 'bunny',
      'jobId': job,
      'videoId': video,
      'libraryId': '123',
      'token': signature,
      'expiresAt': '2030-01-01T00:00:00Z',
      'title': capability().title,
      'collectionId': capability().collectionId,
    };
    final parsed = VideoUploadAuthorization.fromMap(payload).bunny!;
    expect(parsed.title, capability().title);
    expect(parsed.collectionId, capability().collectionId);
    for (final field in ['title', 'collectionId']) {
      final missing = Map<String, dynamic>.from(payload)..remove(field);
      expect(
        () => VideoUploadAuthorization.fromMap(missing),
        throwsFormatException,
      );
    }
  });
  test('Bunny upload resumes without forwarding account credentials', () async {
    final adapter = _TusAdapter();
    final client = VideoServiceClient.forTesting(
      uploadEndpoint: Uri.parse(
        'https://video.bsky.app/xrpc/app.bsky.video.uploadVideo',
      ),
      dio: Dio()..httpClientAdapter = adapter,
    );
    final result = await client.upload(
      source: VideoUploadSource(
        length: 8,
        openRead: () => Stream.value([0, 1, 2, 3, 4, 5, 6, 7]),
      ),
      ownerDid: 'did:plc:alice',
      authorizationHeader: 'Bearer must-not-leave-app',
      bunny: capability(),
    );
    expect(result.jobId, job);
    expect(result.outcome.name, 'processing');
    expect(adapter.methods, ['POST', 'PATCH', 'HEAD', 'PATCH']);
    expect(adapter.lastBytes, [3, 4, 5, 6, 7]);
    for (final request in adapter.requests) {
      expect(request.uri.host, 'video.bunnycdn.com');
      expect(request.followRedirects, isFalse);
      expect(request.headers.containsKey('authorization'), isFalse);
      expect(request.headers.containsKey('AccessKey'), isFalse);
      expect(request.headers['AuthorizationSignature'], signature);
    }
    final metadata =
        adapter.requests.first.headers['Upload-Metadata'] as String;
    final parts = {
      for (final part in metadata.split(','))
        part.split(' ').first: utf8.decode(base64Decode(part.split(' ').last)),
    };
    expect(parts['title'], capability().title);
    expect(parts['collection'], capability().collectionId);
    expect(capability().toString(), isNot(contains(signature)));
  });
  test(
    'Bunny diagnostics keep HTTP cause without capability or body',
    () async {
      final client = VideoServiceClient.forTesting(
        uploadEndpoint: Uri.parse(
          'https://video.bsky.app/xrpc/app.bsky.video.uploadVideo',
        ),
        dio: Dio()..httpClientAdapter = _FailureAdapter(),
      );
      try {
        await client.upload(
          source: VideoUploadSource(
            length: 1,
            openRead: () => Stream.value([1]),
          ),
          ownerDid: 'did:plc:alice',
          authorizationHeader: 'Bearer private-account',
          bunny: capability(),
        );
        fail('Expected transport failure');
      } on VideoTransportException catch (error) {
        final selected = selectedCause(error).toString();
        expect(selected, contains('DioException'));
        expect(selected, contains('HTTP 502'));
        expect(selected, isNot(contains(signature)));
        expect(selected, isNot(contains('private-provider-body')));
        expect(selected, isNot(contains('private-account')));
      }
    },
  );
  test('foreign TUS location is rejected before sending capability', () async {
    final adapter = _TusAdapter(foreign: true);
    final client = VideoServiceClient.forTesting(
      uploadEndpoint: Uri.parse(
        'https://video.bsky.app/xrpc/app.bsky.video.uploadVideo',
      ),
      dio: Dio()..httpClientAdapter = adapter,
    );
    await expectLater(
      client.upload(
        source: VideoUploadSource(length: 1, openRead: () => Stream.value([1])),
        ownerDid: 'did:plc:alice',
        authorizationHeader: 'Bearer private',
        bunny: capability(),
      ),
      throwsA(isA<VideoTransportException>()),
    );
    expect(adapter.requests, hasLength(1));
  });
}

class _TusAdapter implements HttpClientAdapter {
  new({this.foreign = false});
  final bool foreign;
  final List<RequestOptions> requests = [];
  final List<String> methods = [];
  List<int> lastBytes = [];
  int patches = 0;
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    methods.add(options.method);
    if (options.method == 'POST') {
      return ResponseBody.fromString(
        '',
        201,
        headers: {
          'location': [
            if (foreign)
              'https://untrusted.example/upload'
            else
              '/tusupload/test-upload',
          ],
        },
      );
    }
    if (options.method == 'HEAD') {
      return ResponseBody.fromString(
        '',
        200,
        headers: {
          'upload-offset': ['3'],
        },
      );
    }
    patches++;
    lastBytes = [];
    if (requestStream != null) {
      await requestStream.forEach(lastBytes.addAll);
    }
    if (patches == 1) {
      throw DioException(
        requestOptions: options,
        type: DioExceptionType.connectionError,
      );
    }
    return ResponseBody.fromString(
      '',
      204,
      headers: {
        'upload-offset': ['8'],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

class _FailureAdapter implements HttpClientAdapter {
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async => ResponseBody.fromString('private-provider-body', 502);
  @override
  void close({bool force = false}) {}
}
