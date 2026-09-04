import 'dart:convert';

import 'package:craftsky_app/feed/media/video_source_validator.dart';
import 'package:craftsky_app/feed/models/video_service_result.dart';
import 'package:dio/dio.dart';

const _uploadPath = '/xrpc/app.bsky.video.uploadVideo';
const _statusPath = '/xrpc/app.bsky.video.getJobStatus';
const _uploadName = 'video.mp4';
const _maxResponseBytes = 65536;

final class VideoUploadSource {
  const VideoUploadSource({required this.length, required this.openRead});

  final int length;
  final Stream<List<int>> Function() openRead;

  @override
  String toString() => 'VideoUploadSource(length: $length)';
}

final class VideoTransportException implements Exception {
  const VideoTransportException(this.kind);

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
  VideoServiceConfiguration({required this.uploadEndpoint}) {
    _requireApprovedEndpoint(uploadEndpoint);
  }

  factory VideoServiceConfiguration.bluesky() => VideoServiceConfiguration(
    uploadEndpoint: Uri.parse(
      'https://video.bsky.app/xrpc/app.bsky.video.uploadVideo',
    ),
  );

  final Uri uploadEndpoint;
}

final class VideoServiceClient {
  factory VideoServiceClient({required Uri uploadEndpoint}) {
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

  factory VideoServiceClient.fromConfiguration(
    VideoServiceConfiguration configuration,
  ) => VideoServiceClient(uploadEndpoint: configuration.uploadEndpoint);

  factory VideoServiceClient.forTesting({
    required Uri uploadEndpoint,
    required Dio dio,
  }) {
    _requireApprovedEndpoint(uploadEndpoint);
    return VideoServiceClient._(uploadEndpoint, dio);
  }

  VideoServiceClient._(this._uploadEndpoint, this._dio);

  final Uri _uploadEndpoint;
  final Dio _dio;

  Future<VideoServiceResult> upload({
    required VideoUploadSource source,
    required String ownerDid,
    required String authorizationHeader,
    ProgressCallback? onProgress,
    CancelToken? cancelToken,
  }) async {
    _requireApprovedEndpoint(_uploadEndpoint);
    if (source.length <= 0 || source.length > maxVideoSourceBytes) {
      throw const VideoTransportException(VideoTransportFailure.invalidSource);
    }
    final uri = _uploadEndpoint.replace(
      queryParameters: {'did': ownerDid, 'name': _uploadName},
    );
    try {
      final response = await _dio.postUri<Map<String, dynamic>>(
        uri,
        data: _boundedSource(source),
        options: Options(
          contentType: 'video/mp4',
          followRedirects: false,
          maxRedirects: 0,
          validateStatus: (status) =>
              status != null &&
              ((status >= 200 && status < 300) || status == 409),
          headers: {
            'authorization': authorizationHeader,
            Headers.contentLengthHeader: source.length,
          },
        ),
        cancelToken: cancelToken,
        onSendProgress: onProgress,
      );
      _requireUnredirected(response);
      final result = VideoServiceResult.fromJson(
        response.data!,
      ).withRetryAfter(_parseRetryAfter(response));
      if (response.statusCode == 409 &&
          result.outcome != VideoServiceOutcome.completed) {
        throw const VideoTransportException(VideoTransportFailure.unavailable);
      }
      return result;
    } on VideoTransportException {
      rethrow;
    } on DioException catch (error) {
      if (CancelToken.isCancel(error)) rethrow;
      throw const VideoTransportException(VideoTransportFailure.unavailable);
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
        options: Options(followRedirects: false, maxRedirects: 0),
        cancelToken: cancelToken,
      );
      _requireUnredirected(response);
      final jobStatus = response.data!['jobStatus'];
      if (jobStatus is! Map<String, dynamic>) {
        throw const FormatException('Invalid video job status response');
      }
      return VideoServiceResult.fromJson(
        jobStatus,
      ).withRetryAfter(_parseRetryAfter(response));
    } on DioException catch (error) {
      if (CancelToken.isCancel(error)) rethrow;
      throw const VideoTransportException(VideoTransportFailure.unavailable);
    }
  }

  Stream<List<int>> _boundedSource(VideoUploadSource source) async* {
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
