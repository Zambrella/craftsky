import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/feed/composer/composer_media_uploader.dart';
import 'package:craftsky_app/feed/composer/link_preview_candidate.dart';
import 'package:craftsky_app/feed/composer/link_preview_controller.dart';
import 'package:craftsky_app/feed/models/link_preview.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:crypto/crypto.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'uploads ready images sequentially and returns current display order',
    () async {
      final started = <String>[];
      final uploader = ComposerMediaUploader();

      final result = await uploader.materializeImmediate(
        composerId: 'composer',
        images: [_image('one', 'a'), _image('two', 'b')],
        ownershipIsCurrent: () => true,
        upload:
            ({required bytes, required mimeType, required cancelToken}) async {
              started.add(String.fromCharCodes(bytes));
              return _uploaded('cid-${started.length}', bytes.length);
            },
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
      final uploader = ComposerMediaUploader();
      Future<UploadedImageBlob> upload({
        required List<int> bytes,
        required String mimeType,
        required CancelToken cancelToken,
      }) async {
        calls++;
        if (String.fromCharCodes(bytes) == 'b' && failSecond) {
          throw StateError('safe fake failure');
        }
        return _uploaded('cid-${String.fromCharCodes(bytes)}', bytes.length);
      }

      final images = [_image('one', 'a'), _image('two', 'b')];

      await expectLater(
        uploader.materializeImmediate(
          composerId: 'composer',
          images: images,
          ownershipIsCurrent: () => true,
          upload: upload,
        ),
        throwsStateError,
      );
      failSecond = false;
      final result = await uploader.materializeImmediate(
        composerId: 'composer',
        images: images.reversed.toList(),
        ownershipIsCurrent: () => true,
        upload: upload,
      );

      expect(calls, 3);
      expect(result!.map((image) => image.blob.link), ['cid-b', 'cid-a']);
    },
  );

  test('gives every actual upload an independent timeout token', () async {
    final tokens = <CancelToken>[];
    final uploader = ComposerMediaUploader(
      transferBudget: const Duration(milliseconds: 5),
    );

    await expectLater(
      uploader.materializeImmediate(
        composerId: 'composer',
        images: [_image('one', 'a'), _image('two', 'b')],
        ownershipIsCurrent: () => true,
        upload: ({required bytes, required mimeType, required cancelToken}) {
          tokens.add(cancelToken);
          if (String.fromCharCodes(bytes) == 'a') {
            return Completer<UploadedImageBlob>().future;
          }
          return Future.value(_uploaded('cid-b', bytes.length));
        },
      ),
      throwsA(isA<TimeoutException>()),
    );
    expect(tokens, hasLength(1));
    expect(tokens.single.isCancelled, isTrue);
  });

  test(
    'stops before the next upload when captured ownership changes',
    () async {
      final uploaded = <String>[];
      var ownershipIsCurrent = true;
      final uploader = ComposerMediaUploader();

      await expectLater(
        uploader.materializeImmediate(
          composerId: 'composer',
          images: [_image('one', 'a'), _image('two', 'b')],
          ownershipIsCurrent: () => ownershipIsCurrent,
          upload:
              ({
                required bytes,
                required mimeType,
                required cancelToken,
              }) async {
                uploaded.add(String.fromCharCodes(bytes));
                ownershipIsCurrent = false;
                return _uploaded('cid-a', bytes.length);
              },
        ),
        throwsStateError,
      );

      expect(uploaded, ['a']);
    },
  );

  test('rejects bytes mutated after local preparation', () async {
    var calls = 0;
    final bytes = Uint8List.fromList('prepared'.codeUnits);
    final image = ComposerImageDraft(
      id: 'one',
      fileName: 'one.jpg',
      mimeType: 'image/jpeg',
      altText: 'one',
      phase: ImageReady(
        bytes: bytes,
        mimeType: 'image/jpeg',
        width: 10,
        height: 20,
        sha256: sha256.convert(bytes).toString(),
      ),
    );
    final uploader = ComposerMediaUploader();
    Future<UploadedImageBlob> upload({
      required List<int> bytes,
      required String mimeType,
      required CancelToken cancelToken,
    }) async {
      calls += 1;
      return _uploaded('cid-one', bytes.length);
    }

    await uploader.materializeImmediate(
      composerId: 'composer',
      images: [image],
      ownershipIsCurrent: () => true,
      upload: upload,
    );
    bytes[0] ^= 0xff;

    await expectLater(
      uploader.materializeImmediate(
        composerId: 'composer',
        images: [image],
        ownershipIsCurrent: () => true,
        upload: upload,
      ),
      throwsStateError,
    );
    expect(calls, 1);
  });

  test(
    'IT-013 materializes the synchronous selection and uploads its thumbnail',
    () async {
      final uploader = ComposerMediaUploader();
      final selection = SelectedLinkPreview(
        candidate: LinkPreviewCandidate.parse('https://example.com/a#source'),
        preview: LinkPreview(
          url: Uri.parse('https://final.example/a#metadata'),
          title: 'Pattern',
          description: 'Description',
          thumbnail: LinkPreviewThumbnail(
            bytes: Uint8List.fromList([1, 2, 3]),
            mimeType: 'image/png',
            width: 20,
            height: 10,
          ),
        ),
      );
      var calls = 0;

      final external = await uploader.materializeImmediateExternal(
        composerId: 'composer',
        selection: selection,
        ownershipIsCurrent: () => true,
        upload:
            ({required bytes, required mimeType, required cancelToken}) async {
              calls++;
              expect(bytes, [1, 2, 3]);
              expect(mimeType, 'image/png');
              return _uploaded('cid-thumb', bytes.length);
            },
      );

      expect(calls, 1);
      expect(external.uri, 'https://final.example/a#metadata');
      expect(external.title, 'Pattern');
      expect(external.description, 'Description');
      expect(external.thumb?.link, 'cid-thumb');
    },
  );

  test('IT-013 metadata-only selection performs no upload', () async {
    final uploader = ComposerMediaUploader();
    final selection = SelectedLinkPreview(
      candidate: LinkPreviewCandidate.parse('https://example.com/a'),
      preview: LinkPreview(
        url: Uri.parse('https://final.example/a'),
        title: 'Pattern',
        description: 'Description',
      ),
    );

    final external = await uploader.materializeImmediateExternal(
      composerId: 'composer',
      selection: selection,
      ownershipIsCurrent: () => true,
      upload:
          ({required bytes, required mimeType, required cancelToken}) async {
            fail('metadata-only previews must not upload');
          },
    );

    expect(external.uri, 'https://final.example/a');
    expect(external.thumb, isNull);
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
    sha256: sha256.convert(content.codeUnits).toString(),
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
