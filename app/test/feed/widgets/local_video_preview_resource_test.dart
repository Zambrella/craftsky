import 'dart:io';
import 'dart:typed_data';

import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/widgets/local_video_preview_resource.dart';
import 'package:flutter_test/flutter_test.dart';

LocalVideoSelection selection({Uri? uri, int bytes = 3}) => LocalVideoSelection(
  displayName: 'private-name.mov',
  mimeType: 'video/quicktime',
  byteLength: bytes,
  duration: null,
  headerBytes: Uint8List(0),
  openRead: () => Stream.value([1, 2, 3]),
  localUri: uri,
);
void main() {
  test('restored sources are streamed to a disposable local file', () async {
    final resource = await createLocalVideoPreviewResource(selection());
    final file = File.fromUri(resource.uri);
    expect(await file.readAsBytes(), [1, 2, 3]);
    expect(file.path, isNot(contains('private-name')));
    await resource.dispose();
    await resource.dispose();
    expect(file.existsSync(), isFalse);
  });
  test(
    'a picked local source is reused and never deleted by preview cleanup',
    () async {
      final directory = await Directory.systemTemp.createTemp(
        'video-preview-test-',
      );
      final file = await File('${directory.path}/source.mov')
          .writeAsBytes([1, 2, 3]);
      try {
        final resource = await createLocalVideoPreviewResource(
          selection(uri: file.uri),
        );
        expect(resource.uri, file.uri);
        await resource.dispose();
        expect(file.existsSync(), isTrue);
      } finally {
        await directory.delete(recursive: true);
      }
    },
  );
  test('truncated private draft sources cannot be previewed', () async {
    await expectLater(
      createLocalVideoPreviewResource(selection(bytes: 5)),
      throwsFormatException,
    );
  });
}
