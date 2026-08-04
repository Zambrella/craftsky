import 'dart:io';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/data/draft_file_store.dart';
import 'package:craftsky_app/drafts/data/file_local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('failed manifest switch preserves the last complete draft', () async {
    final root = await Directory.systemTemp.createTemp('draft-failure-');
    addTearDown(() => root.delete(recursive: true));
    final owner = AccountKey('did:plc:alice');
    final files = _FailingReplaceFileStore();
    var generated = 30;
    String nextId() =>
        '00000000-0000-4000-8000-${(generated++).toString().padLeft(12, '0')}';
    final repository = FileLocalPostDraftRepository(
      documentsRoot: root.path,
      owner: owner,
      fileStore: files,
      clock: () => DateTime.utc(2026, 8, 3),
      nextId: nextId,
    );
    const draftId = '00000000-0000-4000-8000-000000000001';
    final first = await repository.save(
      _request(
        owner: owner,
        draftId: draftId,
        text: 'Last complete value',
        bytes: [1],
      ),
    );
    files.failReplace = true;

    await expectLater(
      repository.save(
        _request(
          owner: owner,
          draftId: draftId,
          text: 'Must not become visible',
          bytes: [2],
          expectedRevision: first.revision,
        ),
      ),
      throwsA(
        isA<DraftRepositoryException>().having(
          (error) => error.reason,
          'reason',
          DraftRepositoryFailureReason.storageUnavailable,
        ),
      ),
    );

    final restarted = FileLocalPostDraftRepository(
      documentsRoot: root.path,
      owner: owner,
      fileStore: IoDraftFileStore(),
      clock: () => DateTime.utc(2026, 8, 3),
      nextId: nextId,
    );
    final restored = await restarted.get(draftId);
    expect(
      (restored.content as StandardDraftContent).text,
      'Last complete value',
    );
    expect(
      await restarted.readMedia(draftId, restored.media.single.mediaId),
      [1],
    );
  });
}

DraftWriteRequest _request({
  required AccountKey owner,
  required String draftId,
  required String text,
  required List<int> bytes,
  int? expectedRevision,
}) => DraftWriteRequest(
  id: draftId,
  owner: owner,
  kind: LocalPostDraftKind.standard,
  expectedRevision: expectedRevision,
  content: StandardDraftContent(text: text, languages: const ['en']),
  schedule: const DraftScheduleIntent.now(),
  orderedMedia: [
    PreparedDraftMedia(
      mediaId: '00000000-0000-4000-8000-000000000002',
      displayFileName: 'image.png',
      mimeType: 'image/png',
      bytes: Uint8List.fromList(bytes),
      width: 1,
      height: 1,
      altText: '',
    ),
  ],
);

final class _FailingReplaceFileStore implements DraftFileStore {
  final _delegate = IoDraftFileStore();
  bool failReplace = false;

  @override
  Future<void> atomicReplace({
    required String sourcePath,
    required String targetPath,
  }) {
    if (failReplace) {
      throw const DraftFileStoreException(
        DraftFileStoreFailureReason.ioFailure,
      );
    }
    return _delegate.atomicReplace(
      sourcePath: sourcePath,
      targetPath: targetPath,
    );
  }

  @override
  Future<void> deleteDirectory(String path) => _delegate.deleteDirectory(path);

  @override
  Future<void> deleteFile(String path) => _delegate.deleteFile(path);

  @override
  Future<bool> directoryExists(String path) => _delegate.directoryExists(path);

  @override
  Future<void> ensureDirectory(String path) => _delegate.ensureDirectory(path);

  @override
  Future<bool> fileExists(String path) => _delegate.fileExists(path);

  @override
  Future<List<String>> listChildDirectories(String path) =>
      _delegate.listChildDirectories(path);

  @override
  Future<List<String>> listChildFiles(String path) =>
      _delegate.listChildFiles(path);

  @override
  Future<void> moveDirectory({
    required String sourcePath,
    required String targetPath,
  }) => _delegate.moveDirectory(
    sourcePath: sourcePath,
    targetPath: targetPath,
  );

  @override
  Future<Uint8List> readBytes(String path) => _delegate.readBytes(path);

  @override
  Future<bool> isSymbolicLink(String path) => _delegate.isSymbolicLink(path);

  @override
  Future<void> writeBytesFlushed(String path, Uint8List bytes) =>
      _delegate.writeBytesFlushed(path, bytes);
}
