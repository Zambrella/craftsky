import 'dart:convert';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/feed/composer/composer_video_publication.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/data/video_service_client.dart';
import 'package:craftsky_app/feed/models/create_post_video.dart';
import 'package:craftsky_app/feed/models/video_service_result.dart';
import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
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
    expect(parts['filetype'], 'video/mp4');
    expect(capability().toString(), isNot(contains(signature)));
  });
  test(
    'MOV composer upload preserves QuickTime MIME and publishes an MP4 proof',
    () async {
      final bytes = Uint8List.fromList([
        0,
        0,
        0,
        20,
        ...'ftypqt  '.codeUnits,
        ...List<int>.filled(8, 0),
      ]);
      final adapter = _TusAdapter(length: bytes.length);
      final service = VideoServiceClient.forTesting(
        uploadEndpoint: Uri.parse(
          'https://video.bsky.app/xrpc/app.bsky.video.uploadVideo',
        ),
        dio: Dio()..httpClientAdapter = adapter,
      );
      final selection = LocalVideoSelection(
        displayName: 'private-camera-name.mov',
        mimeType: 'video/quicktime',
        byteLength: bytes.length,
        duration: const Duration(seconds: 1),
        width: 1080,
        height: 1920,
        headerBytes: bytes.sublist(0, 12),
        openRead: () => Stream.value(bytes),
      );
      CreatePostVideo? published;
      final coordinator = composerVideoPublication(
        owner: ActiveAccountLease(
          session: AccountSessionLease(
            account: AccountKey('did:plc:abcdefghijklmnopqrstuvwx'),
            sessionGeneration: 1,
          ),
          activationGeneration: 1,
        ),
        selection: selection,
        api: _BunnyApi(
          VideoUploadAuthorization.fromMap({
            'provider': 'bunny',
            'jobId': job,
            'videoId': video,
            'libraryId': '123',
            'token': signature,
            'expiresAt': '2030-01-01T00:00:00Z',
            'title': capability().title,
            'collectionId': capability().collectionId,
          }),
        ),
        service: service,
        publish: (proof, {required allowBlobRecovery}) async =>
            published = proof,
        onProgress: (_) {},
      );
      await coordinator.publish(
        altText: 'A short MOV',
        aspectRatio: (1080, 1920),
      );
      final metadata =
          adapter.requests.first.headers['Upload-Metadata'] as String;
      final fields = {
        for (final part in metadata.split(','))
          part.split(' ').first: utf8.decode(
            base64Decode(part.split(' ').last),
          ),
      };
      expect(fields['filetype'], 'video/quicktime');
      expect(fields['title'], capability().title);
      expect(fields.values.join(' '), isNot(contains('private-camera-name')));
      expect(adapter.lastBytes, bytes.sublist(3));
      expect(published?.blob.mimeType, 'video/mp4');
      expect(published?.alt, 'A short MOV');
    },
  );

  test('unsupported source MIME is rejected before contacting Bunny', () async {
    final adapter = _TusAdapter();
    final service = VideoServiceClient.forTesting(
      uploadEndpoint: Uri.parse(
        'https://video.bsky.app/xrpc/app.bsky.video.uploadVideo',
      ),
      dio: Dio()..httpClientAdapter = adapter,
    );
    await expectLater(
      service.upload(
        source: VideoUploadSource(
          length: 8,
          mimeType: 'video/webm',
          openRead: () => Stream.value(List<int>.filled(8, 0)),
        ),
        ownerDid: 'did:plc:alice',
        authorizationHeader: 'Bearer private',
        bunny: capability(),
      ),
      throwsA(isA<VideoTransportException>()),
    );
    expect(adapter.requests, isEmpty);
  });

  test('legacy MP4 upload does not mislabel a MOV source', () async {
    final adapter = _TusAdapter();
    final service = VideoServiceClient.forTesting(
      uploadEndpoint: Uri.parse(
        'https://video.bsky.app/xrpc/app.bsky.video.uploadVideo',
      ),
      dio: Dio()..httpClientAdapter = adapter,
    );
    await expectLater(
      service.upload(
        source: VideoUploadSource(
          length: 8,
          mimeType: 'video/quicktime',
          openRead: () => Stream.value(List<int>.filled(8, 0)),
        ),
        ownerDid: 'did:plc:alice',
        authorizationHeader: 'Bearer private',
      ),
      throwsA(
        isA<VideoTransportException>().having(
          (e) => e.kind,
          'kind',
          VideoTransportFailure.invalidSource,
        ),
      ),
    );
    expect(adapter.requests, isEmpty);
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
  new({this.foreign = false, this.length = 8});
  final bool foreign;
  final int length;
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
        'upload-offset': ['$length'],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

class _BunnyApi extends PostApiClient {
  new(this.authorization) : super(Dio());
  final VideoUploadAuthorization authorization;
  @override
  Future<VideoUploadLimits> getVideoUploadLimits() async =>
      const VideoUploadLimits(canUpload: true);
  @override
  Future<VideoUploadAuthorization> authorizeVideoUpload() async =>
      authorization;
  @override
  Future<VideoServiceResult> getBunnyVideoStatus(
    String jobId, {
    CancelToken? cancelToken,
  }) async => VideoServiceResult.fromJson({
    'jobId': jobId,
    'state': 'JOB_STATE_COMPLETED',
    'blob': {
      r'$type': 'blob',
      'ref': {r'$link': 'bafyprocessed'},
      'mimeType': 'video/mp4',
      'size': 8,
    },
  });
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
