import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/composer/draft_media_write_adapter.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';

final class StandardDraftSnapshotAdapter {
  const StandardDraftSnapshotAdapter();

  DraftWriteRequest toWriteRequest({
    required String id,
    required AccountKey owner,
    required String text,
    required List<String> languages,
    required DraftScheduleIntent schedule,
    required List<ComposerImageDraft> images,
    int? existingRevision,
    DateTime? existingCreatedAt,
  }) {
    return DraftWriteRequest(
      id: id,
      owner: owner,
      kind: LocalPostDraftKind.standard,
      createdAt: existingCreatedAt,
      expectedRevision: existingRevision,
      content: StandardDraftContent(
        text: text,
        languages: List.unmodifiable(languages),
      ),
      schedule: schedule,
      orderedMedia: draftMediaWritesFromComposer(images),
    );
  }
}
