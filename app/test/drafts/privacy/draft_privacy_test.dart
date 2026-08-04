import 'dart:io';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/data/file_local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/feed/composer/composer_submission_coordinator.dart';
import 'package:craftsky_app/feed/composer/submission_screen_awake.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'IT-016 local and submission diagnostics redact privacy canaries',
    () async {
      const canaries = [
        'private-text-canary',
        'private-project-canary',
        'private-alt-canary',
        'private-owner-canary',
        'private-path-canary',
        'private-file-canary',
        'private-api-failure-canary',
      ];
      final root = await Directory.systemTemp.createTemp(
        'private-path-canary-',
      );
      addTearDown(() => root.delete(recursive: true));
      final owner = AccountKey('did:plc:private-owner-canary');
      final repository = FileLocalPostDraftRepository(
        documentsRoot: root.path,
        owner: owner,
      );
      const draftId = '00000000-0000-4000-8000-000000000001';
      final request = DraftWriteRequest(
        id: draftId,
        owner: owner,
        kind: LocalPostDraftKind.standard,
        content: const StandardDraftContent(
          text: 'private-text-canary',
          languages: ['en'],
        ),
        schedule: const DraftScheduleIntent.now(),
        orderedMedia: [
          PreparedDraftMedia(
            mediaId: '00000000-0000-4000-8000-000000000002',
            displayFileName: 'private-file-canary.jpg',
            mimeType: 'image/jpeg',
            bytes: Uint8List.fromList('private-image-bytes-canary'.codeUnits),
            width: 1,
            height: 1,
            altText: 'private-alt-canary',
          ),
        ],
      );
      final diagnostics = <Object>[];

      final saved = await repository.save(request);
      diagnostics
        ..add(repository)
        ..add(request)
        ..add(saved)
        ..addAll(await repository.list())
        ..add(await repository.get(draftId))
        ..add(
          const ProjectDraftContent(
            body: 'private-project-canary',
            languages: ['en'],
            knownProjectFieldValues: {'title': 'private-project-canary'},
          ),
        );
      await repository.readMedia(draftId, saved.media.single.mediaId);
      await repository.delete(draftId);
      try {
        await repository.get(draftId);
      } on Object catch (error) {
        diagnostics.add(error);
      }

      final coordinator = ComposerSubmissionCoordinator(
        screenAwake: const _NoopScreenAwake(),
      );
      await coordinator.run(
        presentOverlay: () async {},
        ownershipIsCurrent: () => true,
        saveOriginSnapshot: () async {},
        operation: () async {
          throw StateError('private-api-failure-canary');
        },
        didSucceed: () => false,
        deleteOriginAfterSuccess: () async {},
        onRunningChanged: ({required running}) {},
        onFailure: diagnostics.add,
      );
      await coordinator.dispose();

      final emitted = diagnostics.join('\n');
      for (final canary in canaries) {
        expect(emitted, isNot(contains(canary)));
      }
    },
  );
}

final class _NoopScreenAwake implements SubmissionScreenAwake {
  const _NoopScreenAwake();

  @override
  Future<void> enable() async {}

  @override
  Future<void> disable() async {}
}
