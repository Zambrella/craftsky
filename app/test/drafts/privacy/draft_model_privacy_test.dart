import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/models/draft_manifest_codec.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('draft model and manifest errors redact private values', () {
    const canaries = [
      'private-body-canary',
      'private-alt-canary',
      'private-file-canary',
      'did:plc:private-owner-canary',
    ];
    const media = DraftMediaDescriptor(
      mediaId: '00000000-0000-4000-8000-000000000002',
      storageRevision: '00000000-0000-4000-8000-000000000003',
      storageFileName: 'private-file-canary.jpg',
      displayFileName: 'private-file-canary.jpg',
      mimeType: 'image/jpeg',
      byteLength: 12,
      sha256:
          '0123456789abcdef0123456789abcdef'
          '0123456789abcdef0123456789abcdef',
      width: 10,
      height: 10,
      altText: 'private-alt-canary',
      order: 0,
    );
    const content = StandardDraftContent(
      text: 'private-body-canary',
      languages: ['en'],
    );
    final draft = LocalPostDraft(
      id: '00000000-0000-4000-8000-000000000001',
      owner: AccountKey('did:plc:private-owner-canary'),
      kind: LocalPostDraftKind.standard,
      createdAt: DateTime.utc(2026),
      updatedAt: DateTime.utc(2026),
      content: content,
      schedule: const DraftScheduleIntent.now(),
      media: const [media],
    );
    const error = DraftManifestException(
      DraftManifestFailureReason.corrupt,
    );

    final diagnostics = [
      draft,
      draft.owner,
      content,
      media,
      draft.schedule,
      error,
    ].join('\n');

    for (final canary in canaries) {
      expect(diagnostics, isNot(contains(canary)));
    }
  });
}
