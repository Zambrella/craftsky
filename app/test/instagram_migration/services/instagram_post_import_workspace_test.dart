import 'dart:io';

import 'package:craftsky_app/instagram_migration/services/instagram_post_import_workspace.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:path/path.dart' as path;

void main() {
  test(
    'reclaims interrupted import files and keeps other cache files',
    () async {
      final sandbox = await Directory.systemTemp.createTemp(
        'instagram-workspace-test-',
      );
      addTearDown(() => sandbox.delete(recursive: true));
      final cache = await Directory(path.join(sandbox.path, 'cache')).create();
      final temp = await Directory(path.join(sandbox.path, 'temp')).create();
      InstagramPostImportWorkspace openWorkspace() =>
          InstagramPostImportWorkspace(
            cacheDirectory: () async => cache,
            legacyMediaDirectory: temp,
          );

      final interrupted = await openWorkspace().createReviewDirectory();
      final archive = await File(path.join(interrupted.path, 'archive.zip'))
          .create();
      final extracted = await interrupted.createTemp('media-');
      await File(path.join(extracted.path, 'video.mp4')).create();
      final legacyZip = await cache.createTemp('craftsky-instagram-posts-');
      final legacyMedia = await temp.createTemp('craftsky-instagram-');
      final unrelated = await temp.createTemp('craftsky-instagram-export-');

      final current = openWorkspace();
      final (first, second) = await (
        current.createReviewDirectory(),
        current.createReviewDirectory(),
      ).wait;
      expect(first.path, isNot(second.path));
      expect(first.existsSync(), isTrue);
      expect(second.existsSync(), isTrue);
      expect(archive.existsSync(), isFalse);
      expect(extracted.existsSync(), isFalse);
      expect(legacyZip.existsSync(), isFalse);
      expect(legacyMedia.existsSync(), isFalse);
      expect(unrelated.existsSync(), isTrue);
    },
  );
}
