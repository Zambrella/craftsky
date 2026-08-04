import 'dart:io';
import 'dart:typed_data';

import 'package:craftsky_app/drafts/data/draft_file_store.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:path/path.dart' as p;

void main() {
  test('flushes files and atomically switches an existing manifest', () async {
    final root = await Directory.systemTemp.createTemp('draft-file-store-');
    addTearDown(() => root.delete(recursive: true));
    final store = IoDraftFileStore();
    final bundle = p.join(root.path, 'bundle');
    final target = p.join(bundle, 'manifest.json');
    final pending = p.join(bundle, '.pending-manifest.json');

    await store.ensureDirectory(bundle);
    await store.writeBytesFlushed(target, Uint8List.fromList([1]));
    await store.writeBytesFlushed(pending, Uint8List.fromList([2, 3]));
    await store.atomicReplace(sourcePath: pending, targetPath: target);

    expect(await store.readBytes(target), [2, 3]);
    expect(await store.fileExists(pending), isFalse);
  });

  test('maps filesystem failures without exposing paths', () async {
    final store = IoDraftFileStore();
    const canaryPath = '/private/draft-path-canary/missing.json';

    Object? failure;
    try {
      await store.readBytes(canaryPath);
    } on Object catch (error) {
      failure = error;
    }

    expect(failure, isA<DraftFileStoreException>());
    expect(failure.toString(), isNot(contains(canaryPath)));
  });

  test('detects symbolic links without following them', () async {
    final root = await Directory.systemTemp.createTemp('draft-file-link-');
    addTearDown(() => root.delete(recursive: true));
    final target = await Directory(p.join(root.path, 'target')).create();
    final link = Link(p.join(root.path, 'link'));
    await link.create(target.path);
    final store = IoDraftFileStore();

    expect(await store.isSymbolicLink(link.path), isTrue);
    expect(await store.isSymbolicLink(target.path), isFalse);
  });
}
