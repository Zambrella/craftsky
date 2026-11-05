import 'dart:async';
import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';

import 'package:craftsky_app/feed/media/video_source_validator.dart';
import 'package:craftsky_app/feed/models/video_service_result.dart';
import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:dio/dio.dart';

const _uploadPath = '/xrpc/app.bsky.video.uploadVideo';
const _statusPath = '/xrpc/app.bsky.video.getJobStatus';
const _maxResponseBytes = 65536;
const _freeBoxLength = 16;

final class VideoUploadSource {
  const new({
    required this.length,
    required this.openRead,
    this.mimeType = 'video/mp4',
  });

  final int length;
  final String mimeType;
  final Stream<List<int>> Function() openRead;

  @override
  String toString() => 'VideoUploadSource(length: $length)';
}

final class VideoTransportException
    implements Exception, DiagnosticFailureCause {
  const new(
    this.kind, {
    this.diagnosticCause,
    this.diagnosticStack,
    this.httpStatus,
  });
  @override
  final Object? diagnosticCause;
  @override
  final StackTrace? diagnosticStack;
  final int? httpStatus;

  final VideoTransportFailure kind;

  @override
  String toString() => 'VideoTransportException(${kind.name})';
}

enum VideoTransportFailure {
  invalidDestination,
  invalidSource,
  responseTooLarge,
  unavailable,
}

final class VideoServiceConfiguration {
  new({required this.uploadEndpoint}) {
    _requireApprovedEndpoint(uploadEndpoint);
  }

  factory bluesky() => VideoServiceConfiguration(
    uploadEndpoint: Uri.parse(
      'https://video.bsky.app/xrpc/app.bsky.video.uploadVideo',
    ),
  );

  final Uri uploadEndpoint;
}

final class VideoServiceClient {
  factory({required Uri uploadEndpoint}) {
    _requireApprovedEndpoint(uploadEndpoint);
    return VideoServiceClient._(
      uploadEndpoint,
      Dio(
        BaseOptions(
          connectTimeout: const Duration(seconds: 15),
          sendTimeout: const Duration(minutes: 15),
          receiveTimeout: const Duration(seconds: 30),
          followRedirects: false,
          maxRedirects: 0,
          responseDecoder: _decodeBoundedResponse,
        ),
      ),
    );
  }

  factory fromConfiguration(VideoServiceConfiguration configuration) =>
      VideoServiceClient(uploadEndpoint: configuration.uploadEndpoint);

  factory forTesting({required Uri uploadEndpoint, required Dio dio}) {
    _requireApprovedEndpoint(uploadEndpoint);
    return VideoServiceClient._(uploadEndpoint, dio);
  }

  new _(this._uploadEndpoint, this._dio);

  final Uri _uploadEndpoint;
  final Dio _dio;

  Future<VideoServiceResult> upload({
    required VideoUploadSource source,
    required String ownerDid,
    required String authorizationHeader,
    bool bypassDeduplication = false,
    BunnyVideoUpload? bunny,
    ProgressCallback? onProgress,
    CancelToken? cancelToken,
  }) async {
    if (bunny != null) {
      return await _uploadBunny(source, bunny, onProgress, cancelToken);
    }
    _requireApprovedEndpoint(_uploadEndpoint);
    if (source.mimeType != 'video/mp4' ||
        source.length <= 0 ||
        source.length > maxVideoSourceBytes) {
      throw const VideoTransportException(VideoTransportFailure.invalidSource);
    }
    if (bypassDeduplication &&
        source.length > maxVideoSourceBytes - _freeBoxLength) {
      throw const VideoTransportException(VideoTransportFailure.invalidSource);
    }
    final suffix = bypassDeduplication ? _freshFreeBox() : Uint8List(0);
    final uploadLength = source.length + suffix.length;
    final uri = _uploadEndpoint.replace(
      queryParameters: {'did': ownerDid, 'name': _freshUploadName()},
    );
    try {
      final response = await _dio.postUri<Map<String, dynamic>>(
        uri,
        data: _boundedSource(source, suffix),
        options: Options(
          contentType: 'video/mp4',
          followRedirects: false,
          maxRedirects: 0,
          validateStatus: (status) =>
              status != null &&
              ((status >= 200 && status < 300) || status == 409),
          headers: {
            'authorization': authorizationHeader,
            Headers.contentLengthHeader: uploadLength,
          },
        ),
        cancelToken: cancelToken,
        onSendProgress: onProgress,
      );
      _requireUnredirected(response);
      final status = _jobStatus(response.data!);
      final result = VideoServiceResult.fromJson(status)
          .withRetryAfter(_parseRetryAfter(response));
      if (response.statusCode == 409) {
        if (status['error'] == 'already_exists' &&
            result.outcome != VideoServiceOutcome.completed &&
            result.jobId.isNotEmpty) {
          return VideoServiceResult(
            outcome: VideoServiceOutcome.processing,
            jobId: result.jobId,
            progress: result.progress,
            retryAfter: result.retryAfter,
          );
        }
        if (result.outcome == VideoServiceOutcome.processing) {
          throw const VideoTransportException(
            VideoTransportFailure.unavailable,
          );
        }
      }
      return result;
    } on VideoTransportException {
      rethrow;
    } on DioException catch (error, stackTrace) {
      if (CancelToken.isCancel(error)) rethrow;
      throw VideoTransportException(
        VideoTransportFailure.unavailable,
        diagnosticCause: error,
        diagnosticStack: stackTrace,
        httpStatus: error.response?.statusCode,
      );
    }
  }

