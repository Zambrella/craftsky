import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/feed/composer/composer_media_uploader.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'uploads ready images sequentially and returns current display order',
    () async {
      final started = <String>[];
      final uploader = ComposerMediaUploader(
        upload:
            ({required bytes, required mimeType, required cancelToken}) async {
              started.add(String.fromCharCodes(bytes));
              return _uploaded('cid-${started.length}', bytes.length);
            },
      );

      final result = await uploader.materializeImmediate(
        composerId: 'composer',
        images: [_image('one', 'a'), _image('two', 'b')],
      );

      expect(started, ['a', 'b']);
      expect(result!.map((image) => image.blob.link), ['cid-1', 'cid-2']);
    },
  );

  test(
    'retry reuses successful unchanged blobs and uploads only missing work',
    () async {
      var calls = 0;
      var failSecond = true;
      final uploader = ComposerMediaUploader(
        upload:
            ({required bytes, required mimeType, required cancelToken}) async {
              calls++;
              if (String.fromCharCodes(bytes) == 'b' && failSecond) {
                throw StateError('safe fake failure');
              }
              return _uploaded(
                'cid-${String.fromCharCodes(bytes)}',
                bytes.length,
              );
            },
      );
      final images = [_image('one', 'a'), _image('two', 'b')];

      await expectLater(
        uploader.materializeImmediate(composerId: 'composer', images: images),
        throwsStateError,
      );
      failSecond = false;
      final result = await uploader.materializeImmediate(
        composerId: 'composer',
        images: images.reversed.toList(),
      );

      expect(calls, 3);
      expect(result!.map((image) => image.blob.link), ['cid-b', 'cid-a']);
    },
  );

  test('gives every actual upload an independent timeout token', () async {
    final tokens = <CancelToken>[];
    final uploader = ComposerMediaUploader(
      transferBudget: const Duration(milliseconds: 5),
      upload: ({required bytes, required mimeType, required cancelToken}) {
        tokens.add(cancelToken);
        if (String.fromCharCodes(bytes) == 'a') {
          return Completer<UploadedImageBlob>().future;
        }
        return Future.value(_uploaded('cid-b', bytes.length));
      },
    );

    await expectLater(
      uploader.materializeImmediate(
        composerId: 'composer',
        images: [_image('one', 'a'), _image('two', 'b')],
      ),
      throwsA(isA<TimeoutException>()),
    );
    expect(tokens, hasLength(1));
    expect(tokens.single.isCancelled, isTrue);
  });
}

ComposerImageDraft _image(String id, String content) => ComposerImageDraft(
  id: id,
  fileName: '$id.jpg',
  mimeType: 'image/jpeg',
  altText: id,
  phase: ImageReady(
    bytes: Uint8List.fromList(content.codeUnits),
    mimeType: 'image/jpeg',
    width: 10,
    height: 20,
    sha256: 'digest-$content',
  ),
);

UploadedImageBlob _uploaded(String cid, int size) => UploadedImageBlob(
  cid: cid,
  mime: 'image/jpeg',
  size: size,
  blob: UploadedBlob(
    type: 'blob',
    ref: UploadedBlobRef(link: cid),
    mimeType: 'image/jpeg',
    size: size,
  ),
);
