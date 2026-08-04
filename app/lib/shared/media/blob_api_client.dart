import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:dio/dio.dart';

/// Shared AppView blob upload endpoint.
class BlobApiClient {
  const BlobApiClient(this._dio);

  final Dio _dio;

  /// POST /v1/blobs/images — uploads prepared image bytes.
  Future<UploadedImageBlob> uploadImage({
    required List<int> bytes,
    required String mimeType,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
    CancelToken? cancelToken,
  }) => unwrapApi(() async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/v1/blobs/images',
      data: bytes,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
      // The composer owns the complete per-image transfer budget and cancels
      // through [cancelToken]. Do not let the shared 15-second response
      // timeout end a valid large upload before that budget expires.
      options: Options(
        contentType: mimeType,
        receiveTimeout: Duration.zero,
      ),
    );
    return UploadedImageBlob.fromMap(res.data!);
  });
}