  Future<VideoServiceResult> getJobStatus(
    String jobId, {
    CancelToken? cancelToken,
  }) async {
    final uri = _uploadEndpoint.replace(
      path: _statusPath,
      queryParameters: {'jobId': jobId},
    );
    try {
      final response = await _dio.getUri<Map<String, dynamic>>(
        uri,
        options: Options(
          followRedirects: false,
          maxRedirects: 0,
          validateStatus: (status) =>
              status != null &&
              ((status >= 200 && status < 300) || status == 409),
        ),
        cancelToken: cancelToken,
      );
      _requireUnredirected(response);
      final responseData = response.data!;
      if (responseData['jobStatus'] is! Map<String, dynamic>) {
        throw const FormatException('Invalid video job status response');
      }
      return VideoServiceResult.fromJson(_jobStatus(responseData))
          .withRetryAfter(_parseRetryAfter(response));
    } on DioException catch (error, stackTrace) {
      if (CancelToken.isCancel(error)) rethrow;
      throw VideoTransportException(
        VideoTransportFailure.unavailable,
        diagnosticCause: error,
        diagnosticStack: stackTrace,
        httpStatus: error.response?.statusCode,
      );
    }
  }

  Future<VideoServiceResult> _uploadBunny(
    VideoUploadSource source,
    BunnyVideoUpload capability,
    ProgressCallback? onProgress,
    CancelToken? cancelToken,
  ) async {
    if (!supportedVideoSourceMimeTypes.contains(source.mimeType) ||
        source.length <= 0 ||
        source.length > maxVideoSourceBytes ||
        !capability.expiresAt.isAfter(DateTime.now().toUtc())) {
      throw const VideoTransportException(VideoTransportFailure.invalidSource);
    }
    final headers = <String, Object>{
      'Tus-Resumable': '1.0.0',
      'AuthorizationSignature': capability.signature,
      'AuthorizationExpire':
          '${capability.expiresAt.millisecondsSinceEpoch ~/ 1000}',
      'LibraryId': capability.libraryId,
      'VideoId': capability.videoId,
    };
    final endpoint = Uri.parse('https://video.bunnycdn.com/tusupload');
    final fileType = base64Encode(utf8.encode(source.mimeType));
    final title = base64Encode(utf8.encode(capability.title));
    final collection = base64Encode(utf8.encode(capability.collectionId));
    // This isolated client carries only the upload capability, never CraftSky
    // sessions or the library key. Redirects are rejected on every request.
    try {
      final created = await _dio.postUri<Object?>(
        endpoint,
        options: Options(
          headers: {
            ...headers,
            'Upload-Length': '${source.length}',
            'Upload-Metadata':
                'filetype $fileType,title $title,collection $collection',
          },
          followRedirects: false,
          maxRedirects: 0,
          responseType: ResponseType.plain,
          validateStatus: (s) => s == 201,
        ),
        cancelToken: cancelToken,
      );
      final location = created.headers.value('location');
      if (location == null) {
        throw const VideoTransportException(
          VideoTransportFailure.invalidDestination,
        );
      }
      final target = endpoint.resolve(location);
      if (!_approvedBunnyLocation(target) ||
          created.isRedirect ||
          created.redirects.isNotEmpty) {
        throw const VideoTransportException(
          VideoTransportFailure.invalidDestination,
        );
      }
      var offset = 0;
      for (var attempt = 0; attempt < 4; attempt++) {
        try {
          final response = await _dio.patchUri<Object?>(
            target,
            data: _sourceFromOffset(source, offset),
            options: Options(
              contentType: 'application/offset+octet-stream',
              headers: {
                ...headers,
                'Upload-Offset': '$offset',
                Headers.contentLengthHeader: source.length - offset,
              },
              followRedirects: false,
              maxRedirects: 0,
              responseType: ResponseType.plain,
              validateStatus: (s) => s == 204,
            ),
            cancelToken: cancelToken,
            onSendProgress: (sent, total) =>
                onProgress?.call(offset + sent, source.length),
          );
          final next = int.tryParse(
            response.headers.value('upload-offset') ?? '',
          );
          if (response.isRedirect ||
              response.redirects.isNotEmpty ||
              next != source.length) {
            throw const VideoTransportException(
              VideoTransportFailure.invalidDestination,
            );
          }
          return VideoServiceResult(
            outcome: VideoServiceOutcome.processing,
            jobId: capability.jobId,
          );
        } on DioException catch (error) {
          if (CancelToken.isCancel(error) || attempt == 3) rethrow;
          final status = await _dio.headUri<Object?>(
            target,
            options: Options(
              headers: headers,
              followRedirects: false,
              maxRedirects: 0,
              responseType: ResponseType.plain,
              validateStatus: (s) => s == 200 || s == 204,
            ),
            cancelToken: cancelToken,
          );
          final next = int.tryParse(
            status.headers.value('upload-offset') ?? '',
          );
          if (status.isRedirect ||
              status.redirects.isNotEmpty ||
              next == null ||
              next < 0 ||
              next > source.length) {
            throw const VideoTransportException(
              VideoTransportFailure.invalidDestination,
            );
          }
          if (next == source.length) {
            return VideoServiceResult(
              outcome: VideoServiceOutcome.processing,
              jobId: capability.jobId,
            );
          }
          offset = next;
        }
      }
      throw const VideoTransportException(VideoTransportFailure.unavailable);
    } on DioException catch (error, stackTrace) {
      if (CancelToken.isCancel(error)) rethrow;
      // Diagnostic selection retains typed status/causes, never request headers.
      throw VideoTransportException(
        VideoTransportFailure.unavailable,
        diagnosticCause: error,
        diagnosticStack: stackTrace,
        httpStatus: error.response?.statusCode,
      );
    }
  }

