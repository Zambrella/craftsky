import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/media/blob_api_client.dart';
import 'package:crypto/crypto.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:multiformats/multiformats.dart' as multiformats;

const endpoint = 'https://pds.example/xrpc/com.atproto.repo.uploadBlob';
const bytes = [1, 2, 3, 4];
final now = DateTime.utc(2026, 10, 13, 12);
String cidFor(List<int> value) => multiformats.CID.fromList([
  1,
  0x55,
  0x12,
  0x20,
  ...sha256.convert(value).bytes,
]).toString();

Map<String, dynamic> authorization({
  String destination = endpoint,
  DateTime? expiry,
}) => {
  'token': 'upload-capability-secret',
  'uploadEndpoint': destination,
  'expiresAt': (expiry ?? now.add(const Duration(minutes: 5)))
      .toIso8601String(),
};
Map<String, dynamic> uploadBody({
  String? cid,
  String mime = 'image/jpeg',
  int size = 4,
}) => {
  'blob': {
    r'$type': 'blob',
    'ref': {r'$link': cid ?? cidFor(bytes)},
    'mimeType': mime,
    'size': size,
  },
};
ResponseBody jsonResponse(Object body, [int status = 200]) =>
    ResponseBody.fromString(
      jsonEncode(body),
      status,
      headers: {
        Headers.contentTypeHeader: ['application/json'],
      },
    );

