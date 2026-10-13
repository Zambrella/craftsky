@TestOn('browser')
library;

import 'dart:js_interop';
import 'dart:typed_data';

import 'package:craftsky_app/shared/media/image_upload_http_client_web.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:web/web.dart' as web;

void main() {
  test(
    'browser fetch forbids redirects, cookies, referrers and caching',
    () async {
      final adapter = ImageUploadWebAdapter(
        fetch: (uri, init) async {
          expect(uri, 'https://pds.example/xrpc/com.atproto.repo.uploadBlob');
          expect(init.redirect, 'error');
          expect(init.credentials, 'omit');
          expect(init.cache, 'no-store');
          expect(init.referrerPolicy, 'no-referrer');
      final headers = init.headers as web.Headers;
          expect(headers.get('authorization'), 'Bearer capability-secret');
          expect(headers.has('content-length'), isFalse);
          expect((init.body! as JSUint8Array).toDart, [1, 2, 3]);
          return web.Response(
            '{"blob":{}}'.toJS,
            web.ResponseInit(status: 200),
          );
        },
      );
      final dio = Dio()..httpClientAdapter = adapter;
      addTearDown(() => dio.close(force: true));
      final response = await dio.postUri<ResponseBody>(
        Uri.parse('https://pds.example/xrpc/com.atproto.repo.uploadBlob'),
        data: Stream.value(Uint8List.fromList([1, 2, 3])),
        options: Options(
          responseType: ResponseType.stream,
          followRedirects: false,
          headers: {
            'authorization': 'Bearer capability-secret',
            Headers.contentLengthHeader: 3,
          },
        ),
      );
      expect(
        await response.data!.stream.expand((chunk) => chunk).toList(),
        isNotEmpty,
      );
    },
  );

  test('closing the browser adapter aborts an in-flight upload', () async {
    late web.AbortSignal signal;
    final adapter = ImageUploadWebAdapter(
      fetch: (_, init) async {
        signal = init.signal!;
        return web.Response('response'.toJS);
      },
    );
    final response = await adapter.fetch(
      RequestOptions(
        path: 'https://pds.example/xrpc/com.atproto.repo.uploadBlob',
        headers: {},
      ),
      Stream.value(Uint8List.fromList([1])),
      null,
    );
    expect(signal.aborted, isFalse);
    adapter.close(force: true);
    expect(signal.aborted, isTrue);
    // Release the response stream reader too.
    await response.stream.drain<void>();
  });
}
