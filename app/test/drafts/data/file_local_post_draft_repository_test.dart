import 'dart:io';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/data/draft_file_store.dart';
import 'package:craftsky_app/drafts/data/file_local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:flutter_test/flutter_test.dart';

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
}
