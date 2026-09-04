import 'dart:io';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/data/file_local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  for (final kind in LocalPostDraftKind.values) {
    test('IR-010 preserves unknown duration in a ${kind.name} draft', () async {
      final root = await Directory.systemTemp.createTemp('unknown-duration-');
      addTearDown(() => root.delete(recursive: true));
      final owner = AccountKey('did:plc:alice');
      final repository = FileLocalPostDraftRepository(
        documentsRoot: root.path,
        owner: owner,
        nextId: () => '00000000-0000-4000-8000-000000000010',
      );

      final saved = await repository.save(
        DraftWriteRequest(
          id: kind == LocalPostDraftKind.standard
              ? '00000000-0000-4000-8000-000000000001'
              : '00000000-0000-4000-8000-000000000002',
          owner: owner,
          kind: kind,
          content: kind == LocalPostDraftKind.standard
              ? const StandardDraftContent(text: 'Unknown', languages: ['en'])
              : const ProjectDraftContent(
                  body: 'Unknown',
                  languages: ['en'],
                  knownProjectFieldValues: {},
                ),
          schedule: const DraftScheduleIntent.now(),
          orderedMedia: const [],
          video: PreparedDraftVideo(
            displayFileName: 'clip.mp4',
            mimeType: 'video/mp4',
            byteLength: 2,
            openSource: () => Stream.value([1, 2]),
            width: 16,
            height: 9,
            duration: null,
            altText: '',
            posterMimeType: 'image/jpeg',
            posterBytes: Uint8List.fromList([3]),
          ),
        ),
      );

      expect(saved.video?.duration, isNull);

      final reopened = FileLocalPostDraftRepository(
        documentsRoot: root.path,
        owner: owner,
      );
      expect((await reopened.get(saved.id)).video?.duration, isNull);
    });
  }

  test(
    'IT-011 streams source and poster into a restorable draft',
    () async {
      final root = await Directory.systemTemp.createTemp('video-draft-repo-');
      addTearDown(() => root.delete(recursive: true));
      final owner = AccountKey('did:plc:alice');
      var generated = 10;
      String nextId() =>
          '00000000-0000-4000-8000-'
          '${(generated++).toString().padLeft(12, '0')}';
      final repository = FileLocalPostDraftRepository(
        documentsRoot: root.path,
        owner: owner,
        nextId: nextId,
        videoSourceQuotaBytes: 8,
      );
      var sourceListenCount = 0;

      final saved = await repository.save(
        DraftWriteRequest(
          id: '00000000-0000-4000-8000-000000000001',
          owner: owner,
          kind: LocalPostDraftKind.standard,
          content: const StandardDraftContent(
            text: 'Video work',
            languages: ['en'],
          ),
          schedule: const DraftScheduleIntent.now(),
          orderedMedia: const [],
          video: PreparedDraftVideo(
            displayFileName: 'clip.mp4',
            mimeType: 'video/mp4',
            byteLength: 6,
            openSource: () {
              sourceListenCount++;
              return Stream.fromIterable(const [
                <int>[1, 2],
                <int>[3, 4],
                <int>[5, 6],
              ]);
            },
            width: 1080,
            height: 1920,
            duration: const Duration(seconds: 12),
            altText: 'Spinning yarn',
            posterMimeType: 'image/jpeg',
            posterBytes: Uint8List.fromList([7, 8]),
          ),
        ),
      );

      expect(sourceListenCount, 1);
      expect(saved.video, isNotNull);
      expect(saved.video!.sourceByteLength, 6);
      expect(saved.video!.posterByteLength, 2);

      final restarted = FileLocalPostDraftRepository(
        documentsRoot: root.path,
        owner: owner,
        nextId: nextId,
        videoSourceQuotaBytes: 8,
      );
      final restored = await restarted.get(saved.id);

      expect(restored.video!.altText, 'Spinning yarn');
      expect(restored.video!.duration, const Duration(seconds: 12));
      expect(await restarted.openVideoSource(saved.id).toList(), [
        [1, 2, 3, 4, 5, 6],
      ]);
      expect(await restarted.readVideoPoster(saved.id), [7, 8]);

      await expectLater(
        restarted.save(
          DraftWriteRequest(
            id: '00000000-0000-4000-8000-000000000002',
            owner: owner,
            kind: LocalPostDraftKind.project,
            content: const ProjectDraftContent(
              body: 'Too much',
              languages: ['en'],
              knownProjectFieldValues: {},
            ),
            schedule: const DraftScheduleIntent.now(),
            orderedMedia: const [],
            video: PreparedDraftVideo(
              displayFileName: 'other.mp4',
              mimeType: 'video/mp4',
              byteLength: 3,
              openSource: () => Stream.value([9, 9, 9]),
              width: 16,
              height: 9,
              duration: const Duration(seconds: 1),
              altText: '',
              posterMimeType: 'image/jpeg',
              posterBytes: Uint8List.fromList([1]),
            ),
          ),
        ),
        throwsA(
          isA<DraftRepositoryException>().having(
            (error) => error.reason,
            'reason',
            DraftRepositoryFailureReason.quotaExceeded,
          ),
        ),
      );
      expect((await restarted.get(saved.id)).video, isNotNull);

      await restarted.delete(saved.id);
      await expectLater(
        restarted.openVideoSource(saved.id).toList(),
        throwsA(isA<DraftRepositoryException>()),
      );
    },
  );
}
