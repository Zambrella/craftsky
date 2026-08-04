import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/composer/draft_submission_origin.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('advances the expected revision after every recovery snapshot', () {
    final origin = DraftSubmissionOrigin(_draft(revision: 1));

    // Separate calls keep the intermediate revision assertion observable.
    // ignore: cascade_invocations
    origin.acceptSnapshot(_draft(revision: 2));
    expect(origin.draft?.revision, 2);

    origin.acceptSnapshot(_draft(revision: 3));
    expect(origin.draft?.revision, 3);
  });

  test('rejects a snapshot for a different draft identity', () {
    final origin = DraftSubmissionOrigin(_draft(revision: 1));

    expect(
      () => origin.acceptSnapshot(_draft(id: _otherId, revision: 2)),
      throwsStateError,
    );
    expect(origin.draft?.revision, 1);
  });
}

LocalPostDraft _draft({required int revision, String id = _draftId}) =>
    LocalPostDraft(
      id: id,
      owner: AccountKey('did:plc:alice'),
      kind: LocalPostDraftKind.standard,
      createdAt: DateTime.utc(2026, 8, 3),
      updatedAt: DateTime.utc(2026, 8, 3),
      content: const StandardDraftContent(text: 'work', languages: ['en']),
      schedule: const DraftScheduleIntent.now(),
      media: const [],
      revision: revision,
    );

const _draftId = '00000000-0000-4000-8000-000000000001';
const _otherId = '00000000-0000-4000-8000-000000000002';