  Stream<List<int>> _sourceFromOffset(
    VideoUploadSource source,
    int offset,
  ) async* {
    var seen = 0;
    await for (final chunk in source.openRead()) {
      final start = seen;
      seen += chunk.length;
      if (seen > source.length) {
        throw const VideoTransportException(
          VideoTransportFailure.invalidSource,
        );
      }
      if (seen <= offset) continue;
      yield chunk.sublist((offset - start).clamp(0, chunk.length));
    }
    if (seen != source.length) {
      throw const VideoTransportException(VideoTransportFailure.invalidSource);
    }
  }

  Stream<List<int>> _boundedSource(
    VideoUploadSource source,
    Uint8List suffix,
  ) async* {
    var sent = 0;
    await for (final chunk in source.openRead()) {
      sent += chunk.length;
      if (sent > source.length || sent > maxVideoSourceBytes) {
        throw const VideoTransportException(
          VideoTransportFailure.invalidSource,
        );
      }
      yield chunk;
    }
    if (sent != source.length) {
      throw const VideoTransportException(VideoTransportFailure.invalidSource);
    }
    if (suffix.isNotEmpty) yield suffix;
  }

  void _requireUnredirected(Response<Object?> response) {
    if (response.isRedirect ||
        response.redirects.isNotEmpty ||
        response.realUri.host != _uploadEndpoint.host) {
      throw const VideoTransportException(
        VideoTransportFailure.invalidDestination,
      );
    }
  }

  @override
  String toString() => 'VideoServiceClient(<isolated>)';
}

Duration? _parseRetryAfter(Response<Object?> response) {
  final seconds = int.tryParse(response.headers.value('retry-after') ?? '');
  return seconds != null && seconds >= 0 ? Duration(seconds: seconds) : null;
}

Map<String, Object?> _jobStatus(Map<String, Object?> response) {
  final jobStatus = response['jobStatus'];
  if (jobStatus is! Map<String, Object?>) return response;
  final error = response['error'];
  return error is String && jobStatus['error'] == null
      ? {...jobStatus, 'error': error}
      : jobStatus;
}

String _freshUploadName() {
  final random = Random.secure();
  final suffix = List.generate(
    16,
    (_) => random.nextInt(256).toRadixString(16).padLeft(2, '0'),
  ).join();
  return 'craftsky-$suffix.mp4';
}

Uint8List _freshFreeBox() {
  final box = Uint8List(_freeBoxLength)
    ..setRange(0, 8, const [0, 0, 0, _freeBoxLength, 102, 114, 101, 101]);
  final random = Random.secure();
  for (var index = 8; index < box.length; index++) {
    box[index] = random.nextInt(256);
  }
  return box;
}

void _requireApprovedEndpoint(Uri endpoint) {
  if (endpoint.scheme != 'https' ||
      endpoint.host.toLowerCase() != 'video.bsky.app' ||
      endpoint.port != 443 ||
      endpoint.path != _uploadPath ||
      endpoint.hasQuery ||
      endpoint.hasFragment) {
    throw ArgumentError('Unapproved video upload endpoint');
  }
}

String _decodeBoundedResponse(
  List<int> bytes,
  RequestOptions options,
  ResponseBody response,
) {
  if (bytes.length > _maxResponseBytes) {
    throw const VideoTransportException(VideoTransportFailure.responseTooLarge);
  }
  return utf8.decode(bytes);
}

bool _approvedBunnyLocation(Uri uri) =>
    uri.scheme == 'https' &&
    uri.host == 'video.bunnycdn.com' &&
    uri.port == 443 &&
    uri.userInfo.isEmpty &&
    uri.path.startsWith('/tusupload/') &&
    !uri.hasQuery &&
    !uri.hasFragment;
