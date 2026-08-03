import 'dart:io';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/data/draft_file_store.dart';
import 'package:craftsky_app/drafts/data/draft_storage_paths.dart';
import 'package:craftsky_app/drafts/data/file_local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'keeps damaged drafts visible and reconciles unreferenced files',
    () async {
      final root = await Directory.systemTemp.createTemp('draft-recovery-');
      addTearDown(() => root.delete(recursive: true));
      final owner = AccountKey('did:plc:alice');
      final files = IoDraftFileStore();
      final paths = DraftStoragePaths(documentsRoot: root.path, owner: owner);
      var generated = 40;
      String nextId() =>
          '00000000-0000-4000-8000-'
          '${(generated++).toString().padLeft(12, '0')}';
      final repository = FileLocalPostDraftRepository(
        documentsRoot: root.path,
        owner: owner,
        fileStore: files,
        clock: () => DateTime.utc(2026, 8, 3),
        nextId: nextId,
      );
      const healthyId = '00000000-0000-4000-8000-000000000001';
      const damagedId = '00000000-0000-4000-8000-000000000002';
      const corruptId = '00000000-0000-4000-8000-000000000003';
      final healthy = await repository.save(
        _request(owner, healthyId, 'Healthy'),
      );
      final damaged = await repository.save(
        _request(owner, damagedId, 'Preserved private text'),
      );
      await files.deleteFile(
        paths.mediaFilePath(damagedId, damaged.media.single.storageFileName),
      );
      await files.ensureDirectory(paths.draftDirectory(corruptId));
      await files.writeBytesFlushed(
        paths.manifestPath(corruptId),
        Uint8List.fromList('{broken'.codeUnits),
      );
      final orphanPath = paths.mediaFilePath(healthyId, 'orphan.jpg');
      await files.writeBytesFlushed(orphanPath, Uint8List.fromList([9]));
      final pendingPath = paths.pendingManifestPath(healthyId, nextId());
      await files.writeBytesFlushed(pendingPath, Uint8List.fromList([9]));

      final drafts = await repository.list();

      expect(drafts, hasLength(3));
      expect(
        drafts.singleWhere((draft) => draft.id == healthyId).availability,
        LocalPostDraftAvailability.available,
      );
      final damagedDraft = drafts.singleWhere((draft) => draft.id == damagedId);
      expect(damagedDraft.availability, LocalPostDraftAvailability.unavailable);
      expect(
        (damagedDraft.content as StandardDraftContent).text,
        'Preserved private text',
      );
      expect(
        damagedDraft.media.single.availability,
        DraftMediaAvailability.unavailable,
      );
      expect(damagedDraft.media.single.altText, 'Private alt');
      expect(
        drafts.singleWhere((draft) => draft.id == corruptId).availability,
        LocalPostDraftAvailability.unavailable,
      );
      expect(await files.fileExists(orphanPath), isFalse);
      expect(await files.fileExists(pendingPath), isFalse);
      expect(
        await repository.readMedia(healthyId, healthy.media.single.mediaId),
        [1, 2, 3],
      );
    },
  );
}

DraftWriteRequest _request(AccountKey owner, String id, String text) =>
    DraftWriteRequest(
      id: id,
      owner: owner,
      kind: LocalPostDraftKind.standard,
      content: StandardDraftContent(text: text, languages: const ['en']),
      schedule: const DraftScheduleIntent.now(),
      orderedMedia: [
        PreparedDraftMedia(
          mediaId: id.replaceRange(id.length - 1, null, '9'),
          displayFileName: 'image.png',
          mimeType: 'image/png',
          bytes: Uint8List.fromList([1, 2, 3]),
          width: 1,
          height: 1,
          altText: 'Private alt',
        ),
      ],
    );
