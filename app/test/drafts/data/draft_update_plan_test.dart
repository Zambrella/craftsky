import 'dart:typed_data';

import 'package:craftsky_app/drafts/data/draft_update_plan.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('reuses unchanged media and writes replacements immutably', () {
    const retained = DraftMediaDescriptor(
      mediaId: '00000000-0000-4000-8000-000000000001',
      storageRevision: '00000000-0000-4000-8000-000000000011',
      storageFileName: 'retained.jpg',
      displayFileName: 'retained.jpg',
      mimeType: 'image/jpeg',
      byteLength: 3,
      sha256:
          '039058c6f2c0cb492c533b0a4d14ef77'
          'cc0f78abccced5287d84a1a2011cfb81',
      width: 20,
      height: 10,
      altText: 'old alt',
      order: 0,
    );
    const removed = DraftMediaDescriptor(
      mediaId: '00000000-0000-4000-8000-000000000002',
      storageRevision: '00000000-0000-4000-8000-000000000012',
      storageFileName: 'removed.png',
      displayFileName: 'removed.png',
      mimeType: 'image/png',
      byteLength: 4,
      sha256:
          '9f64a747e1b97f131fabb6b447296c9b'
          '6f0201e79fb3c5356e6c77e89b6a806a',
      width: 10,
      height: 10,
      altText: '',
      order: 1,
    );

    final plan = DraftUpdatePlan.build(
      currentMedia: const [retained, removed],
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
          altText: 'new alt',
        ),
        PreparedDraftMedia(
          mediaId: '00000000-0000-4000-8000-000000000003',
          displayFileName: 'new.png',
          mimeType: 'image/png',
          bytes: Uint8List.fromList([1, 2, 3, 4]),
          width: 30,
          height: 40,
          altText: 'new image',
        ),
      ],
      nextStorageRevision: () => '00000000-0000-4000-8000-000000000013',
    );

    expect(plan.mediaWrites, hasLength(1));
    expect(plan.mediaWrites.single.mediaId, endsWith('0003'));
    expect(plan.nextMedia, hasLength(2));
    expect(plan.nextMedia.first.storageFileName, retained.storageFileName);
    expect(plan.nextMedia.first.altText, 'new alt');
    expect(plan.nextMedia.first.order, 0);
    expect(plan.nextMedia.last.storageRevision, endsWith('0013'));
    expect(plan.nextMedia.last.storageFileName, endsWith('.png'));
    expect(plan.nextMedia.last.order, 1);
    expect(plan.cleanupFileNames, [removed.storageFileName]);
  });
}
