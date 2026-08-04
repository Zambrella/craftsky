import 'dart:io';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/data/draft_file_store.dart';
import 'package:craftsky_app/drafts/data/draft_storage_paths.dart';
import 'package:craftsky_app/drafts/data/file_local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/draft_manifest_codec.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:path/path.dart' as p;

void main() {
  test(
    'saves prepared bytes and restores them after repository restart',
    () async {
      final root = await Directory.systemTemp.createTemp('local-drafts-');
      addTearDown(() => root.delete(recursive: true));
      final owner = AccountKey('did:plc:alice');
      final now = DateTime.utc(2026, 8, 3, 12);
      const draftId = '00000000-0000-4000-8000-000000000001';
      final bytes = Uint8List.fromList([1, 2, 3, 4]);
      var generated = 10;
      String nextId() =>
          '00000000-0000-4000-8000-'
          '${(generated++).toString().padLeft(12, '0')}';
      final repository = FileLocalPostDraftRepository(
        documentsRoot: root.path,
        owner: owner,
        fileStore: IoDraftFileStore(),
        clock: () => now,
        nextId: nextId,
      );

      final saved = await repository.save(
        DraftWriteRequest(
          id: draftId,
          owner: owner,
          kind: LocalPostDraftKind.standard,
          content: const StandardDraftContent(
            text: 'Offline work',
            languages: ['en'],
          ),
          schedule: const DraftScheduleIntent.now(),
          orderedMedia: [
            PreparedDraftMedia(
              mediaId: '00000000-0000-4000-8000-000000000002',
              displayFileName: 'swatch.png',
              mimeType: 'image/png',
              bytes: bytes,
              width: 2,
              height: 2,
              altText: 'Blue swatch',
            ),
          ],
        ),
      );

      expect(saved.revision, 1);
      expect(saved.createdAt, now);
      expect(saved.updatedAt, now);
      expect(saved.media, hasLength(1));

      final restarted = FileLocalPostDraftRepository(
        documentsRoot: root.path,
        owner: owner,
        fileStore: IoDraftFileStore(),
        clock: () => now,
        nextId: nextId,
      );
      final restored = await restarted.get(draftId);
      final restoredBytes = await restarted.readMedia(
        draftId,
        restored.media.single.mediaId,
      );

      expect((restored.content as StandardDraftContent).text, 'Offline work');
      expect(restoredBytes, bytes);
      expect(await restarted.list(), hasLength(1));
    },
  );

  test(
    'updates in place, reuses retained bytes, and deletes idempotently',
    () async {
      final root = await Directory.systemTemp.createTemp(
        'local-drafts-update-',
      );
      addTearDown(() => root.delete(recursive: true));
      final owner = AccountKey('did:plc:alice');
      var now = DateTime.utc(2026, 8, 3, 12);
      var generated = 20;
      String nextId() =>
          '00000000-0000-4000-8000-'
          '${(generated++).toString().padLeft(12, '0')}';
      final repository = FileLocalPostDraftRepository(
        documentsRoot: root.path,
        owner: owner,
        fileStore: IoDraftFileStore(),
        clock: () => now,
        nextId: nextId,
      );
      const draftId = '00000000-0000-4000-8000-000000000001';
      final first = await repository.save(
        DraftWriteRequest(
          id: draftId,
          owner: owner,
          kind: LocalPostDraftKind.standard,
          content: const StandardDraftContent(text: 'One', languages: ['en']),
          schedule: const DraftScheduleIntent.now(),
          orderedMedia: [
            for (var index = 0; index < 2; index++)
              PreparedDraftMedia(
                mediaId: '00000000-0000-4000-8000-00000000000${index + 2}',
                displayFileName: 'image-$index.png',
                mimeType: 'image/png',
                bytes: Uint8List.fromList([index + 1]),
                width: 1,
                height: 1,
                altText: '',
              ),
          ],
        ),
      );
      final retained = first.media.first;
      now = now.add(const Duration(hours: 1));

      final updated = await repository.save(
        DraftWriteRequest(
          id: draftId,
          owner: owner,
          kind: LocalPostDraftKind.standard,
          expectedRevision: first.revision,
          content: const StandardDraftContent(text: 'Two', languages: ['en']),
          schedule: const DraftScheduleIntent.now(),
          orderedMedia: [
            ExistingStoredMedia(
              mediaId: retained.mediaId,
              storageRevision: retained.storageRevision,
              expectedSha256: retained.sha256,
              displayFileName: retained.displayFileName,
              mimeType: retained.mimeType,
              byteLength: retained.byteLength,
              width: retained.width,
              height: retained.height,
              altText: 'Changed alt',
            ),
          ],
        ),
      );

      expect(updated.id, first.id);
      expect(updated.createdAt, first.createdAt);
      expect(updated.updatedAt, now);
      expect(updated.revision, 2);
      expect(updated.media.single.storageFileName, retained.storageFileName);
      expect(updated.media.single.altText, 'Changed alt');
      expect(await repository.readMedia(draftId, retained.mediaId), [1]);

      await repository.delete(draftId);
      await repository.delete(draftId);
      expect(await repository.list(), isEmpty);
    },
  );

  test(
    'rejects a draft directory symlink that escapes the account root',
    () async {
      final root = await Directory.systemTemp.createTemp('local-drafts-link-');
      addTearDown(() => root.delete(recursive: true));
      final owner = AccountKey('did:plc:alice');
      const draftId = '00000000-0000-4000-8000-000000000001';
      final paths = DraftStoragePaths(documentsRoot: root.path, owner: owner);
      final outside = await Directory(
        '${root.path}${Platform.pathSeparator}outside',
      ).create();
      final escapedDraft = LocalPostDraft(
        id: draftId,
        owner: owner,
        kind: LocalPostDraftKind.standard,
        createdAt: DateTime.utc(2026, 8, 3),
        updatedAt: DateTime.utc(2026, 8, 3),
        content: const StandardDraftContent(text: 'escaped', languages: ['en']),
        schedule: const DraftScheduleIntent.now(),
        media: const [],
      );
      await File(
        '${outside.path}${Platform.pathSeparator}manifest.json',
      ).writeAsString(DraftManifestCodec.encode(escapedDraft));
      await Directory(paths.accountRoot).create(recursive: true);
      await Link(paths.draftDirectory(draftId)).create(outside.path);
      final repository = FileLocalPostDraftRepository(
        documentsRoot: root.path,
        owner: owner,
      );

      await expectLater(
        repository.get(draftId),
        throwsA(
          isA<DraftRepositoryException>().having(
            (error) => error.reason,
            'reason',
            DraftRepositoryFailureReason.unavailable,
          ),
        ),
      );
    },
  );

  test(
    'rejects a linked storage ancestor before reading an outside manifest',
    () async {
      final root = await Directory.systemTemp.createTemp(
        'local-drafts-ancestor-link-',
      );
      addTearDown(() => root.delete(recursive: true));
      final owner = AccountKey('did:plc:alice');
      const draftId = '00000000-0000-4000-8000-000000000001';
      final paths = DraftStoragePaths(documentsRoot: root.path, owner: owner);
      final storageRoot = p.dirname(paths.accountRoot);
      final outside = await Directory(
        p.join(root.path, 'outside-v1'),
      ).create();
      final escapedDirectory = await Directory(
        p.join(outside.path, p.basename(paths.accountRoot), draftId),
      ).create(recursive: true);
      final escapedDraft = LocalPostDraft(
        id: draftId,
        owner: owner,
        kind: LocalPostDraftKind.standard,
        createdAt: DateTime.utc(2026, 8, 3),
        updatedAt: DateTime.utc(2026, 8, 3),
        content: const StandardDraftContent(
          text: 'outside private canary',
          languages: ['en'],
        ),
        schedule: const DraftScheduleIntent.now(),
        media: const [],
      );
      await File(
        p.join(escapedDirectory.path, 'manifest.json'),
      ).writeAsString(DraftManifestCodec.encode(escapedDraft));
      await Directory(p.dirname(storageRoot)).create(recursive: true);
      await Link(storageRoot).create(outside.path);
      final repository = FileLocalPostDraftRepository(
        documentsRoot: root.path,
        owner: owner,
      );

      await expectLater(
        repository.get(draftId),
        throwsA(
          isA<DraftRepositoryException>().having(
            (error) => error.reason,
            'reason',
            DraftRepositoryFailureReason.unavailable,
          ),
        ),
      );
    },
  );

  test(
    'reconciliation never follows a linked media directory',
    () async {
      final root = await Directory.systemTemp.createTemp(
        'local-drafts-media-link-',
      );
      addTearDown(() => root.delete(recursive: true));
      final owner = AccountKey('did:plc:alice');
      const draftId = '00000000-0000-4000-8000-000000000001';
      final repository = FileLocalPostDraftRepository(
        documentsRoot: root.path,
        owner: owner,
      );
      await repository.save(
        DraftWriteRequest(
          id: draftId,
          owner: owner,
          kind: LocalPostDraftKind.standard,
          content: const StandardDraftContent(
            text: 'work',
            languages: ['en'],
          ),
          schedule: const DraftScheduleIntent.now(),
          orderedMedia: const [],
        ),
      );
      final paths = DraftStoragePaths(documentsRoot: root.path, owner: owner);
      final mediaDirectory = Directory(paths.mediaDirectory(draftId));
      await mediaDirectory.delete();
      final outside = await Directory(
        p.join(root.path, 'outside-media'),
      ).create();
      final canary = File(p.join(outside.path, 'outside-private-canary.bin'));
      await canary.writeAsBytes([1, 2, 3]);
      await Link(mediaDirectory.path).create(outside.path);

      final drafts = await repository.list();

      expect(drafts, hasLength(1));
      expect(drafts.single.id, draftId);
      // Test assertions intentionally inspect the real filesystem boundary.
      // ignore: avoid_slow_async_io
      expect(await canary.exists(), isTrue);
      expect(await canary.readAsBytes(), [1, 2, 3]);
    },
  );

  test('read and reconciliation never follow a linked media file', () async {
    final root = await Directory.systemTemp.createTemp(
      'local-drafts-media-file-link-',
    );
    addTearDown(() => root.delete(recursive: true));
    final owner = AccountKey('did:plc:alice');
    const draftId = '00000000-0000-4000-8000-000000000001';
    final repository = FileLocalPostDraftRepository(
      documentsRoot: root.path,
      owner: owner,
    );
    final saved = await repository.save(
      DraftWriteRequest(
        id: draftId,
        owner: owner,
        kind: LocalPostDraftKind.standard,
        content: const StandardDraftContent(text: 'work', languages: ['en']),
        schedule: const DraftScheduleIntent.now(),
        orderedMedia: [
          PreparedDraftMedia(
            mediaId: '00000000-0000-4000-8000-000000000002',
            displayFileName: 'image.jpg',
            mimeType: 'image/jpeg',
            bytes: Uint8List.fromList([1, 2, 3]),
            width: 1,
            height: 1,
            altText: '',
          ),
        ],
      ),
    );
    final paths = DraftStoragePaths(documentsRoot: root.path, owner: owner);
    final mediaPath = paths.mediaFilePath(
      draftId,
      saved.media.single.storageFileName,
    );
    await File(mediaPath).delete();
    final canary = File(p.join(root.path, 'outside-media-canary.bin'));
    await canary.writeAsBytes([9, 8, 7]);
    await Link(mediaPath).create(canary.path);

    await expectLater(
      repository.readMedia(draftId, saved.media.single.mediaId),
      throwsA(isA<DraftRepositoryException>()),
    );
    final listed = await repository.list();

    expect(listed.single.availability, LocalPostDraftAvailability.unavailable);
    expect(await canary.readAsBytes(), [9, 8, 7]);
    // Test assertions intentionally inspect the real filesystem boundary.
    // ignore: avoid_slow_async_io
    expect(await Link(mediaPath).exists(), isTrue);
  });

  test(
    'delete refuses a linked draft directory and preserves its target',
    () async {
      final root = await Directory.systemTemp.createTemp(
        'local-drafts-delete-link-',
      );
      addTearDown(() => root.delete(recursive: true));
      final owner = AccountKey('did:plc:alice');
      const draftId = '00000000-0000-4000-8000-000000000001';
      final paths = DraftStoragePaths(documentsRoot: root.path, owner: owner);
      await Directory(paths.accountRoot).create(recursive: true);
      final outside = await Directory(
        p.join(root.path, 'outside-delete-target'),
      ).create();
      final canary = File(p.join(outside.path, 'outside-private-canary.bin'));
      await canary.writeAsBytes([4, 5, 6]);
      await Link(paths.draftDirectory(draftId)).create(outside.path);
      final repository = FileLocalPostDraftRepository(
        documentsRoot: root.path,
        owner: owner,
      );

      await expectLater(
        repository.delete(draftId),
        throwsA(
          isA<DraftRepositoryException>().having(
            (error) => error.reason,
            'reason',
            DraftRepositoryFailureReason.storageUnavailable,
          ),
        ),
      );

      // Test assertions intentionally inspect the real filesystem boundary.
      // ignore: avoid_slow_async_io
      expect(await canary.exists(), isTrue);
      expect(await canary.readAsBytes(), [4, 5, 6]);
    },
  );
}
