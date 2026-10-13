import 'dart:async';
import 'dart:convert';

import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:craftsky_app/shared/media/image_upload_http_client.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:crypto/crypto.dart';
import 'package:dio/dio.dart';
import 'package:multiformats/multiformats.dart' as multiformats;

const _uploadPath = '/xrpc/com.atproto.repo.uploadBlob';
const _maxResponseBytes = 65536;
const _imageMimeTypes = {'image/jpeg', 'image/png', 'image/webp'};

/// Requests a short-lived capability from AppView, then sends prepared bytes
/// to the exact PDS destination using an isolated client with no interceptors.
class BlobApiClient {
  new(
    this._dio, {
    Dio Function()? pdsClientFactory,
    DateTime Function()? now,
  }) : _pdsClientFactory = pdsClientFactory ?? newImageUploadHttpClient,
       _now = now ?? DateTime.now;

  final Dio _dio;
  final Dio Function() _pdsClientFactory;
  final DateTime Function() _now;
  final _transfers = <CancelToken>{};
  bool _disposed = false;
  bool _lifecycleActive = true;

  void setLifecycleActive({required bool active}) {
    _lifecycleActive = active;
    if (!active) cancelUploads();
  }

  void cancelUploads() {
    for (final token in _transfers.toList()) {
      token.cancel('image upload interrupted');
    }
  }

  void dispose() {
    _disposed = true;
    cancelUploads();
  }

  Future<UploadedImageBlob> uploadImage({
    required List<int> bytes,
    required String mimeType,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
    CancelToken? cancelToken,
  }) async {
    if (_disposed || !_lifecycleActive || cancelToken?.isCancelled == true) {
      throw const ApiCanceled();
    }
    if (bytes.isEmpty ||
        bytes.length > 2000000 ||
        !_imageMimeTypes.contains(mimeType)) {
      throw const ApiBadRequest('invalid_image_upload');
    }
    final transfer = CancelToken();
    _transfers.add(transfer);
    // The callback retains only the cancellation token, never the capability.
    unawaited(
      cancelToken?.whenCancel.then((_) {
        if (!transfer.isCancelled) transfer.cancel('image upload canceled');
      }),
    );
    final deadline = Timer(const Duration(minutes: 2), () {
      transfer.cancel('image transfer timed out');
    });
    Dio? pds;
    RequestOptions? requestOptions;
    try {
      final authorization = await unwrapApi(
        () => _dio.post<Map<String, dynamic>>(
          '/v1/blobs/images/authorization',
          cancelToken: transfer,
        ),
      );
      final data = authorization.data;
      final endpoint = data?['uploadEndpoint'];
      final token = data?['token'];
      final expiry = data?['expiresAt'];
      data?.remove('token');
      if (transfer.isCancelled || _disposed || !_lifecycleActive) {
        throw const ApiCanceled();
      }
      if (endpoint is! String ||
          token is! String ||
          token.trim().isEmpty ||
          expiry is! String) {
        throw const FormatException('Invalid image upload authorization');
      }
      final uri = Uri.tryParse(endpoint);
      final expiresAt = DateTime.tryParse(expiry);
      if (uri == null ||
          uri.scheme != 'https' ||
          uri.host.isEmpty ||
          uri.userInfo.isNotEmpty ||
          uri.path != _uploadPath ||
          uri.hasQuery ||
          uri.hasFragment ||
          expiresAt == null ||
          !expiresAt.isAfter(_now()) ||
          expiresAt.isAfter(_now().add(const Duration(minutes: 6)))) {
        throw const FormatException('Invalid image upload authorization');
      }
      pds = _pdsClientFactory();
      if (!imageUploadSupportsSendProgress) {
        onSendProgress?.call(0, bytes.length);
      }
      final response = await pds.postUri<ResponseBody>(
        uri,
        data: Stream.value(bytes),
        options: Options(
          contentType: mimeType,
          responseType: ResponseType.stream,
          followRedirects: false,
          maxRedirects: 0,
          receiveTimeout: Duration.zero,
          validateStatus: (status) =>
              status != null && status >= 200 && status < 300,
          headers: {
            'authorization': 'Bearer $token',
            Headers.contentLengthHeader: bytes.length,
          },
        ),
        cancelToken: transfer,
        onSendProgress: imageUploadSupportsSendProgress ? onSendProgress : null,
        onReceiveProgress: onReceiveProgress,
      );
      requestOptions = response.requestOptions;
      if (!imageUploadSupportsSendProgress) {
        onSendProgress?.call(bytes.length, bytes.length);
      }
      if (response.isRedirect ||
          response.redirects.isNotEmpty ||
          response.realUri != uri) {
        throw const ApiBadRequest('invalid_image_upload_destination');
      }
      final body = <int>[];
      await for (final chunk in response.data!.stream) {
        if (transfer.isCancelled) throw const ApiCanceled();
        if (body.length + chunk.length > _maxResponseBytes) {
          throw const FormatException('Image upload response exceeds limit');
        }
        body.addAll(chunk);
      }
      if (transfer.isCancelled || _disposed) throw const ApiCanceled();
      final decoded = jsonDecode(utf8.decode(body));
      if (decoded is! Map<String, dynamic> ||
          decoded['blob'] is! Map<String, dynamic>) {
        throw const FormatException('Invalid image upload response');
      }
      final blob = UploadedBlob.fromMap(
        decoded['blob'] as Map<String, dynamic>,
      );
      if (blob.type != 'blob' ||
          !_imageMimeTypes.contains(blob.mimeType) ||
          blob.size != bytes.length ||
          blob.ref.link !=
              multiformats.CID.fromList([
                1,
                0x55,
                0x12,
                0x20,
                ...sha256.convert(bytes).bytes,
              ]).toString()) {
        throw const FormatException('Invalid image upload blob');
      }
      return UploadedImageBlob(
        blob: blob,
        cid: blob.ref.link,
        mime: blob.mimeType,
        size: blob.size,
      );
    } on DioException catch (error, stack) {
      requestOptions = error.requestOptions;
      // Remove capabilities and private payloads before retaining the original
      // typed transport cause for diagnostics. PDS errors never sign out the
      // CraftSky session and provider response prose is not surfaced.
      error.requestOptions.headers.clear();
      error.requestOptions.data = null;
      error.response?.data = null;
      if (CancelToken.isCancel(error) || transfer.isCancelled) {
        throw const ApiCanceled();
      }
      final details = ApiFailureDetails(
        statusCode: error.response?.statusCode,
        method: 'POST',
        cause: error,
        stackTrace: stack,
      );
      if (error.type == DioExceptionType.badResponse) {
        throw ApiBadRequest('image_upload_failed', details: details);
      }
      throw ApiNetworkError('image_upload_unavailable', details: details);
    } finally {
      deadline.cancel();
      requestOptions?.headers.clear();
      requestOptions?.data = null;
      pds?.close(force: true);
      _transfers.remove(transfer);
    }
  }

  @override
  String toString() => 'BlobApiClient(<isolated>)';
}
