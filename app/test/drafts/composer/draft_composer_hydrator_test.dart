import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/composer/draft_composer_hydrator.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'hydrates available bytes and preserves unavailable media metadata',
    () async {
      final draft = LocalPostDraft(
        id: '00000000-0000-4000-8000-000000000001',
        owner: AccountKey('did:plc:alice'),
        kind: LocalPostDraftKind.standard,
        createdAt: DateTime.utc(2026),
        updatedAt: DateTime.utc(2026),
        content: const StandardDraftContent(text: 'Body', languages: ['en']),
        schedule: const DraftScheduleIntent.now(),
        media: const [_available, _missing],
      );
      final repository = _MediaRepository({
        _available.mediaId: Uint8List.fromList([1, 2, 3]),
      });

      final seed = await const DraftComposerHydrator().hydrate(
        repository: repository,
        draft: draft,
      );

      expect(seed.draft, same(draft));
      expect(seed.media, hasLength(2));
      expect(seed.media.first.bytes, [1, 2, 3]);
      expect(seed.media.first.isAvailable, isTrue);
      expect(seed.media.last.bytes, isNull);
      expect(seed.media.last.isAvailable, isFalse);
      expect(seed.media.last.descriptor.altText, 'Preserved missing alt');
      expect(seed.canSubmit, isFalse);
    },
  );
}

const _available = DraftMediaDescriptor(
  mediaId: '00000000-0000-4000-8000-000000000002',
  storageRevision: '00000000-0000-4000-8000-000000000012',
  storageFileName: 'available.png',
  displayFileName: 'available.png',
  mimeType: 'image/png',
  byteLength: 3,
  sha256:
      '039058c6f2c0cb492c533b0a4d14ef77'
      'cc0f78abccced5287d84a1a2011cfb81',
  width: 1,
  height: 1,
  altText: 'Available alt',
  order: 0,
);

const _missing = DraftMediaDescriptor(
  mediaId: '00000000-0000-4000-8000-000000000003',
  storageRevision: '00000000-0000-4000-8000-000000000013',
  storageFileName: 'missing.png',
  displayFileName: 'missing.png',
  mimeType: 'image/png',
  byteLength: 3,
  sha256:
      '039058c6f2c0cb492c533b0a4d14ef77'
      'cc0f78abccced5287d84a1a2011cfb81',
  width: 1,
  height: 1,
  altText: 'Preserved missing alt',
  order: 1,
  availability: DraftMediaAvailability.unavailable,
);

final class _MediaRepository implements LocalPostDraftRepository {
  _MediaRepository(this.bytes);

  final Map<String, Uint8List> bytes;

  @override
  Future<Uint8List> readMedia(String draftId, String mediaId) async {
    final result = bytes[mediaId];
    if (result == null) {
      throw const DraftRepositoryException(
        DraftRepositoryFailureReason.unavailable,
      );
    }
    return result;
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}
