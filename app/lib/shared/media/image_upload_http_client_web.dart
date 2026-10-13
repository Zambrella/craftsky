import 'dart:async';
import 'dart:js_interop';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:web/web.dart' as web;

const imageUploadSupportsSendProgress = false;

Dio newImageUploadHttpClient() =>
    Dio()..httpClientAdapter = ImageUploadWebAdapter();

typedef ImageUploadFetch =
    Future<web.Response> Function(String uri, web.RequestInit init);

/// XHR follows redirects automatically. Fetch's redirect:error prevents even
/// same-origin redirects from forwarding the upload capability to another path.
class ImageUploadWebAdapter implements HttpClientAdapter {
  new({ImageUploadFetch? fetch})
    : _fetch =
          fetch ?? ((uri, init) => web.window.fetch(uri.toJS, init).toDart);
  final ImageUploadFetch _fetch;
  final _active = <web.AbortController, Timer>{};
  bool _closed = false;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    if (_closed) {
      throw DioException(
        requestOptions: options,
        type: DioExceptionType.cancel,
      );
    }
    final abort = web.AbortController();
    unawaited(cancelFuture?.then((_) => abort.abort()));
    // Bound both an unresponsive upload and a stalled response body.
    // JS interop methods cannot be torn off by the web compiler.
    // ignore: unnecessary_lambdas
    final timeout = Timer(const Duration(minutes: 2), () => abort.abort());
    _active[abort] = timeout;
    try {
      final bytes = BytesBuilder(copy: false);
      await for (final chunk in requestStream!) {
        if (bytes.length + chunk.length > 2000000) {
          throw const FormatException('Image upload exceeds limit');
        }
        bytes.add(chunk);
      }
      if (abort.signal.aborted) {
        throw DioException(
          requestOptions: options,
          type: DioExceptionType.cancel,
        );
      }
      final headers = web.Headers();
      for (final entry in options.headers.entries) {
        // Browsers compute Content-Length; setting it manually is forbidden.
        if (entry.key.toLowerCase() != Headers.contentLengthHeader) {
          headers.set(entry.key, entry.value.toString());
        }
      }
      final payload = bytes.takeBytes();
      final response = await _fetch(
        options.uri.toString(),
        web.RequestInit(
          method: 'POST',
          credentials: 'omit',
          redirect: 'error',
          cache: 'no-store',
          referrerPolicy: 'no-referrer',
          signal: abort.signal,
          headers: headers,
          body: payload.toJS,
        ),
      );
      if (response.redirected ||
          (response.url.isNotEmpty && response.url != options.uri.toString())) {
        throw const FormatException('Image upload redirected');
      }
      final responseHeaders = <String, List<String>>{};
      final contentType = response.headers.get('content-type');
      if (contentType != null) {
        responseHeaders[Headers.contentTypeHeader] = [contentType];
      }
      Stream<Uint8List> read() async* {
        web.ReadableStreamDefaultReader? reader;
        try {
          if (response.body == null) return;
          reader = web.ReadableStreamDefaultReader(response.body!);
          while (true) {
            final chunk = await reader.read().toDart;
            if (chunk.done) break;
            yield (chunk.value! as JSUint8Array).toDart;
          }
        } finally {
          timeout.cancel();
          abort.abort();
          reader?.releaseLock();
          _active.remove(abort);
        }
      }

      return ResponseBody(read(), response.status, headers: responseHeaders);
    } on Object catch (error, stack) {
      timeout.cancel();
      final canceled = abort.signal.aborted;
      abort.abort();
      _active.remove(abort);
      throw DioException(
        requestOptions: options,
        type: canceled
            ? DioExceptionType.cancel
            : DioExceptionType.connectionError,
        error: error,
        stackTrace: stack,
      );
    }
  }

  @override
  void close({bool force = false}) {
    _closed = true;
    for (final entry in _active.entries) {
      entry.value.cancel();
      entry.key.abort();
    }
    _active.clear();
  }
}