void main() {
  ({BlobApiClient client, _Adapter appview, _Adapter pds}) setup({
    Map<String, dynamic>? auth,
    Future<ResponseBody> Function(
      RequestOptions,
      Stream<Uint8List>?,
      Future<void>?,
    )?
    upload,
  }) {
    final appview = _Adapter((options, _, _) async {
      expect(
        options.uri.toString(),
        'https://appview.example/v1/blobs/images/authorization',
      );
      expect(options.method, 'POST');
      return jsonResponse(auth ?? authorization());
    });
    final pds = _Adapter(
      upload ??
          (options, stream, _) async {
            expect(options.uri.toString(), endpoint);
            expect(options.followRedirects, isFalse);
            expect(options.maxRedirects, 0);
            expect(
              options.headers['authorization'],
              'Bearer upload-capability-secret',
            );
            expect(
              options.headers.containsKey('X-Craftsky-Device-Id'),
              isFalse,
            );
            expect(options.headers[Headers.contentLengthHeader], 4);
            expect(await stream!.expand((chunk) => chunk).toList(), bytes);
            return jsonResponse(uploadBody());
          },
    );
    final dio = Dio(
      BaseOptions(
        baseUrl: 'https://appview.example',
        headers: {
          'authorization': 'Bearer craftsky-session-secret',
          'X-Craftsky-Device-Id': 'device-secret',
        },
      ),
    )..httpClientAdapter = appview;
    final client = BlobApiClient(
      dio,
      now: () => now,
      pdsClientFactory: () => Dio()..httpClientAdapter = pds,
    );
    addTearDown(client.dispose);
    addTearDown(() => dio.close(force: true));
    return (client: client, appview: appview, pds: pds);
  }

  test('authorizes through AppView and uploads only to the PDS', () async {
    final fixture = setup();
    // PostApiClient delegates to the same shared upload implementation.
    final result = await PostApiClient(
      Dio(),
      blobClient: fixture.client,
    ).uploadImage(bytes: bytes, mimeType: 'image/jpeg');
    expect(result.cid, cidFor(bytes));
    expect(result.mime, 'image/jpeg');
    expect(result.size, 4);
    expect(fixture.appview.calls, 1);
    expect(fixture.pds.calls, 1);
    expect(fixture.pds.last!.headers, isEmpty);
    expect(fixture.pds.last!.data, isNull);
  });

  for (final destination in [
    'http://pds.example/xrpc/com.atproto.repo.uploadBlob',
    'https://user:secret@pds.example/xrpc/com.atproto.repo.uploadBlob',
    'https://pds.example/xrpc/com.atproto.repo.createRecord',
    '$endpoint?token=secret',
    '$endpoint#fragment',
  ]) {
    test('rejects unsafe authorization destination $destination', () async {
      final fixture = setup(auth: authorization(destination: destination));
      await expectLater(
        fixture.client.uploadImage(bytes: bytes, mimeType: 'image/jpeg'),
        throwsFormatException,
      );
      expect(fixture.pds.calls, 0);
    });
  }

  test('rejects expired and excessively long authorization', () async {
    for (final expiry in [
      now,
      now.subtract(const Duration(seconds: 1)),
      now.add(const Duration(minutes: 30)),
    ]) {
      final fixture = setup(auth: authorization(expiry: expiry));
      await expectLater(
        fixture.client.uploadImage(bytes: bytes, mimeType: 'image/jpeg'),
        throwsFormatException,
      );
      expect(fixture.pds.calls, 0);
    }
  });

  test(
    'rejects redirects without retaining the capability or signing out',
    () async {
      final fixture = setup(
        upload: (_, _, _) async => ResponseBody.fromString(
          '',
          307,
          headers: {
            'location': ['https://evil.example'],
          },
        ),
      );
      ApiException? failure;
      try {
        await fixture.client.uploadImage(bytes: bytes, mimeType: 'image/jpeg');
      } on ApiException catch (error) {
        failure = error;
      }
      expect(failure, isA<ApiBadRequest>());
      expect(fixture.pds.calls, 1);
      expect(fixture.pds.last!.headers, isEmpty);
      expect(fixture.pds.last!.data, isNull);
      expect(
        failure!.details.cause.toString(),
        isNot(contains('upload-capability-secret')),
      );
    },
  );

  test(
    'PDS 401 stays an upload failure, with protected transport cause',
    () async {
      final fixture = setup(
        upload: (_, _, _) async =>
            jsonResponse({'message': 'upload-capability-secret'}, 401),
      );
      try {
        await fixture.client.uploadImage(bytes: bytes, mimeType: 'image/jpeg');
        fail('upload succeeded');
      } on ApiBadRequest catch (error) {
        expect(error.details.statusCode, 401);
        final cause = error.details.cause! as DioException;
        expect(cause.requestOptions.headers, isEmpty);
        expect(cause.requestOptions.data, isNull);
        expect(cause.response?.data, isNull);
      }
    },
  );

  test(
    'rejects foreign CID, false size, and non-image blob metadata',
    () async {
      for (final body in [
        uploadBody(cid: cidFor([9])),
        uploadBody(size: 5),
        uploadBody(mime: 'video/mp4'),
      ]) {
        final fixture = setup(upload: (_, _, _) async => jsonResponse(body));
        await expectLater(
          fixture.client.uploadImage(bytes: bytes, mimeType: 'image/jpeg'),
          throwsFormatException,
        );
      }
    },
  );

  test('bounds PDS responses', () async {
    final fixture = setup(
      upload: (_, _, _) async => ResponseBody.fromString('x' * 65537, 200),
    );
    await expectLater(
      fixture.client.uploadImage(bytes: bytes, mimeType: 'image/jpeg'),
      throwsFormatException,
    );
  });

  test(
    'backgrounded clients cannot acquire capabilities until resumed',
    () async {
      final fixture = setup();
      fixture.client.setLifecycleActive(active: false);
      await expectLater(
        fixture.client.uploadImage(bytes: bytes, mimeType: 'image/jpeg'),
        throwsA(isA<ApiCanceled>()),
      );
      expect(fixture.appview.calls, 0);
      fixture.client.setLifecycleActive(active: true);
      final result = await fixture.client.uploadImage(
        bytes: bytes,
        mimeType: 'image/jpeg',
      );
      expect(result.cid, cidFor(bytes));
    },
  );

  test('invalid source is rejected before capability issuance', () async {
    final fixture = setup();
    for (final value in [<int>[], List<int>.filled(2000001, 0)]) {
      await expectLater(
        fixture.client.uploadImage(bytes: value, mimeType: 'image/jpeg'),
        throwsA(isA<ApiBadRequest>()),
      );
    }
    await expectLater(
      fixture.client.uploadImage(bytes: bytes, mimeType: 'video/mp4'),
      throwsA(isA<ApiBadRequest>()),
    );
    expect(fixture.appview.calls, 0);
  });

  test(
    'caller cancellation and client disposal abort active PDS transfers',
    () async {
      for (final dispose in [false, true]) {
        final started = Completer<void>();
        final fixture = setup(
          upload: (options, _, cancel) async {
            started.complete();
            await cancel;
            throw DioException(
              requestOptions: options,
              type: DioExceptionType.cancel,
            );
          },
        );
        final token = CancelToken();
        final future = fixture.client.uploadImage(
          bytes: bytes,
          mimeType: 'image/jpeg',
          cancelToken: token,
        );
        final expectation = expectLater(future, throwsA(isA<ApiCanceled>()));
        await started.future;
        if (dispose) {
          fixture.client.dispose();
        } else {
          token.cancel();
        }
        await expectation;
        expect(fixture.pds.last!.headers, isEmpty);
      }
    },
  );
}

class _Adapter implements HttpClientAdapter {
  new(this.respond);
  final Future<ResponseBody> Function(
    RequestOptions,
    Stream<Uint8List>?,
    Future<void>?,
  )
  respond;
  int calls = 0;
  RequestOptions? last;
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) {
    calls++;
    last = options;
    return respond(options, requestStream, cancelFuture);
  }

  @override
  void close({bool force = false}) {}
}
