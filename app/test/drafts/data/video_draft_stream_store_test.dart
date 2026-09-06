import 'dart:io';

import 'package:craftsky_app/drafts/data/video_draft_stream_store.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-015 streams and hashes a source without collecting it', () async {
    final directory = await Directory.systemTemp.createTemp('video-draft-');
    addTearDown(() => directory.delete(recursive: true));
    final target = File('${directory.path}/source.mp4');
    var producedChunks = 0;

    final result = await writeVideoDraftStream(
      source: Stream.fromIterable(List.generate(4, (index) => [index, index])),
      targetPath: target.path,
      maximumBytes: 8,
      onChunkWritten: (_) => producedChunks++,
    );

    expect(result.byteLength, 8);
    expect(
      result.sha256,
      'bd4dc1ab59e0dee8c965d817a17edf87a55a5e25dfc98e1519f54d1e5a52f9e3',
    );
    expect(producedChunks, 4);
    expect(await target.readAsBytes(), [0, 0, 1, 1, 2, 2, 3, 3]);
  });

  test(
    'UT-015 removes an incomplete source when its limit is exceeded',
    () async {
      final directory = await Directory.systemTemp.createTemp('video-draft-');
      addTearDown(() => directory.delete(recursive: true));
      final target = File('${directory.path}/source.mp4');

      await expectLater(
        writeVideoDraftStream(
          source: Stream.fromIterable(const [
            <int>[1, 2],
            <int>[3, 4],
          ]),
          targetPath: target.path,
          maximumBytes: 3,
        ),
        throwsA(isA<VideoDraftStreamException>()),
      );
      // Async access mirrors the production cleanup boundary.
      // ignore: avoid_slow_async_io
      expect(await target.exists(), isFalse);
    },
  );
}
