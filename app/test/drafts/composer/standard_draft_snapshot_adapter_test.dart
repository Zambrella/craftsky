import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/composer/standard_draft_snapshot_adapter.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('builds an incomplete standard snapshot with reusable stored media', () {
    final request = const StandardDraftSnapshotAdapter().toWriteRequest(
      id: '00000000-0000-4000-8000-000000000001',
      owner: AccountKey('did:plc:alice'),
      existingRevision: 4,
      existingCreatedAt: DateTime.utc(2026, 8),
      text: '',
      languages: const ['en', 'cy'],
      schedule: DraftScheduleIntent.later(
        scheduledAtUtc: DateTime.utc(2026, 8, 4, 18),
        savedOffsetMinutes: 60,
      ),
      images: [
        ComposerImageDraft(
          id: '00000000-0000-4000-8000-000000000002',
          fileName: 'image.png',
          mimeType: 'image/png',
          altText: 'Updated alt',
          previewBytes: Uint8List.fromList([1, 2, 3]),
          phase: ImageReady(
            bytes: Uint8List.fromList([1, 2, 3]),
            mimeType: 'image/png',
            width: 2,
            height: 3,
            sha256:
                '039058c6f2c0cb492c533b0a4d14ef77'
                'cc0f78abccced5287d84a1a2011cfb81',
            storedOrigin: const StoredDraftMediaOrigin(
              mediaId: '00000000-0000-4000-8000-000000000002',
              storageRevision: '00000000-0000-4000-8000-000000000012',
              sha256:
                  '039058c6f2c0cb492c533b0a4d14ef77'
                  'cc0f78abccced5287d84a1a2011cfb81',
              byteLength: 3,
            ),
          ),
        ),
      ],
    );

    expect(request.expectedRevision, 4);
    expect(request.createdAt, DateTime.utc(2026, 8));
    expect((request.content as StandardDraftContent).text, isEmpty);
    expect((request.content as StandardDraftContent).languages, ['en', 'cy']);
    expect(request.orderedMedia.single, isA<ExistingStoredMedia>());
    expect(request.orderedMedia.single.altText, 'Updated alt');
  });
}
