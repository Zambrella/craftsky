import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/models/draft_manifest_codec.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('round-trips every version 1 standard draft field', () {
    final draft = LocalPostDraft(
      id: '00000000-0000-4000-8000-000000000001',
      owner: AccountKey('did:plc:alice'),
      kind: LocalPostDraftKind.standard,
      createdAt: DateTime.utc(2026, 8, 1, 10),
      updatedAt: DateTime.utc(2026, 8, 2, 11, 30),
      content: const StandardDraftContent(
        text: 'A private draft',
        languages: ['en', 'cy'],
      ),
      schedule: DraftScheduleIntent.later(
        scheduledAtUtc: DateTime.utc(2026, 8, 4, 18),
        savedOffsetMinutes: 60,
      ),
      media: const [
        DraftMediaDescriptor(
          mediaId: '00000000-0000-4000-8000-000000000002',
          storageRevision: '00000000-0000-4000-8000-000000000003',
          storageFileName:
              '00000000-0000-4000-8000-000000000002-'
              '00000000-0000-4000-8000-000000000003.jpg',
          displayFileName: 'swatch.jpg',
          mimeType: 'image/jpeg',
          byteLength: 1234,
          sha256:
              '0123456789abcdef0123456789abcdef'
              '0123456789abcdef0123456789abcdef',
          width: 800,
          height: 600,
          altText: 'Blue knitted swatch',
          order: 0,
        ),
      ],
    );

    final encoded = DraftManifestCodec.encode(draft);
    final decoded = DraftManifestCodec.decode(encoded);

    expect(decoded.id, draft.id);
    expect(decoded.owner, draft.owner);
    expect(decoded.kind, LocalPostDraftKind.standard);
    expect(decoded.createdAt, draft.createdAt);
    expect(decoded.updatedAt, draft.updatedAt);
    expect(decoded.content, isA<StandardDraftContent>());
    expect((decoded.content as StandardDraftContent).text, 'A private draft');
    expect(
      (decoded.content as StandardDraftContent).languages,
      ['en', 'cy'],
    );
    expect(decoded.schedule.choice, DraftScheduleChoice.later);
    expect(decoded.schedule.scheduledAtUtc, DateTime.utc(2026, 8, 4, 18));
    expect(decoded.schedule.savedOffsetMinutes, 60);
    expect(decoded.media, hasLength(1));
    expect(decoded.media.single.mediaId, draft.media.single.mediaId);
    expect(
      decoded.media.single.storageFileName,
      draft.media.single.storageFileName,
    );
    expect(decoded.media.single.altText, 'Blue knitted swatch');
  });

  test('round-trips whitelisted incomplete project fields', () {
    final draft = LocalPostDraft(
      id: '00000000-0000-4000-8000-000000000011',
      owner: AccountKey('did:plc:alice'),
      kind: LocalPostDraftKind.project,
      createdAt: DateTime.utc(2026, 8),
      updatedAt: DateTime.utc(2026, 8, 2),
      content: const ProjectDraftContent(
        body: 'Work in progress',
        languages: ['en'],
        knownProjectFieldValues: {
          'title': 'Blue jumper',
          'materials': [
            {'text': 'Wool'},
          ],
          'unfinishedOptionalValue': null,
        },
      ),
      schedule: const DraftScheduleIntent.now(),
      media: const [],
    );

    final decoded = DraftManifestCodec.decode(DraftManifestCodec.encode(draft));

    expect(decoded.kind, LocalPostDraftKind.project);
    final content = decoded.content as ProjectDraftContent;
    expect(content.body, 'Work in progress');
    expect(content.languages, ['en']);
    expect(content.knownProjectFieldValues['title'], 'Blue jumper');
    expect(content.knownProjectFieldValues['materials'], [
      {'text': 'Wool'},
    ]);
    expect(
      content.knownProjectFieldValues,
      contains('unfinishedOptionalValue'),
    );
  });

  test('rejects a future schema with a content-free typed error', () {
    final source = jsonEncode({
      'schemaVersion': 2,
      'id': '00000000-0000-4000-8000-000000000001',
      'owner': 'did:plc:private-owner-canary',
      'privateCanary': 'do-not-report-this-content',
    });

    expect(
      () => DraftManifestCodec.decode(source),
      throwsA(
        isA<DraftManifestException>()
            .having(
              (error) => error.reason,
              'reason',
              DraftManifestFailureReason.unsupportedVersion,
            )
            .having(
              (error) => error.toString(),
              'safe output',
              'DraftManifestException(unsupportedVersion)',
            ),
      ),
    );
  });

  test('validates persisted media before returning a draft', () {
    final manifest =
        jsonDecode(
              DraftManifestCodec.encode(_emptyDraft()),
            )
            as Map<String, Object?>;
    manifest['media'] = [
      {
        'mediaId': '00000000-0000-4000-8000-000000000002',
        'storageRevision': '00000000-0000-4000-8000-000000000003',
        'storageFileName': '../private-canary.jpg',
        'displayFileName': 'swatch.jpg',
        'mimeType': 'image/jpeg',
        'byteLength': 1234,
        'sha256':
            '0123456789abcdef0123456789abcdef'
            '0123456789abcdef0123456789abcdef',
        'width': 800,
        'height': 600,
        'altText': 'private alt canary',
        'order': 0,
      },
    ];

    expect(
      () => DraftManifestCodec.decode(jsonEncode(manifest)),
      throwsA(
        isA<DraftManifestException>().having(
          (error) => error.reason,
          'reason',
          DraftManifestFailureReason.invalidMedia,
        ),
      ),
    );
  });

  test('rejects duplicate media identities in a manifest', () {
    final manifest =
        jsonDecode(
              DraftManifestCodec.encode(_emptyDraft()),
            )
            as Map<String, Object?>;
    manifest['media'] = [
      _validMediaMap(order: 0),
      _validMediaMap(order: 1),
    ];

    expect(
      () => DraftManifestCodec.decode(jsonEncode(manifest)),
      throwsA(
        isA<DraftManifestException>().having(
          (error) => error.reason,
          'reason',
          DraftManifestFailureReason.invalidMedia,
        ),
      ),
    );
  });

  test('validates media before writing a manifest', () {
    final draft = LocalPostDraft(
      id: '00000000-0000-4000-8000-000000000001',
      owner: AccountKey('did:plc:alice'),
      kind: LocalPostDraftKind.standard,
      createdAt: DateTime.utc(2026, 8),
      updatedAt: DateTime.utc(2026, 8, 2),
      content: const StandardDraftContent(text: '', languages: ['en']),
      schedule: const DraftScheduleIntent.now(),
      media: const [
        DraftMediaDescriptor(
          mediaId: '00000000-0000-4000-8000-000000000002',
          storageRevision: '00000000-0000-4000-8000-000000000003',
          storageFileName: '../outside.jpg',
          displayFileName: 'swatch.jpg',
          mimeType: 'image/jpeg',
          byteLength: 1234,
          sha256:
              '0123456789abcdef0123456789abcdef'
              '0123456789abcdef0123456789abcdef',
          width: 800,
          height: 600,
          altText: '',
          order: 0,
        ),
      ],
    );

    expect(
      () => DraftManifestCodec.encode(draft),
      throwsA(isA<DraftManifestException>()),
    );
  });
}

LocalPostDraft _emptyDraft() => LocalPostDraft(
  id: '00000000-0000-4000-8000-000000000001',
  owner: AccountKey('did:plc:alice'),
  kind: LocalPostDraftKind.standard,
  createdAt: DateTime.utc(2026, 8, 1, 10),
  updatedAt: DateTime.utc(2026, 8, 2, 11, 30),
  content: const StandardDraftContent(text: '', languages: ['en']),
  schedule: const DraftScheduleIntent.now(),
  media: const [],
);

Map<String, Object?> _validMediaMap({required int order}) => {
  'mediaId': '00000000-0000-4000-8000-000000000002',
  'storageRevision': '00000000-0000-4000-8000-000000000003',
  'storageFileName':
      '00000000-0000-4000-8000-000000000002-'
      '00000000-0000-4000-8000-000000000003.jpg',
  'displayFileName': 'swatch.jpg',
  'mimeType': 'image/jpeg',
  'byteLength': 1234,
  'sha256':
      '0123456789abcdef0123456789abcdef'
      '0123456789abcdef0123456789abcdef',
  'width': 800,
  'height': 600,
  'altText': 'Blue knitted swatch',
  'order': order,
};
