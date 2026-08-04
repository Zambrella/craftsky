import 'dart:io';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/data/file_local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'IT-005 sign-out retains local drafts while accounts remain isolated',
    () async {
      final root = await Directory.systemTemp.createTemp('draft-retention-');
      addTearDown(() => root.delete(recursive: true));
      final alice = AccountKey('did:plc:alice');
      final bob = AccountKey('did:plc:bob');
      final aliceRepository = FileLocalPostDraftRepository(
        documentsRoot: root.path,
        owner: alice,
      );
      const draftId = '00000000-0000-4000-8000-000000000001';
      final bytes = Uint8List.fromList([1, 2, 3, 4]);

      await aliceRepository.save(
        DraftWriteRequest(
          id: draftId,
          owner: alice,
          kind: LocalPostDraftKind.standard,
          content: const StandardDraftContent(
            text: 'Alice private draft',
            languages: ['en'],
          ),
          schedule: const DraftScheduleIntent.now(),
          orderedMedia: [
            PreparedDraftMedia(
              mediaId: '00000000-0000-4000-8000-000000000002',
              displayFileName: 'private.jpg',
              mimeType: 'image/jpeg',
              bytes: bytes,
              width: 1,
              height: 1,
              altText: 'Private image',
            ),
          ],
        ),
      );

      // Dropping all session-owned objects models sign-out: no repository
      // cleanup is invoked. A different account still receives its own root.
      final bobRepository = FileLocalPostDraftRepository(
        documentsRoot: root.path,
        owner: bob,
      );
      expect(await bobRepository.list(), isEmpty);

      final signedBackIn = FileLocalPostDraftRepository(
        documentsRoot: root.path,
        owner: alice,
      );
      final retained = await signedBackIn.get(draftId);
      expect(
        (retained.content as StandardDraftContent).text,
        'Alice private draft',
      );
      expect(
        await signedBackIn.readMedia(draftId, retained.media.single.mediaId),
        bytes,
      );

      await signedBackIn.delete(draftId);
      expect(await signedBackIn.list(), isEmpty);
      expect(await bobRepository.list(), isEmpty);
    },
  );
}
